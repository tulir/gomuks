// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type SyncRoom struct {
	Meta          *database.Room                                `json:"meta"`
	Timeline      []database.TimelineRowTuple                   `json:"timeline"`
	State         map[event.Type]map[string]database.EventRowID `json:"state"`
	AccountData   map[event.Type]*database.AccountData          `json:"account_data"`
	Events        []*database.Event                             `json:"events"`
	Reset         bool                                          `json:"reset"`
	Notifications []SyncNotification                            `json:"notifications"`
}

type SyncNotification struct {
	RowID database.EventRowID `json:"event_rowid"`
	Sound bool                `json:"sound"`
}

type SyncComplete struct {
	Since       *string                              `json:"since,omitempty"`
	ClearState  bool                                 `json:"clear_state,omitempty"`
	Rooms       map[id.RoomID]*SyncRoom              `json:"rooms"`
	AccountData map[event.Type]*database.AccountData `json:"account_data"`
	LeftRooms   []id.RoomID                          `json:"left_rooms"`
}

func (c *SyncComplete) IsEmpty() bool {
	return len(c.Rooms) == 0 && len(c.LeftRooms) == 0 && len(c.AccountData) == 0
}

type SyncStatusType string

const (
	SyncStatusOK      SyncStatusType = "ok"
	SyncStatusWaiting SyncStatusType = "waiting"
	SyncStatusErrored SyncStatusType = "errored"
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
