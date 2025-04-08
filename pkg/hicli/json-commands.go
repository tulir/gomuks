// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

func (h *HiClient) handleJSONCommand(ctx context.Context, req *JSONCommand) (any, error) {
	switch req.Command {
	case "get_state":
		return h.State(), nil
	case "cancel":
		return unmarshalAndCall(req.Data, func(params *cancelRequestParams) (bool, error) {
			h.jsonRequestsLock.Lock()
			cancelTarget, ok := h.jsonRequests[params.RequestID]
			h.jsonRequestsLock.Unlock()
			if ok {
				return false, nil
			}
			if params.Reason == "" {
				cancelTarget(nil)
			} else {
				cancelTarget(errors.New(params.Reason))
			}
			return true, nil
		})
	case "send_message":
		return unmarshalAndCall(req.Data, func(params *sendMessageParams) (*database.Event, error) {
			return h.SendMessage(ctx, params.RoomID, params.BaseContent, params.Extra, params.Text, params.RelatesTo, params.Mentions)
		})
	case "send_event":
		return unmarshalAndCall(req.Data, func(params *sendEventParams) (*database.Event, error) {
			return h.Send(ctx, params.RoomID, params.EventType, params.Content, params.DisableEncryption, params.Synchronous)
		})
	case "resend_event":
		return unmarshalAndCall(req.Data, func(params *resendEventParams) (*database.Event, error) {
			return h.Resend(ctx, params.TransactionID)
		})
	case "report_event":
		return unmarshalAndCall(req.Data, func(params *reportEventParams) (bool, error) {
			return true, h.Client.ReportEvent(ctx, params.RoomID, params.EventID, params.Reason)
		})
	case "redact_event":
		return unmarshalAndCall(req.Data, func(params *redactEventParams) (*mautrix.RespSendEvent, error) {
			return h.Client.RedactEvent(ctx, params.RoomID, params.EventID, mautrix.ReqRedact{
				Reason: params.Reason,
			})
		})
	case "set_state":
		return unmarshalAndCall(req.Data, func(params *sendStateEventParams) (id.EventID, error) {
			return h.SetState(ctx, params.RoomID, params.EventType, params.StateKey, params.Content, mautrix.ReqSendEvent{
				UnstableDelay: time.Duration(params.DelayMS) * time.Millisecond,
			})
		})
	case "update_delayed_event":
		return unmarshalAndCall(req.Data, func(params *updateDelayedEventParams) (*mautrix.RespUpdateDelayedEvent, error) {
			return h.Client.UpdateDelayedEvent(ctx, &mautrix.ReqUpdateDelayedEvent{
				DelayID: params.DelayID,
				Action:  params.Action,
			})
		})
	case "set_membership":
		return unmarshalAndCall(req.Data, func(params *setMembershipParams) (any, error) {
			switch params.Action {
			case "invite":
				return h.Client.InviteUser(ctx, params.RoomID, &mautrix.ReqInviteUser{UserID: params.UserID, Reason: params.Reason})
			case "kick":
				return h.Client.KickUser(ctx, params.RoomID, &mautrix.ReqKickUser{UserID: params.UserID, Reason: params.Reason})
			case "ban":
				return h.Client.BanUser(ctx, params.RoomID, &mautrix.ReqBanUser{UserID: params.UserID, Reason: params.Reason})
			case "unban":
				return h.Client.UnbanUser(ctx, params.RoomID, &mautrix.ReqUnbanUser{UserID: params.UserID, Reason: params.Reason})
			default:
				return nil, fmt.Errorf("unknown action %q", params.Action)
			}
		})
	case "set_account_data":
		return unmarshalAndCall(req.Data, func(params *setAccountDataParams) (bool, error) {
			if params.RoomID != "" {
				return true, h.Client.SetRoomAccountData(ctx, params.RoomID, params.Type, params.Content)
			} else {
				return true, h.Client.SetAccountData(ctx, params.Type, params.Content)
			}
		})
	case "mark_read":
		return unmarshalAndCall(req.Data, func(params *markReadParams) (bool, error) {
			return true, h.MarkRead(ctx, params.RoomID, params.EventID, params.ReceiptType)
		})
	case "set_typing":
		return unmarshalAndCall(req.Data, func(params *setTypingParams) (bool, error) {
			return true, h.SetTyping(ctx, params.RoomID, time.Duration(params.Timeout)*time.Millisecond)
		})
	case "get_profile":
		return unmarshalAndCall(req.Data, func(params *getProfileParams) (*mautrix.RespUserProfile, error) {
			return h.Client.GetProfile(ctx, params.UserID)
		})
	case "set_profile_field":
		return unmarshalAndCall(req.Data, func(params *setProfileFieldParams) (bool, error) {
			return true, h.Client.UnstableSetProfileField(ctx, params.Field, params.Value)
		})
	case "get_mutual_rooms":
		return unmarshalAndCall(req.Data, func(params *getProfileParams) ([]id.RoomID, error) {
			return h.GetMutualRooms(ctx, params.UserID)
		})
	case "track_user_devices":
		return unmarshalAndCall(req.Data, func(params *getProfileParams) (*ProfileEncryptionInfo, error) {
			err := h.TrackUserDevices(ctx, params.UserID)
			if err != nil {
				return nil, err
			}
			return h.GetProfileEncryptionInfo(ctx, params.UserID)
		})
	case "get_profile_encryption_info":
		return unmarshalAndCall(req.Data, func(params *getProfileParams) (*ProfileEncryptionInfo, error) {
			return h.GetProfileEncryptionInfo(ctx, params.UserID)
		})
	case "get_event":
		return unmarshalAndCall(req.Data, func(params *getEventParams) (*database.Event, error) {
			if params.Unredact {
				return h.GetUnredactedEvent(ctx, params.RoomID, params.EventID)
			}
			return h.GetEvent(ctx, params.RoomID, params.EventID)
		})
	case "get_related_events":
		return unmarshalAndCall(req.Data, func(params *getRelatedEventsParams) ([]*database.Event, error) {
			return h.DB.Event.GetRelatedEvents(ctx, params.RoomID, params.EventID, params.RelationType)
		})
	case "get_room_state":
		return unmarshalAndCall(req.Data, func(params *getRoomStateParams) ([]*database.Event, error) {
			return h.GetRoomState(ctx, params.RoomID, params.IncludeMembers, params.FetchMembers, params.Refetch)
		})
	case "get_specific_room_state":
		return unmarshalAndCall(req.Data, func(params *getSpecificRoomStateParams) ([]*database.Event, error) {
			return h.DB.CurrentState.GetMany(ctx, params.Keys)
		})
	case "get_receipts":
		return unmarshalAndCall(req.Data, func(params *getReceiptsParams) (map[id.EventID][]*database.Receipt, error) {
			return h.GetReceipts(ctx, params.RoomID, params.EventIDs)
		})
	case "paginate":
		return unmarshalAndCall(req.Data, func(params *paginateParams) (*PaginationResponse, error) {
			return h.Paginate(ctx, params.RoomID, params.MaxTimelineID, params.Limit)
		})
	case "paginate_server":
		return unmarshalAndCall(req.Data, func(params *paginateParams) (*PaginationResponse, error) {
			return h.PaginateServer(ctx, params.RoomID, params.Limit)
		})
	case "get_room_summary":
		return unmarshalAndCall(req.Data, func(params *joinRoomParams) (*mautrix.RespRoomSummary, error) {
			return h.Client.GetRoomSummary(ctx, params.RoomIDOrAlias, params.Via...)
		})
	case "join_room":
		return unmarshalAndCall(req.Data, func(params *joinRoomParams) (*mautrix.RespJoinRoom, error) {
			return h.Client.JoinRoom(ctx, params.RoomIDOrAlias, &mautrix.ReqJoinRoom{
				Via:    params.Via,
				Reason: params.Reason,
			})
		})
	case "knock_room":
		return unmarshalAndCall(req.Data, func(params *joinRoomParams) (*mautrix.RespKnockRoom, error) {
			return h.Client.KnockRoom(ctx, params.RoomIDOrAlias, &mautrix.ReqKnockRoom{
				Via:    params.Via,
				Reason: params.Reason,
			})
		})
	case "leave_room":
		return unmarshalAndCall(req.Data, func(params *leaveRoomParams) (*mautrix.RespLeaveRoom, error) {
			return h.Client.LeaveRoom(ctx, params.RoomID, &mautrix.ReqLeave{Reason: params.Reason})
		})
	case "create_room":
		return unmarshalAndCall(req.Data, func(params *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error) {
			return h.Client.CreateRoom(ctx, params)
		})
	case "mute_room":
		return unmarshalAndCall(req.Data, func(params *muteRoomParams) (bool, error) {
			if params.Muted {
				return true, h.Client.PutPushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID), &mautrix.ReqPutPushRule{
					Actions: []pushrules.PushActionType{},
				})
			} else {
				return false, h.Client.DeletePushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID))
			}
		})
	case "ensure_group_session_shared":
		return unmarshalAndCall(req.Data, func(params *ensureGroupSessionSharedParams) (bool, error) {
			return true, h.EnsureGroupSessionShared(ctx, params.RoomID)
		})
	case "send_to_device":
		return unmarshalAndCall(req.Data, func(params *sendToDeviceParams) (*mautrix.RespSendToDevice, error) {
			params.EventType.Class = event.ToDeviceEventType
			return h.SendToDevice(ctx, params.EventType, params.ReqSendToDevice, params.Encrypted)
		})
	case "resolve_alias":
		return unmarshalAndCall(req.Data, func(params *resolveAliasParams) (*mautrix.RespAliasResolve, error) {
			return h.Client.ResolveAlias(ctx, params.Alias)
		})
	case "request_openid_token":
		return h.Client.RequestOpenIDToken(ctx)
	case "logout":
		if h.LogoutFunc == nil {
			return nil, errors.New("logout not supported")
		}
		return true, h.LogoutFunc(ctx)
	case "login":
		return unmarshalAndCall(req.Data, func(params *loginParams) (bool, error) {
			return true, h.LoginPassword(ctx, params.HomeserverURL, params.Username, params.Password)
		})
	case "login_custom":
		return unmarshalAndCall(req.Data, func(params *loginCustomParams) (bool, error) {
			var err error
			h.Client.HomeserverURL, err = url.Parse(params.HomeserverURL)
			if err != nil {
				return false, err
			}
			return true, h.Login(ctx, params.Request)
		})
	case "verify":
		return unmarshalAndCall(req.Data, func(params *verifyParams) (bool, error) {
			return true, h.Verify(ctx, params.RecoveryKey)
		})
	case "discover_homeserver":
		return unmarshalAndCall(req.Data, func(params *discoverHomeserverParams) (*mautrix.ClientWellKnown, error) {
			_, homeserver, err := params.UserID.Parse()
			if err != nil {
				return nil, err
			}
			return mautrix.DiscoverClientAPI(ctx, homeserver)
		})
	case "get_login_flows":
		return unmarshalAndCall(req.Data, func(params *getLoginFlowsParams) (*mautrix.RespLoginFlows, error) {
			cli, err := h.tempClient(params.HomeserverURL)
			if err != nil {
				return nil, err
			}
			err = h.checkServerVersions(ctx, cli)
			if err != nil {
				return nil, err
			}
			return cli.GetLoginFlows(ctx)
		})
	case "register_push":
		return unmarshalAndCall(req.Data, func(params *database.PushRegistration) (bool, error) {
			return true, h.DB.PushRegistration.Put(ctx, params)
		})
	case "listen_to_device":
		return unmarshalAndCall(req.Data, func(listen *bool) (bool, error) {
			return h.ToDeviceInSync.Swap(*listen), nil
		})
	case "get_turn_servers":
		return h.Client.TurnServer(ctx)
	case "get_media_config":
		return h.Client.GetMediaConfig(ctx)
	default:
		return nil, fmt.Errorf("unknown command %q", req.Command)
	}
}

func unmarshalAndCall[T, O any](data json.RawMessage, fn func(*T) (O, error)) (output O, err error) {
	var input T
	err = json.Unmarshal(data, &input)
	if err != nil {
		return
	}
	return fn(&input)
}

type cancelRequestParams struct {
	RequestID int64  `json:"request_id"`
	Reason    string `json:"reason"`
}

type sendMessageParams struct {
	RoomID      id.RoomID                  `json:"room_id"`
	BaseContent *event.MessageEventContent `json:"base_content"`
	Extra       map[string]any             `json:"extra"`
	Text        string                     `json:"text"`
	RelatesTo   *event.RelatesTo           `json:"relates_to"`
	Mentions    *event.Mentions            `json:"mentions"`
}

type sendEventParams struct {
	RoomID            id.RoomID       `json:"room_id"`
	EventType         event.Type      `json:"type"`
	Content           json.RawMessage `json:"content"`
	DisableEncryption bool            `json:"disable_encryption"`
	Synchronous       bool            `json:"synchronous"`
}

type resendEventParams struct {
	TransactionID string `json:"transaction_id"`
}

type reportEventParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
	Reason  string     `json:"reason"`
}

type redactEventParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
	Reason  string     `json:"reason"`
}

type sendStateEventParams struct {
	RoomID    id.RoomID       `json:"room_id"`
	EventType event.Type      `json:"type"`
	StateKey  string          `json:"state_key"`
	Content   json.RawMessage `json:"content"`
	DelayMS   int             `json:"delay_ms"`
}

type updateDelayedEventParams struct {
	DelayID string `json:"delay_id"`
	Action  string `json:"action"`
}

type setMembershipParams struct {
	Action string    `json:"action"`
	RoomID id.RoomID `json:"room_id"`
	UserID id.UserID `json:"user_id"`
	Reason string    `json:"reason"`
}

type setAccountDataParams struct {
	RoomID  id.RoomID       `json:"room_id,omitempty"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

type markReadParams struct {
	RoomID      id.RoomID         `json:"room_id"`
	EventID     id.EventID        `json:"event_id"`
	ReceiptType event.ReceiptType `json:"receipt_type"`
}

type setTypingParams struct {
	RoomID  id.RoomID `json:"room_id"`
	Timeout int       `json:"timeout"`
}

type getProfileParams struct {
	UserID id.UserID `json:"user_id"`
}

type setProfileFieldParams struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type getEventParams struct {
	RoomID   id.RoomID  `json:"room_id"`
	EventID  id.EventID `json:"event_id"`
	Unredact bool       `json:"unredact"`
}

type getRelatedEventsParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`

	RelationType event.RelationType `json:"relation_type"`
}

type getRoomStateParams struct {
	RoomID         id.RoomID `json:"room_id"`
	Refetch        bool      `json:"refetch"`
	FetchMembers   bool      `json:"fetch_members"`
	IncludeMembers bool      `json:"include_members"`
}

type getSpecificRoomStateParams struct {
	Keys []database.RoomStateGUID `json:"keys"`
}

type ensureGroupSessionSharedParams struct {
	RoomID id.RoomID `json:"room_id"`
}

type sendToDeviceParams struct {
	*mautrix.ReqSendToDevice
	EventType event.Type `json:"event_type"`
	Encrypted bool       `json:"encrypted"`
}

type resolveAliasParams struct {
	Alias id.RoomAlias `json:"alias"`
}

type loginParams struct {
	HomeserverURL string `json:"homeserver_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type loginCustomParams struct {
	HomeserverURL string            `json:"homeserver_url"`
	Request       *mautrix.ReqLogin `json:"request"`
}

type verifyParams struct {
	RecoveryKey string `json:"recovery_key"`
}

type discoverHomeserverParams struct {
	UserID id.UserID `json:"user_id"`
}

type getLoginFlowsParams struct {
	HomeserverURL string `json:"homeserver_url"`
}

type paginateParams struct {
	RoomID        id.RoomID              `json:"room_id"`
	MaxTimelineID database.TimelineRowID `json:"max_timeline_id"`
	Limit         int                    `json:"limit"`
}

type joinRoomParams struct {
	RoomIDOrAlias string   `json:"room_id_or_alias"`
	Via           []string `json:"via"`
	Reason        string   `json:"reason"`
}

type leaveRoomParams struct {
	RoomID id.RoomID `json:"room_id"`
	Reason string    `json:"reason"`
}

type getReceiptsParams struct {
	RoomID   id.RoomID    `json:"room_id"`
	EventIDs []id.EventID `json:"event_ids"`
}

type muteRoomParams struct {
	RoomID id.RoomID `json:"room_id"`
	Muted  bool      `json:"muted"`
}
