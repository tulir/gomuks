// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package matrix

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"errors"

	sync "github.com/sasha-s/go-deadlock"
	bolt "go.etcd.io/bbolt"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/gomuks/matrix/rooms"
)

type HistoryManager struct {
	sync.Mutex

	db *bolt.DB

	historyEndPtr map[*rooms.Room]uint64
}

var bucketRoomStreams = []byte("room_streams")
var bucketRoomEventIDs = []byte("room_event_ids")
var bucketStreamPointers = []byte("room_stream_pointers")

const halfUint64 = ^uint64(0) >> 1

func NewHistoryManager(dbPath string) (*HistoryManager, error) {
	hm := &HistoryManager{
		historyEndPtr: make(map[*rooms.Room]uint64),
	}
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout:      1,
		NoGrowSync:   false,
		FreelistType: bolt.FreelistArrayType,
	})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(bucketRoomStreams)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(bucketRoomEventIDs)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(bucketStreamPointers)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	hm.db = db
	return hm, nil
}

func (hm *HistoryManager) Close() error {
	return hm.db.Close()
}

var (
	EventNotFoundError = errors.New("event not found")
	RoomNotFoundError  = errors.New("room not found")
)

func (hm *HistoryManager) getStreamIndex(tx *bolt.Tx, roomID []byte, eventID []byte) (*bolt.Bucket, []byte, error) {
	eventIDs := tx.Bucket(bucketRoomEventIDs).Bucket(roomID)
	if eventIDs == nil {
		return nil, nil, RoomNotFoundError
	}
	index := eventIDs.Get(eventID)
	if index == nil {
		return nil, nil, EventNotFoundError
	}
	stream := tx.Bucket(bucketRoomStreams).Bucket(roomID)
	return stream, index, nil
}

func (hm *HistoryManager) getEvent(tx *bolt.Tx, stream *bolt.Bucket, index []byte) (*muksevt.Event, error) {
	eventData := stream.Get(index)
	if eventData == nil || len(eventData) == 0 {
		return nil, EventNotFoundError
	}
	return unmarshalEvent(eventData)
}

func (hm *HistoryManager) Get(room *rooms.Room, eventID id.EventID) (evt *muksevt.Event, err error) {
	err = hm.db.View(func(tx *bolt.Tx) error {
		if stream, index, err := hm.getStreamIndex(tx, []byte(room.ID), []byte(eventID)); err != nil {
			return err
		} else if evt, err = hm.getEvent(tx, stream, index); err != nil {
			return err
		}
		return nil
	})
	return
}

func (hm *HistoryManager) Update(room *rooms.Room, eventID id.EventID, update func(evt *muksevt.Event) error) error {
	return hm.db.Update(func(tx *bolt.Tx) error {
		if stream, index, err := hm.getStreamIndex(tx, []byte(room.ID), []byte(eventID)); err != nil {
			return err
		} else if evt, err := hm.getEvent(tx, stream, index); err != nil {
			return err
		} else if err = update(evt); err != nil {
			return err
		} else if eventData, err := marshalEvent(evt); err != nil {
			return err
		} else if err := stream.Put(index, eventData); err != nil {
			return err
		}
		return nil
	})
}

func (hm *HistoryManager) Append(room *rooms.Room, events []*event.Event) ([]*muksevt.Event, error) {
	muksEvts, _, err := hm.store(room, events, true)
	return muksEvts, err
}

func (hm *HistoryManager) Prepend(room *rooms.Room, events []*event.Event) ([]*muksevt.Event, uint64, error) {
	return hm.store(room, events, false)
}

func (hm *HistoryManager) store(room *rooms.Room, events []*event.Event, append bool) (newEvents []*muksevt.Event, newPtrStart uint64, err error) {
	hm.Lock()
	defer hm.Unlock()
	newEvents = make([]*muksevt.Event, len(events))
	err = hm.db.Update(func(tx *bolt.Tx) error {
		streamPointers := tx.Bucket(bucketStreamPointers)
		rid := []byte(room.ID)
		stream, err := tx.Bucket(bucketRoomStreams).CreateBucketIfNotExists(rid)
		if err != nil {
			return err
		}
		eventIDs, err := tx.Bucket(bucketRoomEventIDs).CreateBucketIfNotExists(rid)
		if err != nil {
			return err
		}
		if stream.Sequence() < halfUint64 {
			// The sequence counter (i.e. the future) the part after 2^63, i.e. the second half of uint64
			// We set it to -1 because NextSequence will increment it by one.
			err = stream.SetSequence(halfUint64 - 1)
			if err != nil {
				return err
			}
		}
		if append {
			ptrStart, err := stream.NextSequence()
			if err != nil {
				return err
			}
			for i, evt := range events {
				newEvents[i] = muksevt.Wrap(evt)
				if err := put(stream, eventIDs, newEvents[i], ptrStart+uint64(i)); err != nil {
					return err
				}
			}
			err = stream.SetSequence(ptrStart + uint64(len(events)) - 1)
			if err != nil {
				return err
			}
		} else {
			ptrStart, ok := hm.historyEndPtr[room]
			if !ok {
				ptrStartRaw := streamPointers.Get(rid)
				if ptrStartRaw != nil {
					ptrStart = btoi(ptrStartRaw)
				} else {
					ptrStart = halfUint64 - 1
				}
			}
			eventCount := uint64(len(events))
			for i, evt := range events {
				newEvents[i] = muksevt.Wrap(evt)
				if err := put(stream, eventIDs, newEvents[i], -ptrStart-uint64(i)); err != nil {
					return err
				}
			}
			hm.historyEndPtr[room] = ptrStart + eventCount
			// TODO this is not the correct value for newPtrStart, figure out what the f*ck is going on here
			newPtrStart = ptrStart + eventCount
			err := streamPointers.Put(rid, itob(ptrStart+eventCount))
			if err != nil {
				return err
			}
		}

		return nil
	})
	return
}

func (hm *HistoryManager) Load(room *rooms.Room, num int, ptrStart uint64) (events []*muksevt.Event, newPtrStart uint64, err error) {
	hm.Lock()
	defer hm.Unlock()
	err = hm.db.View(func(tx *bolt.Tx) error {
		stream := tx.Bucket(bucketRoomStreams).Bucket([]byte(room.ID))
		if stream == nil {
			return nil
		}
		if ptrStart == 0 {
			ptrStart = stream.Sequence() + 1
		}
		c := stream.Cursor()
		k, v := c.Seek(itob(ptrStart - uint64(num)))
		ptrStartFound := btoi(k)
		if k == nil || ptrStartFound >= ptrStart {
			return nil
		}
		newPtrStart = ptrStartFound
		for ; k != nil && btoi(k) < ptrStart; k, v = c.Next() {
			evt, parseError := unmarshalEvent(v)
			if parseError != nil {
				return parseError
			}
			events = append(events, evt)
		}
		return nil
	})
	// Reverse array because we read/append the history in reverse order.
	i := 0
	j := len(events) - 1
	for i < j {
		events[i], events[j] = events[j], events[i]
		i++
		j--
	}
	return
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func btoi(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func stripRaw(evt *muksevt.Event) {
	evtCopy := *evt.Event
	evtCopy.Content = event.Content{
		Parsed: evt.Content.Parsed,
	}
	evt.Event = &evtCopy
}

func marshalEvent(evt *muksevt.Event) ([]byte, error) {
	stripRaw(evt)
	var buf bytes.Buffer
	enc, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err := gob.NewEncoder(enc).Encode(evt); err != nil {
		_ = enc.Close()
		return nil, err
	} else if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unmarshalEvent(data []byte) (*muksevt.Event, error) {
	evt := &muksevt.Event{}
	if cmpReader, err := gzip.NewReader(bytes.NewReader(data)); err != nil {
		return nil, err
	} else if err := gob.NewDecoder(cmpReader).Decode(evt); err != nil {
		_ = cmpReader.Close()
		return nil, err
	} else if err := cmpReader.Close(); err != nil {
		return nil, err
	}
	return evt, nil
}

func put(streams, eventIDs *bolt.Bucket, evt *muksevt.Event, key uint64) error {
	data, err := marshalEvent(evt)
	if err != nil {
		return err
	}
	keyBytes := itob(key)
	if err = streams.Put(keyBytes, data); err != nil {
		return err
	}
	if err = eventIDs.Put([]byte(evt.ID), keyBytes); err != nil {
		return err
	}
	return nil
}
