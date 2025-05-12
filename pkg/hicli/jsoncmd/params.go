// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

import (
	"encoding/json"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type CommandName string

type CancelRequestParams struct {
	RequestID int64  `json:"request_id"`
	Reason    string `json:"reason"`
}

type SendMessageParams struct {
	RoomID      id.RoomID                   `json:"room_id"`
	BaseContent *event.MessageEventContent  `json:"base_content"`
	Extra       map[string]any              `json:"extra"`
	Text        string                      `json:"text"`
	RelatesTo   *event.RelatesTo            `json:"relates_to"`
	Mentions    *event.Mentions             `json:"mentions"`
	URLPreviews *[]*event.BeeperLinkPreview `json:"url_previews"`
}

type SendEventParams struct {
	RoomID            id.RoomID       `json:"room_id"`
	EventType         event.Type      `json:"type"`
	Content           json.RawMessage `json:"content"`
	DisableEncryption bool            `json:"disable_encryption"`
	Synchronous       bool            `json:"synchronous"`
}

type ResendEventParams struct {
	TransactionID string `json:"transaction_id"`
}

type ReportEventParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
	Reason  string     `json:"reason"`
}

type RedactEventParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
	Reason  string     `json:"reason"`
}

type SendStateEventParams struct {
	RoomID    id.RoomID       `json:"room_id"`
	EventType event.Type      `json:"type"`
	StateKey  string          `json:"state_key"`
	Content   json.RawMessage `json:"content"`
	DelayMS   int             `json:"delay_ms"`
}

type UpdateDelayedEventParams struct {
	DelayID string `json:"delay_id"`
	Action  string `json:"action"`
}

type SetMembershipParams struct {
	Action string    `json:"action"`
	RoomID id.RoomID `json:"room_id"`
	UserID id.UserID `json:"user_id"`
	Reason string    `json:"reason"`
}

type SetAccountDataParams struct {
	RoomID  id.RoomID       `json:"room_id,omitempty"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

type MarkReadParams struct {
	RoomID      id.RoomID         `json:"room_id"`
	EventID     id.EventID        `json:"event_id"`
	ReceiptType event.ReceiptType `json:"receipt_type"`
}

type SetTypingParams struct {
	RoomID  id.RoomID `json:"room_id"`
	Timeout int       `json:"timeout"`
}

type GetProfileParams struct {
	UserID id.UserID `json:"user_id"`
}

type SetProfileFieldParams struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type GetEventParams struct {
	RoomID   id.RoomID  `json:"room_id"`
	EventID  id.EventID `json:"event_id"`
	Unredact bool       `json:"unredact"`
}

type GetRelatedEventsParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`

	RelationType event.RelationType `json:"relation_type"`
}

type GetRoomStateParams struct {
	RoomID         id.RoomID `json:"room_id"`
	Refetch        bool      `json:"refetch"`
	FetchMembers   bool      `json:"fetch_members"`
	IncludeMembers bool      `json:"include_members"`
}

type GetSpecificRoomStateParams struct {
	Keys []database.RoomStateGUID `json:"keys"`
}

type EnsureGroupSessionSharedParams struct {
	RoomID id.RoomID `json:"room_id"`
}

type SendToDeviceParams struct {
	*mautrix.ReqSendToDevice
	EventType event.Type `json:"event_type"`
	Encrypted bool       `json:"encrypted"`
}

type ResolveAliasParams struct {
	Alias id.RoomAlias `json:"alias"`
}

type LoginParams struct {
	HomeserverURL string `json:"homeserver_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type LoginCustomParams struct {
	HomeserverURL string            `json:"homeserver_url"`
	Request       *mautrix.ReqLogin `json:"request"`
}

type VerifyParams struct {
	RecoveryKey string `json:"recovery_key"`
}

type DiscoverHomeserverParams struct {
	UserID id.UserID `json:"user_id"`
}

type GetLoginFlowsParams struct {
	HomeserverURL string `json:"homeserver_url"`
}

type PaginateParams struct {
	RoomID        id.RoomID              `json:"room_id"`
	MaxTimelineID database.TimelineRowID `json:"max_timeline_id"`
	Limit         int                    `json:"limit"`
}

type JoinRoomParams struct {
	RoomIDOrAlias string   `json:"room_id_or_alias"`
	Via           []string `json:"via"`
	Reason        string   `json:"reason"`
}

type LeaveRoomParams struct {
	RoomID id.RoomID `json:"room_id"`
	Reason string    `json:"reason"`
}

type GetReceiptsParams struct {
	RoomID   id.RoomID    `json:"room_id"`
	EventIDs []id.EventID `json:"event_ids"`
}

type MuteRoomParams struct {
	RoomID id.RoomID `json:"room_id"`
	Muted  bool      `json:"muted"`
}

type PingParams struct {
	LastReceivedID int64 `json:"last_received_id"`
}
