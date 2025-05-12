// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

type Container[T any] struct {
	Command   Name  `json:"command"`
	RequestID int64 `json:"request_id"`
	Data      T     `json:"data"`
}

type Name string

func (n Name) String() string {
	return string(n)
}

const (
	ReqGetState                 Name = "get_state"
	ReqCancel                   Name = "cancel"
	ReqSendMessage              Name = "send_message"
	ReqSendEvent                Name = "send_event"
	ReqResendEvent              Name = "resend_event"
	ReqReportEvent              Name = "report_event"
	ReqRedactEvent              Name = "redact_event"
	ReqSetState                 Name = "set_state"
	ReqUpdateDelayedEvent       Name = "update_delayed_event"
	ReqSetMembership            Name = "set_membership"
	ReqSetAccountData           Name = "set_account_data"
	ReqMarkRead                 Name = "mark_read"
	ReqSetTyping                Name = "set_typing"
	ReqGetProfile               Name = "get_profile"
	ReqSetProfileField          Name = "set_profile_field"
	ReqGetMutualRooms           Name = "get_mutual_rooms"
	ReqTrackUserDevices         Name = "track_user_devices"
	ReqGetProfileEncryptionInfo Name = "get_profile_encryption_info"
	ReqGetEvent                 Name = "get_event"
	ReqGetRelatedEvents         Name = "get_related_events"
	ReqGetRoomState             Name = "get_room_state"
	ReqGetSpecificRoomState     Name = "get_specific_room_state"
	ReqGetReceipts              Name = "get_receipts"
	ReqPaginate                 Name = "paginate"
	ReqPaginateServer           Name = "paginate_server"
	ReqGetRoomSummary           Name = "get_room_summary"
	ReqJoinRoom                 Name = "join_room"
	ReqKnockRoom                Name = "knock_room"
	ReqLeaveRoom                Name = "leave_room"
	ReqCreateRoom               Name = "create_room"
	ReqMuteRoom                 Name = "mute_room"
	ReqEnsureGroupSessionShared Name = "ensure_group_session_shared"
	ReqSendToDevice             Name = "send_to_device"
	ReqResolveAlias             Name = "resolve_alias"
	ReqRequestOpenIDToken       Name = "request_openid_token"
	ReqLogout                   Name = "logout"
	ReqLogin                    Name = "login"
	ReqLoginCustom              Name = "login_custom"
	ReqVerify                   Name = "verify"
	ReqDiscoverHomeserver       Name = "discover_homeserver"
	ReqGetLoginFlows            Name = "get_login_flows"
	ReqRegisterPush             Name = "register_push"
	ReqListenToDevice           Name = "listen_to_device"
	ReqGetTurnServers           Name = "get_turn_servers"
	ReqGetMediaConfig           Name = "get_media_config"

	RespError   Name = "error"
	RespSuccess Name = "response"

	ReqPing  Name = "ping"
	RespPong Name = "pong"

	EventSyncComplete    Name = "sync_complete"
	EventSyncStatus      Name = "sync_status"
	EventEventsDecrypted Name = "events_decrypted"
	EventTyping          Name = "typing"
	EventSendComplete    Name = "send_complete"
	EventClientState     Name = "client_state"
	EventImageAuthToken  Name = "image_auth_token"
	EventInitComplete    Name = "init_complete"
	EventRunID           Name = "run_id"
)
