// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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

package gomuks

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/coder/websocket"

	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type WebsocketCloseFunc func(websocket.StatusCode, string)

type EventBuffer struct {
	lock    sync.RWMutex
	buf     []*hicli.JSONCommand
	minID   int64
	maxID   int64
	MaxSize int

	websocketClosers map[uint64]WebsocketCloseFunc
	lastAckedID      map[uint64]int64
	eventListeners   map[uint64]func(*hicli.JSONCommand)
	nextListenerID   uint64
}

func NewEventBuffer(maxSize int) *EventBuffer {
	return &EventBuffer{
		websocketClosers: make(map[uint64]WebsocketCloseFunc),
		lastAckedID:      make(map[uint64]int64),
		eventListeners:   make(map[uint64]func(*hicli.JSONCommand)),
		buf:              make([]*hicli.JSONCommand, 0, 32),
		MaxSize:          maxSize,
		minID:            -1,
	}
}

func (eb *EventBuffer) Push(evt any) {
	data, err := json.Marshal(evt)
	if err != nil {
		panic(fmt.Errorf("failed to marshal event %T: %w", evt, err))
	}
	allowCache := true
	if syncComplete, ok := evt.(*jsoncmd.SyncComplete); ok && syncComplete.Since != nil && *syncComplete.Since == "" {
		// Don't cache initial sync responses
		allowCache = false
	} else if _, ok := evt.(*jsoncmd.Typing); ok {
		// Also don't cache typing events
		allowCache = false
	}
	eb.lock.Lock()
	defer eb.lock.Unlock()
	jc := &hicli.JSONCommand{
		Command: jsoncmd.EventTypeName(evt),
		Data:    data,
	}
	if allowCache {
		eb.addToBuffer(jc)
	}
	for _, listener := range eb.eventListeners {
		listener(jc)
	}
}

func (eb *EventBuffer) GetClosers() []WebsocketCloseFunc {
	eb.lock.Lock()
	defer eb.lock.Unlock()
	return slices.Collect(maps.Values(eb.websocketClosers))
}

func (eb *EventBuffer) Unsubscribe(listenerID uint64) {
	eb.lock.Lock()
	defer eb.lock.Unlock()
	delete(eb.eventListeners, listenerID)
	delete(eb.websocketClosers, listenerID)
}

func (eb *EventBuffer) addToBuffer(evt *hicli.JSONCommand) {
	eb.maxID--
	evt.RequestID = eb.maxID
	if len(eb.lastAckedID) > 0 {
		eb.buf = append(eb.buf, evt)
	} else {
		eb.minID = eb.maxID - 1
	}
	if len(eb.buf) > eb.MaxSize {
		eb.buf = eb.buf[len(eb.buf)-eb.MaxSize:]
		eb.minID = eb.buf[0].RequestID
	}
}

func (eb *EventBuffer) ClearListenerLastAckedID(listenerID uint64) {
	eb.lock.Lock()
	defer eb.lock.Unlock()
	delete(eb.lastAckedID, listenerID)
	eb.gc()
}

func (eb *EventBuffer) SetLastAckedID(listenerID uint64, ackedID int64) {
	eb.lock.Lock()
	defer eb.lock.Unlock()
	eb.lastAckedID[listenerID] = ackedID
	eb.gc()
}

func (eb *EventBuffer) gc() {
	neededMinID := eb.maxID
	for lid, evtID := range eb.lastAckedID {
		if evtID > eb.minID {
			delete(eb.lastAckedID, lid)
		} else if evtID > neededMinID {
			neededMinID = evtID
		}
	}
	if neededMinID < eb.minID {
		eb.buf = eb.buf[eb.minID-neededMinID:]
		eb.minID = neededMinID
	}
}

func (eb *EventBuffer) Subscribe(resumeFrom int64, closeForRestart WebsocketCloseFunc, cb func(*hicli.JSONCommand)) (uint64, []*hicli.JSONCommand) {
	eb.lock.Lock()
	defer eb.lock.Unlock()
	eb.nextListenerID++
	id := eb.nextListenerID
	eb.eventListeners[id] = cb
	if closeForRestart != nil {
		eb.websocketClosers[id] = closeForRestart
	}
	var resumeData []*hicli.JSONCommand
	if resumeFrom < eb.minID {
		resumeData = eb.buf[eb.minID-resumeFrom+1:]
		eb.lastAckedID[id] = resumeFrom
	} else {
		eb.lastAckedID[id] = eb.maxID
	}
	return id, resumeData
}
