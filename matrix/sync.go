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
	"time"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/matrix/rooms"
)

type SyncerSession interface {
	GetRoom(id string) *rooms.Room
	GetUserID() string
}

type EventSource int

const (
	EventSourcePresence    EventSource = 1 << iota
	EventSourceJoin
	EventSourceInvite
	EventSourceLeave
	EventSourceAccountData
	EventSourceTimeline
	EventSourceState
	EventSourceEphemeral
)

type EventHandler func(source EventSource, event *gomatrix.Event)

// GomuksSyncer is the default syncing implementation. You can either write your own syncer, or selectively
// replace parts of this default syncer (e.g. the ProcessResponse method). The default syncer uses the observer
// pattern to notify callers about incoming events. See GomuksSyncer.OnEventType for more information.
type GomuksSyncer struct {
	Session          SyncerSession
	listeners        map[string][]EventHandler // event type to listeners array
	FirstSyncDone    bool
	InitDoneCallback func()
}

// NewGomuksSyncer returns an instantiated GomuksSyncer
func NewGomuksSyncer(session SyncerSession) *GomuksSyncer {
	return &GomuksSyncer{
		Session:       session,
		listeners:     make(map[string][]EventHandler),
		FirstSyncDone: false,
	}
}

// ProcessResponse processes a Matrix sync response.
func (s *GomuksSyncer) ProcessResponse(res *gomatrix.RespSync, since string) (err error) {
	s.processSyncEvents(nil, res.Presence.Events, EventSourcePresence, false)
	s.processSyncEvents(nil, res.AccountData.Events, EventSourceAccountData, false)

	for roomID, roomData := range res.Rooms.Join {
		room := s.Session.GetRoom(roomID)
		s.processSyncEvents(room, roomData.State.Events, EventSourceJoin | EventSourceState, false)
		s.processSyncEvents(room, roomData.Timeline.Events, EventSourceJoin | EventSourceTimeline, false)
		s.processSyncEvents(room, roomData.Ephemeral.Events, EventSourceJoin | EventSourceEphemeral, false)
		s.processSyncEvents(room, roomData.AccountData.Events, EventSourceJoin | EventSourceAccountData, false)

		if len(room.PrevBatch) == 0 {
			room.PrevBatch = roomData.Timeline.PrevBatch
		}
	}

	for roomID, roomData := range res.Rooms.Invite {
		room := s.Session.GetRoom(roomID)
		s.processSyncEvents(room, roomData.State.Events, EventSourceInvite | EventSourceState, false)
	}

	for roomID, roomData := range res.Rooms.Leave {
		room := s.Session.GetRoom(roomID)
		room.HasLeft = true
		s.processSyncEvents(room, roomData.State.Events, EventSourceLeave | EventSourceState, true)
		s.processSyncEvents(room, roomData.Timeline.Events, EventSourceLeave | EventSourceTimeline, false)

		if len(room.PrevBatch) == 0 {
			room.PrevBatch = roomData.Timeline.PrevBatch
		}
	}

	if since == "" && s.InitDoneCallback != nil {
		s.InitDoneCallback()
	}
	s.FirstSyncDone = true

	return
}

func (s *GomuksSyncer) processSyncEvents(room *rooms.Room, events []*gomatrix.Event, source EventSource, checkStateKey bool) {
	for _, event := range events {
		if !checkStateKey || event.StateKey != nil {
			s.processSyncEvent(room, event, source)
		}
	}
}

func isState(event *gomatrix.Event) bool {
	switch event.Type {
	case "m.room.member", "m.room.name", "m.room.topic", "m.room.aliases", "m.room.canonical_alias":
		return true
	default:
		return false
	}
}

func (s *GomuksSyncer) processSyncEvent(room *rooms.Room, event *gomatrix.Event, source EventSource) {
	if room != nil {
		event.RoomID = room.ID
	}
	if isState(event) {
		room.UpdateState(event)
	}
	s.notifyListeners(source, event)
}

// OnEventType allows callers to be notified when there are new events for the given event type.
// There are no duplicate checks.
func (s *GomuksSyncer) OnEventType(eventType string, callback EventHandler) {
	_, exists := s.listeners[eventType]
	if !exists {
		s.listeners[eventType] = []EventHandler{}
	}
	s.listeners[eventType] = append(s.listeners[eventType], callback)
}

func (s *GomuksSyncer) notifyListeners(source EventSource, event *gomatrix.Event) {
	listeners, exists := s.listeners[event.Type]
	if !exists {
		return
	}
	for _, fn := range listeners {
		fn(source, event)
	}
}

// OnFailedSync always returns a 10 second wait period between failed /syncs, never a fatal error.
func (s *GomuksSyncer) OnFailedSync(res *gomatrix.RespSync, err error) (time.Duration, error) {
	return 10 * time.Second, nil
}

// GetFilterJSON returns a filter with a timeline limit of 50.
func (s *GomuksSyncer) GetFilterJSON(userID string) json.RawMessage {
	filter := &gomatrix.Filter{
		Room: gomatrix.RoomFilter{
			IncludeLeave: false,
			State: gomatrix.FilterPart{
				Types: []string{
					"m.room.member",
					"m.room.name",
					"m.room.topic",
					"m.room.canonical_alias",
					"m.room.aliases",
				},
			},
			Timeline: gomatrix.FilterPart{
				Types: []string{"m.room.message", "m.room.member"},
				Limit: 50,
			},
			Ephemeral: gomatrix.FilterPart{
				Types: []string{"m.typing", "m.receipt"},
			},
			AccountData: gomatrix.FilterPart{
				Types: []string{"m.tag"},
			},
		},
		AccountData: gomatrix.FilterPart{
			Types: []string{"m.push_rules", "m.direct"},
		},
		Presence: gomatrix.FilterPart{
			Types: []string{},
		},
	}
	rawFilter, _ := json.Marshal(&filter)
	return rawFilter
}
