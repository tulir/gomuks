// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Based on https://github.com/matrix-org/gomatrix/blob/master/sync.go

package matrix

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/matrix/rooms"
)

// GomuksSyncer is the default syncing implementation. You can either write your own syncer, or selectively
// replace parts of this default syncer (e.g. the ProcessResponse method). The default syncer uses the observer
// pattern to notify callers about incoming events. See GomuksSyncer.OnEventType for more information.
type GomuksSyncer struct {
	Session   *config.Session
	listeners map[string][]gomatrix.OnEventListener // event type to listeners array
}

// NewGomuksSyncer returns an instantiated GomuksSyncer
func NewGomuksSyncer(session *config.Session) *GomuksSyncer {
	return &GomuksSyncer{
		Session:   session,
		listeners: make(map[string][]gomatrix.OnEventListener),
	}
}

// ProcessResponse processes a Matrix sync response.
func (s *GomuksSyncer) ProcessResponse(res *gomatrix.RespSync, since string) (err error) {
	if len(since) == 0 {
		return
	}
	// debug.Print("Processing sync response", since, res)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ProcessResponse for %s since %s panicked: %s\n%s", s.Session.UserID, since, r, debug.Stack())
		}
	}()

	s.processSyncEvents(nil, res.Presence.Events, false, false)
	s.processSyncEvents(nil, res.AccountData.Events, false, false)

	for roomID, roomData := range res.Rooms.Join {
		room := s.Session.GetRoom(roomID)
		s.processSyncEvents(room, roomData.State.Events, true, false)
		s.processSyncEvents(room, roomData.Timeline.Events, false, false)
		s.processSyncEvents(room, roomData.Ephemeral.Events, false, false)

		if len(room.PrevBatch) == 0 {
			room.PrevBatch = roomData.Timeline.PrevBatch
		}
	}

	for roomID, roomData := range res.Rooms.Invite {
		room := s.Session.GetRoom(roomID)
		s.processSyncEvents(room, roomData.State.Events, true, false)
	}

	for roomID, roomData := range res.Rooms.Leave {
		room := s.Session.GetRoom(roomID)
		s.processSyncEvents(room, roomData.Timeline.Events, true, true)

		if len(room.PrevBatch) == 0 {
			room.PrevBatch = roomData.Timeline.PrevBatch
		}
	}

	return
}

func (s *GomuksSyncer) processSyncEvents(room *rooms.Room, events []*gomatrix.Event, isState bool, checkStateKey bool) {
	for _, event := range events {
		if !checkStateKey || event.StateKey != nil {
			s.processSyncEvent(room, event, isState)
		}
	}
}

func (s *GomuksSyncer) processSyncEvent(room *rooms.Room, event *gomatrix.Event, isState bool) {
	if room != nil {
		event.RoomID = room.ID
	}
	if isState {
		room.UpdateState(event)
	}
	s.notifyListeners(event)
}

// OnEventType allows callers to be notified when there are new events for the given event type.
// There are no duplicate checks.
func (s *GomuksSyncer) OnEventType(eventType string, callback gomatrix.OnEventListener) {
	_, exists := s.listeners[eventType]
	if !exists {
		s.listeners[eventType] = []gomatrix.OnEventListener{}
	}
	s.listeners[eventType] = append(s.listeners[eventType], callback)
}

func (s *GomuksSyncer) notifyListeners(event *gomatrix.Event) {
	listeners, exists := s.listeners[event.Type]
	if !exists {
		return
	}
	for _, fn := range listeners {
		fn(event)
	}
}

// OnFailedSync always returns a 10 second wait period between failed /syncs, never a fatal error.
func (s *GomuksSyncer) OnFailedSync(res *gomatrix.RespSync, err error) (time.Duration, error) {
	return 10 * time.Second, nil
}

// GetFilterJSON returns a filter with a timeline limit of 50.
func (s *GomuksSyncer) GetFilterJSON(userID string) json.RawMessage {
	return json.RawMessage(`{"room":{"timeline":{"limit":50}}}`)
}
