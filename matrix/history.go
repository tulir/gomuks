// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
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
	"encoding/binary"
	"encoding/gob"
	"sync"

	bolt "go.etcd.io/bbolt"

	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/mautrix"
)

type HistoryManager struct {
	sync.Mutex

	db *bolt.DB

	historyEndPtr  map[*rooms.Room]uint64
	historyLoadPtr map[*rooms.Room]uint64
}

var bucketRoomStreams = []byte("room_streams")
var bucketRoomEventIDs = []byte("room_event_ids")
var bucketStreamPointers = []byte("room_stream_pointers")

const halfUint64 = ^uint64(0) >> 1

func NewHistoryManager(dbPath string) (*HistoryManager, error) {
	hm := &HistoryManager{
		historyEndPtr:  make(map[*rooms.Room]uint64),
		historyLoadPtr: make(map[*rooms.Room]uint64),
	}
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout: 1,
		NoGrowSync: false,
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

func (hm *HistoryManager) Get(room *rooms.Room, eventID string) (event *mautrix.Event, err error) {
	err = hm.db.View(func(tx *bolt.Tx) error {
		rid := []byte(room.ID)
		eventIDs := tx.Bucket(bucketRoomEventIDs).Bucket(rid)
		if eventIDs == nil {
			return nil
		}
		streamIndex := eventIDs.Get([]byte(eventID))
		if streamIndex == nil {
			return nil
		}
		stream := tx.Bucket(bucketRoomStreams).Bucket(rid)
		eventData := stream.Get(streamIndex)
		var umErr error
		event, umErr = unmarshalEvent(eventData)
		return umErr
	})
	return
}

func (hm *HistoryManager) Append(room *rooms.Room, events []*mautrix.Event) error {
	return hm.store(room, events, true)
}

func (hm *HistoryManager) Prepend(room *rooms.Room, events []*mautrix.Event) error {
	return hm.store(room, events, false)
}

func (hm *HistoryManager) store(room *rooms.Room, events []*mautrix.Event, append bool) error {
	hm.Lock()
	defer hm.Unlock()
	err := hm.db.Update(func(tx *bolt.Tx) error {
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
			for i, event := range events {
				if err := put(stream, eventIDs, event, ptrStart+uint64(i)); err != nil {
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
			for i, event := range events {
				if err := put(stream, eventIDs, event, -ptrStart-uint64(i)); err != nil {
					return err
				}
			}
			hm.historyEndPtr[room] = ptrStart + eventCount
			err := streamPointers.Put(rid, itob(ptrStart+eventCount))
			if err != nil {
				return err
			}
		}

		return nil
	})
	return err
}

func (hm *HistoryManager) Load(room *rooms.Room, num int) (events []*mautrix.Event, err error) {
	hm.Lock()
	defer hm.Unlock()
	err = hm.db.View(func(tx *bolt.Tx) error {
		rid := []byte(room.ID)
		stream := tx.Bucket(bucketRoomStreams).Bucket(rid)
		if stream == nil {
			return nil
		}
		ptrStart, ok := hm.historyLoadPtr[room]
		if !ok {
			ptrStart = stream.Sequence()
		}
		c := stream.Cursor()
		k, v := c.Seek(itob(ptrStart - uint64(num)))
		ptrStartFound := btoi(k)
		if k == nil || ptrStartFound >= ptrStart {
			return nil
		}
		hm.historyLoadPtr[room] = ptrStartFound - 1
		for ; k != nil && btoi(k) < ptrStart; k, v = c.Next() {
			event, parseError := unmarshalEvent(v)
			if parseError != nil {
				return parseError
			}
			events = append(events, event)
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

func marshalEvent(event *mautrix.Event) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(event)
	return buf.Bytes(), err
}

func unmarshalEvent(data []byte) (*mautrix.Event, error) {
	event := &mautrix.Event{}
	return event, gob.NewDecoder(bytes.NewReader(data)).Decode(event)
}

func put(streams, eventIDs *bolt.Bucket, event *mautrix.Event, key uint64) error {
	data, err := marshalEvent(event)
	if err != nil {
		return err
	}
	keyBytes := itob(key)
	if err = streams.Put(keyBytes, data); err != nil {
		return err
	}
	if err = eventIDs.Put([]byte(event.ID), keyBytes); err != nil {
		return err
	}
	return nil
}
