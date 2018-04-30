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

package matrix_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/matrix"
	"maunium.net/go/gomuks/matrix/rooms"
)

func TestGomuksSyncer_ProcessResponse_Initial(t *testing.T) {
	syncer := matrix.NewGomuksSyncer(&mockSyncerSession{})
	var initDoneCalled = false
	syncer.InitDoneCallback = func() {
		initDoneCalled = true
	}

	syncer.ProcessResponse(newRespSync(), "")
	assert.True(t, syncer.FirstSyncDone)
	assert.True(t, initDoneCalled)
}

func TestGomuksSyncer_ProcessResponse(t *testing.T) {
	mss := &mockSyncerSession{
		userID: "@tulir:maunium.net",
		rooms: map[string]*rooms.Room{
			"!foo:maunium.net": {
				Room: gomatrix.NewRoom("!foo:maunium.net"),
			},
			"!bar:maunium.net": {
				Room: gomatrix.NewRoom("!bar:maunium.net"),
			},
			"!test:maunium.net": {
				Room: gomatrix.NewRoom("!test:maunium.net"),
			},
		},
	}
	ml := &mockListener{}
	syncer := matrix.NewGomuksSyncer(mss)
	syncer.OnEventType("m.room.member", ml.receive)
	syncer.OnEventType("m.room.message", ml.receive)

	joinEvt := &gomatrix.Event{
		ID:       "!join:maunium.net",
		Type:     "m.room.member",
		Sender:   "@tulir:maunium.net",
		StateKey: ptr("̣@tulir:maunium.net"),
		Content: map[string]interface{}{
			"membership": "join",
		},
	}
	messageEvt := &gomatrix.Event{
		ID:   "!msg:maunium.net",
		Type: "m.room.message",
		Content: map[string]interface{}{
			"body":    "foo",
			"msgtype": "m.text",
		},
	}
	unhandledEvt := &gomatrix.Event{
		ID:   "!unhandled:maunium.net",
		Type: "m.room.unhandled_event",
	}
	inviteEvt := &gomatrix.Event{
		ID:       "!invite:matrix.org",
		Type:     "m.room.member",
		Sender:   "@you:matrix.org",
		StateKey: ptr("̣@tulir:maunium.net"),
		Content: map[string]interface{}{
			"membership": "invite",
		},
	}
	leaveEvt := &gomatrix.Event{
		ID:       "!leave:matrix.org",
		Type:     "m.room.member",
		Sender:   "@you:matrix.org",
		StateKey: ptr("̣@tulir:maunium.net"),
		Content: map[string]interface{}{
			"membership": "leave",
		},
	}

	resp := newRespSync()
	resp.Rooms.Join["!foo:maunium.net"] = join{
		State:    events{Events: []*gomatrix.Event{joinEvt}},
		Timeline: timeline{Events: []*gomatrix.Event{messageEvt, unhandledEvt}},
	}
	resp.Rooms.Invite["!bar:maunium.net"] = struct {
		State struct {
			Events []*gomatrix.Event `json:"events"`
		} `json:"invite_state"`
	}{
		State: events{Events: []*gomatrix.Event{inviteEvt}},
	}
	resp.Rooms.Leave["!test:maunium.net"] = struct {
		State struct {
			Events []*gomatrix.Event `json:"events"`
		} `json:"state"`
		Timeline struct {
			Events    []*gomatrix.Event `json:"events"`
			Limited   bool              `json:"limited"`
			PrevBatch string            `json:"prev_batch"`
		} `json:"timeline"`
	}{
		State: events{Events: []*gomatrix.Event{leaveEvt}},
	}

	syncer.ProcessResponse(resp, "since")
	assert.Contains(t, ml.received, joinEvt, joinEvt.ID)
	assert.Contains(t, ml.received, messageEvt, messageEvt.ID)
	assert.NotContains(t, ml.received, unhandledEvt, unhandledEvt.ID)
	assert.Contains(t, ml.received, inviteEvt, inviteEvt.ID)
	assert.Contains(t, ml.received, leaveEvt, leaveEvt.ID)
}

type mockSyncerSession struct {
	rooms  map[string]*rooms.Room
	userID string
}

func (mss *mockSyncerSession) GetRoom(id string) *rooms.Room {
	return mss.rooms[id]
}

func (mss *mockSyncerSession) GetUserID() string {
	return mss.userID
}

type events struct {
	Events []*gomatrix.Event `json:"events"`
}

type timeline struct {
	Events    []*gomatrix.Event `json:"events"`
	Limited   bool              `json:"limited"`
	PrevBatch string            `json:"prev_batch"`
}
type join struct {
	State struct {
		Events []*gomatrix.Event `json:"events"`
	} `json:"state"`
	Timeline struct {
		Events    []*gomatrix.Event `json:"events"`
		Limited   bool              `json:"limited"`
		PrevBatch string            `json:"prev_batch"`
	} `json:"timeline"`
	Ephemeral struct {
		Events []*gomatrix.Event `json:"events"`
	} `json:"ephemeral"`
	AccountData struct {
		Events []*gomatrix.Event `json:"events"`
	} `json:"account_data"`
}

func ptr(text string) *string {
	return &text
}

type mockListener struct {
	received []*gomatrix.Event
}

func (ml *mockListener) receive(source matrix.EventSource, evt *gomatrix.Event) {
	ml.received = append(ml.received, evt)
}

func newRespSync() *gomatrix.RespSync {
	resp := &gomatrix.RespSync{NextBatch: "123"}
	resp.Rooms.Join = make(map[string]struct {
		State struct {
			Events []*gomatrix.Event `json:"events"`
		} `json:"state"`
		Timeline struct {
			Events    []*gomatrix.Event `json:"events"`
			Limited   bool              `json:"limited"`
			PrevBatch string            `json:"prev_batch"`
		} `json:"timeline"`
		Ephemeral struct {
			Events []*gomatrix.Event `json:"events"`
		} `json:"ephemeral"`
		AccountData struct {
			Events []*gomatrix.Event `json:"events"`
		} `json:"account_data"`
	})
	resp.Rooms.Invite = make(map[string]struct {
		State struct {
			Events []*gomatrix.Event `json:"events"`
		} `json:"invite_state"`
	})
	resp.Rooms.Leave = make(map[string]struct {
		State struct {
			Events []*gomatrix.Event `json:"events"`
		} `json:"state"`
		Timeline struct {
			Events    []*gomatrix.Event `json:"events"`
			Limited   bool              `json:"limited"`
			PrevBatch string            `json:"prev_batch"`
		} `json:"timeline"`
	})
	return resp
}
