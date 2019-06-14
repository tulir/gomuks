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

package rooms

import (
	"compress/gzip"
	"encoding/gob"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	sync "github.com/sasha-s/go-deadlock"

	"maunium.net/go/gomuks/debug"
)

// RoomCache contains room state info in a hashmap and linked list.
type RoomCache struct {
	sync.Mutex

	listPath  string
	directory string
	maxSize   int
	maxAge    int64
	getOwner  func() string

	Map  map[string]*Room
	head *Room
	tail *Room
	size int
}

func NewRoomCache(listPath, directory string, maxSize int, maxAge int64, getOwner func() string) *RoomCache {
	return &RoomCache{
		listPath:  listPath,
		directory: directory,
		maxSize:   maxSize,
		maxAge:    maxAge,
		getOwner:  getOwner,

		Map: make(map[string]*Room),
	}
}

func (cache *RoomCache) LoadList() error {
	cache.Lock()
	defer cache.Unlock()

	// Open room list file
	file, err := os.OpenFile(cache.listPath, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to open room list file for reading")
	}
	defer debugPrintError(file.Close, "Failed to close room list file after reading")

	// Open gzip reader for room list file
	cmpReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Wrap(err, "failed to read gzip room list")
	}
	defer debugPrintError(cmpReader.Close, "Failed to close room list gzip reader")

	// Open gob decoder for gzip reader
	dec := gob.NewDecoder(cmpReader)
	// Read number of items in list
	var size int
	err = dec.Decode(&size)
	if err != nil {
		return errors.Wrap(err, "failed to read size of room list")
	}

	// Read list
	cache.Map = make(map[string]*Room, size)
	for i := 0; i < size; i++ {
		room := &Room{}
		err = dec.Decode(room)
		if err != nil {
			debug.Printf("Failed to decode %dth room list entry: %v", i+1, err)
			continue
		}
		room.path = cache.roomPath(room.ID)
		room.cache = cache
		cache.Map[room.ID] = room
	}
	return nil
}

func (cache *RoomCache) SaveLoadedRooms() {
	cache.Lock()
	defer cache.Unlock()
	cache.clean()
	for node := cache.head; node != nil; node = node.prev {
		node.Save()
	}
}

func (cache *RoomCache) SaveList() error {
	cache.Lock()
	defer cache.Unlock()

	debug.Print("Saving room list...")
	// Open room list file
	file, err := os.OpenFile(cache.listPath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to open room list file for writing")
	}
	defer debugPrintError(file.Close, "Failed to close room list file after writing")

	// Open gzip writer for room list file
	cmpWriter := gzip.NewWriter(file)
	defer debugPrintError(cmpWriter.Close, "Failed to close room list gzip writer")

	// Open gob encoder for gzip writer
	enc := gob.NewEncoder(cmpWriter)
	// Write number of items in list
	err = enc.Encode(len(cache.Map))
	if err != nil {
		return errors.Wrap(err, "failed to write size of room list")
	}

	// Write list
	for _, node := range cache.Map {
		err = enc.Encode(node)
		if err != nil {
			debug.Printf("Failed to encode room list entry of %s: %v", node.ID, err)
		}
	}
	debug.Print("Room list saved to", cache.listPath, len(cache.Map), cache.size)
	return nil
}

func (cache *RoomCache) Touch(roomID string) {
	cache.Lock()
	node, ok := cache.Map[roomID]
	if !ok || node == nil {
		cache.Unlock()
		return
	}
	cache.touch(node)
	cache.Unlock()
}

func (cache *RoomCache) TouchNode(node *Room) {
	cache.Lock()
	cache.touch(node)
	cache.Unlock()
}

func (cache *RoomCache) touch(node *Room) {
	if node == cache.head {
		return
	}
	debug.Print("Touching", node.ID)
	cache.llPop(node)
	cache.llPush(node)
	node.touch = time.Now().Unix()
}

func (cache *RoomCache) Get(roomID string) *Room {
	cache.Lock()
	node := cache.get(roomID)
	cache.Unlock()
	return node
}

func (cache *RoomCache) GetOrCreate(roomID string) *Room {
	cache.Lock()
	node := cache.get(roomID)
	if node == nil {
		node = cache.newRoom(roomID)
		cache.llPush(node)
	}
	cache.Unlock()
	return node
}

func (cache *RoomCache) get(roomID string) *Room {
	node, ok := cache.Map[roomID]
	if ok && node != nil && node.Loaded() {
		cache.touch(node)
		return node
	}
	return nil
}
func (cache *RoomCache) Put(room *Room) {
	cache.Lock()
	node := cache.get(room.ID)
	if node != nil {
		cache.touch(node)
	} else {
		cache.Map[room.ID] = room
		if room.Loaded() {
			cache.llPush(room)
		}
		node = room
	}
	cache.Unlock()
	node.Save()
}

func (cache *RoomCache) roomPath(roomID string) string {
	return filepath.Join(cache.directory, roomID+".gob.gz")
}

func (cache *RoomCache) Load(roomID string) *Room {
	cache.Lock()
	defer cache.Unlock()
	node, ok := cache.Map[roomID]
	if ok {
		return node
	}

	node = NewRoom(roomID, cache)
	node.Load()
	return node
}

func (cache *RoomCache) llPop(node *Room) {
	if node.prev == nil && node.next == nil {
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node == cache.tail {
		cache.tail = node.next
	}
	if node == cache.head {
		cache.head = node.prev
	}
	node.next = nil
	node.prev = nil
	cache.size--
}

func (cache *RoomCache) llPush(node *Room) {
	if node.next != nil || node.prev != nil {
		debug.PrintStack()
		debug.Print("Tried to llPush node that is already in stack")
		return
	}
	if node == cache.head {
		return
	}
	if cache.head != nil {
		cache.head.next = node
	}
	node.prev = cache.head
	node.next = nil
	cache.head = node
	if cache.tail == nil {
		cache.tail = node
	}
	cache.size++
	cache.clean()
}

func (cache *RoomCache) clean() {
	origSize := cache.size
	maxTS := time.Now().Unix() - cache.maxAge
	for cache.size > cache.maxSize {
		if cache.tail.touch > maxTS {
			break
		}
		cache.tail.Unload()
		cache.llPop(cache.tail)
	}
	if cleaned := origSize - cache.size; cleaned > 0 {
		debug.Print("Cleaned", cleaned, "rooms")
	}
}

func (cache *RoomCache) Unload(node *Room) {
	cache.Lock()
	defer cache.Unlock()
	cache.llPop(node)
	node.Unload()
}

func (cache *RoomCache) newRoom(roomID string) *Room {
	node := NewRoom(roomID, cache)
	cache.Map[node.ID] = node
	return node
}
