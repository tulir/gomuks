// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

import (
	"encoding/json"
	"fmt"

	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

func EventTypeName(evt any) Name {
	switch evt.(type) {
	case *SyncComplete:
		return EventSyncComplete
	case *SyncStatus:
		return EventSyncStatus
	case *EventsDecrypted:
		return EventEventsDecrypted
	case *Typing:
		return EventTyping
	case *SendComplete:
		return EventSendComplete
	case *ClientState:
		return EventClientState
	default:
		panic(fmt.Errorf("unknown event type %T", evt))
	}
}

type SyncRoom struct {
	Meta        *database.Room                                `json:"meta"`
	Timeline    []database.TimelineRowTuple                   `json:"timeline"`
	State       map[event.Type]map[string]database.EventRowID `json:"state"`
	AccountData map[event.Type]*database.AccountData          `json:"account_data"`
	Events      []*database.Event                             `json:"events"`
	Reset       bool                                          `json:"reset"`
	Receipts    map[id.EventID][]*database.Receipt            `json:"receipts"`

	DismissNotifications bool               `json:"dismiss_notifications"`
	Notifications        []SyncNotification `json:"notifications"`
}

type SyncNotification struct {
	RowID     database.EventRowID `json:"event_rowid"`
	Sound     bool                `json:"sound"`
	Highlight bool                `json:"highlight"`
	Event     *database.Event     `json:"-"`
	Room      *database.Room      `json:"-"`
}

type SyncToDevice struct {
	Sender    id.UserID       `json:"sender"`
	Type      event.Type      `json:"type"`
	Content   json.RawMessage `json:"content"`
	Encrypted bool            `json:"encrypted"`
}

type SyncComplete struct {
	Since          *string                              `json:"since,omitempty"`
	ClearState     bool                                 `json:"clear_state,omitempty"`
	AccountData    map[event.Type]*database.AccountData `json:"account_data"`
	Rooms          map[id.RoomID]*SyncRoom              `json:"rooms"`
	LeftRooms      []id.RoomID                          `json:"left_rooms"`
	InvitedRooms   []*database.InvitedRoom              `json:"invited_rooms"`
	SpaceEdges     map[id.RoomID][]*database.SpaceEdge  `json:"space_edges"`
	TopLevelSpaces []id.RoomID                          `json:"top_level_spaces"`

	ToDevice []*SyncToDevice `json:"to_device,omitempty"`
}

func (c *SyncComplete) Notifications(yield func(SyncNotification) bool) {
	for _, room := range c.Rooms {
		for _, notif := range room.Notifications {
			if !yield(notif) {
				return
			}
		}
	}
}

func (c *SyncComplete) IsEmpty() bool {
	return len(c.Rooms) == 0 && len(c.LeftRooms) == 0 && len(c.InvitedRooms) == 0 && len(c.AccountData) == 0
}

type SyncStatusType string

const (
	SyncStatusOK       SyncStatusType = "ok"
	SyncStatusWaiting  SyncStatusType = "waiting"
	SyncStatusErroring SyncStatusType = "erroring"
	SyncStatusFailed   SyncStatusType = "permanently-failed"
)

type SyncStatus struct {
	Type       SyncStatusType     `json:"type"`
	Error      string             `json:"error,omitempty"`
	ErrorCount int                `json:"error_count"`
	LastSync   jsontime.UnixMilli `json:"last_sync,omitempty"`
}

type EventsDecrypted struct {
	RoomID            id.RoomID           `json:"room_id"`
	PreviewEventRowID database.EventRowID `json:"preview_event_rowid,omitempty"`
	Events            []*database.Event   `json:"events"`
}

type Typing struct {
	RoomID id.RoomID `json:"room_id"`
	event.TypingEventContent
}

type SendComplete struct {
	Event *database.Event `json:"event"`
	Error error           `json:"error"`
}

type ClientState struct {
	IsLoggedIn    bool        `json:"is_logged_in"`
	IsVerified    bool        `json:"is_verified"`
	UserID        id.UserID   `json:"user_id,omitempty"`
	DeviceID      id.DeviceID `json:"device_id,omitempty"`
	HomeserverURL string      `json:"homeserver_url,omitempty"`
}

type ImageAuthToken string

type InitComplete struct{}

type RunData struct {
	RunID string `json:"run_id"`
	ETag  string `json:"etag"`
}
