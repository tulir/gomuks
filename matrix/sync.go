package matrix

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/config"
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

func (s *GomuksSyncer) ProcessResponse(res *gomatrix.RespSync, since string) (err error) {
	if !s.shouldProcessResponse(res, since) {
		return
	}
	// gdebug.Print("Processing sync response", since, res)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ProcessResponse panicked! userID=%s since=%s panic=%s\n%s", s.Session.MXID, since, r, debug.Stack())
		}
	}()

	for _, event := range res.Presence.Events {
		s.notifyListeners(event)
	}
	for roomID, roomData := range res.Rooms.Join {
		room := s.Session.GetRoom(roomID)
		for _, event := range roomData.State.Events {
			event.RoomID = roomID
			room.UpdateState(event)
			s.notifyListeners(event)
		}
		for _, event := range roomData.Timeline.Events {
			event.RoomID = roomID
			s.notifyListeners(event)
		}
		for _, event := range roomData.Ephemeral.Events {
			event.RoomID = roomID
			s.notifyListeners(event)
		}

		if len(room.PrevBatch) == 0 {
			room.PrevBatch = roomData.Timeline.PrevBatch
		}
	}
	for roomID, roomData := range res.Rooms.Invite {
		room := s.Session.GetRoom(roomID)
		for _, event := range roomData.State.Events {
			event.RoomID = roomID
			room.UpdateState(event)
			s.notifyListeners(event)
		}
	}
	for roomID, roomData := range res.Rooms.Leave {
		room := s.Session.GetRoom(roomID)
		for _, event := range roomData.Timeline.Events {
			if event.StateKey != nil {
				event.RoomID = roomID
				room.UpdateState(event)
				s.notifyListeners(event)
			}
		}

		if len(room.PrevBatch) == 0 {
			room.PrevBatch = roomData.Timeline.PrevBatch
		}
	}
	return
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

// shouldProcessResponse returns true if the response should be processed. May modify the response to remove
// stuff that shouldn't be processed.
func (s *GomuksSyncer) shouldProcessResponse(resp *gomatrix.RespSync, since string) bool {
	if since == "" {
		return false
	}
	return true
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
