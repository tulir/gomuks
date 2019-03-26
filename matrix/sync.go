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

// Based on https://github.com/matrix-org/mautrix/blob/master/sync.go

package matrix

import (
	"encoding/json"
	"fmt"
	"time"

	"maunium.net/go/mautrix"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/rooms"
)

type SyncerSession interface {
	GetRoom(id string) *rooms.Room
	GetUserID() string
}

type EventSource int

const (
	EventSourcePresence EventSource = 1 << iota
	EventSourceJoin
	EventSourceInvite
	EventSourceLeave
	EventSourceAccountData
	EventSourceTimeline
	EventSourceState
	EventSourceEphemeral
)

func (es EventSource) String() string {
	switch {
	case es == EventSourcePresence:
		return "presence"
	case es == EventSourceAccountData:
		return "user account data"
	case es&EventSourceJoin != 0:
		es -= EventSourceJoin
		switch es {
		case EventSourceState:
			return "joined state"
		case EventSourceTimeline:
			return "joined timeline"
		case EventSourceEphemeral:
			return "room ephemeral (joined)"
		case EventSourceAccountData:
			return "room account data (joined)"
		}
	case es&EventSourceInvite != 0:
		es -= EventSourceInvite
		switch es {
		case EventSourceState:
			return "invited state"
		}
	case es&EventSourceLeave != 0:
		es -= EventSourceLeave
		switch es {
		case EventSourceState:
			return "left state"
		case EventSourceTimeline:
			return "left timeline"
		}
	}
	return fmt.Sprintf("unknown (%d)", es)
}

type EventHandler func(source EventSource, event *mautrix.Event)

// GomuksSyncer is the default syncing implementation. You can either write your own syncer, or selectively
// replace parts of this default syncer (e.g. the ProcessResponse method). The default syncer uses the observer
// pattern to notify callers about incoming events. See GomuksSyncer.OnEventType for more information.
type GomuksSyncer struct {
	Session          SyncerSession
	listeners        map[mautrix.EventType][]EventHandler // event type to listeners array
	FirstSyncDone    bool
	InitDoneCallback func()
}

// NewGomuksSyncer returns an instantiated GomuksSyncer
func NewGomuksSyncer(session SyncerSession) *GomuksSyncer {
	return &GomuksSyncer{
		Session:       session,
		listeners:     make(map[mautrix.EventType][]EventHandler),
		FirstSyncDone: false,
	}
}

// ProcessResponse processes a Matrix sync response.
func (s *GomuksSyncer) ProcessResponse(res *mautrix.RespSync, since string) (err error) {
	debug.Print("Received sync response")
	s.processSyncEvents(nil, res.Presence.Events, EventSourcePresence)
	s.processSyncEvents(nil, res.AccountData.Events, EventSourceAccountData)

	for roomID, roomData := range res.Rooms.Join {
		room := s.Session.GetRoom(roomID)
		s.processSyncEvents(room, roomData.State.Events, EventSourceJoin|EventSourceState)
		s.processSyncEvents(room, roomData.Timeline.Events, EventSourceJoin|EventSourceTimeline)
		s.processSyncEvents(room, roomData.Ephemeral.Events, EventSourceJoin|EventSourceEphemeral)
		s.processSyncEvents(room, roomData.AccountData.Events, EventSourceJoin|EventSourceAccountData)

		if len(room.PrevBatch) == 0 {
			room.PrevBatch = roomData.Timeline.PrevBatch
		}
	}

	for roomID, roomData := range res.Rooms.Invite {
		room := s.Session.GetRoom(roomID)
		s.processSyncEvents(room, roomData.State.Events, EventSourceInvite|EventSourceState)
	}

	for roomID, roomData := range res.Rooms.Leave {
		room := s.Session.GetRoom(roomID)
		room.HasLeft = true
		s.processSyncEvents(room, roomData.State.Events, EventSourceLeave|EventSourceState)
		s.processSyncEvents(room, roomData.Timeline.Events, EventSourceLeave|EventSourceTimeline)

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

func (s *GomuksSyncer) processSyncEvents(room *rooms.Room, events []*mautrix.Event, source EventSource) {
	for _, event := range events {
		s.processSyncEvent(room, event, source)
	}
}

func (s *GomuksSyncer) processSyncEvent(room *rooms.Room, event *mautrix.Event, source EventSource) {
	if room != nil {
		event.RoomID = room.ID
		if source&EventSourceState != 0 || (source&EventSourceTimeline != 0 && event.Type.IsState() && event.StateKey != nil) {
			room.UpdateState(event)
		}
	}
	s.notifyListeners(source, event)
}

// OnEventType allows callers to be notified when there are new events for the given event type.
// There are no duplicate checks.
func (s *GomuksSyncer) OnEventType(eventType mautrix.EventType, callback EventHandler) {
	_, exists := s.listeners[eventType]
	if !exists {
		s.listeners[eventType] = []EventHandler{}
	}
	s.listeners[eventType] = append(s.listeners[eventType], callback)
}

func (s *GomuksSyncer) notifyListeners(source EventSource, event *mautrix.Event) {
	if (event.Type.IsState() && source&EventSourceState == 0 && event.StateKey == nil) ||
		(event.Type.IsAccountData() && source&EventSourceAccountData == 0) ||
		(event.Type.IsEphemeral() && source&EventSourceEphemeral == 0) {
		evtJson, _ := json.Marshal(event)
		debug.Printf("Event of type %s received from mismatching source %s: %s", event.Type.String(), source.String(), string(evtJson))
		return
	}
	listeners, exists := s.listeners[event.Type]
	if !exists {
		return
	}
	for _, fn := range listeners {
		fn(source, event)
	}
}

// OnFailedSync always returns a 10 second wait period between failed /syncs, never a fatal error.
func (s *GomuksSyncer) OnFailedSync(res *mautrix.RespSync, err error) (time.Duration, error) {
	debug.Printf("Sync failed: %v", err)
	return 10 * time.Second, nil
}

// GetFilterJSON returns a filter with a timeline limit of 50.
func (s *GomuksSyncer) GetFilterJSON(userID string) json.RawMessage {
	filter := &mautrix.Filter{
		Room: mautrix.RoomFilter{
			IncludeLeave: false,
			State: mautrix.FilterPart{
				Types: []string{
					"m.room.member",
					"m.room.name",
					"m.room.topic",
					"m.room.canonical_alias",
					"m.room.aliases",
				},
			},
			Timeline: mautrix.FilterPart{
				Types: []string{
					"m.room.message",
					"m.room.member",
					"m.room.name",
					"m.room.topic",
					"m.room.canonical_alias",
					"m.room.aliases",
				},
				Limit: 50,
			},
			Ephemeral: mautrix.FilterPart{
				Types: []string{"m.typing", "m.receipt"},
			},
			AccountData: mautrix.FilterPart{
				Types: []string{"m.tag"},
			},
		},
		AccountData: mautrix.FilterPart{
			Types: []string{"m.push_rules", "m.direct", "net.maunium.gomuks.preferences"},
		},
		Presence: mautrix.FilterPart{
			Types: []string{},
		},
	}
	rawFilter, _ := json.Marshal(&filter)
	return rawFilter
}
