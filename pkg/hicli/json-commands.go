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
			return h.Send(ctx, params.RoomID, params.EventType, params.Content)
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
			return h.SetState(ctx, params.RoomID, params.EventType, params.StateKey, params.Content)
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
	case "get_profile_view":
		return unmarshalAndCall(req.Data, func(params *getProfileViewParams) (*ProfileViewData, error) {
			return h.GetProfileView(ctx, params.RoomID, params.UserID)
		})
	case "get_event":
		return unmarshalAndCall(req.Data, func(params *getEventParams) (*database.Event, error) {
			return h.GetEvent(ctx, params.RoomID, params.EventID)
		})
	case "get_events_by_rowids":
		return unmarshalAndCall(req.Data, func(params *getEventsByRowIDsParams) ([]*database.Event, error) {
			return h.GetEventsByRowIDs(ctx, params.RowIDs)
		})
	case "get_room_state":
		return unmarshalAndCall(req.Data, func(params *getRoomStateParams) ([]*database.Event, error) {
			return h.GetRoomState(ctx, params.RoomID, params.IncludeMembers, params.FetchMembers, params.Refetch)
		})
	case "get_specific_room_state":
		return unmarshalAndCall(req.Data, func(params *getSpecificRoomStateParams) ([]*database.Event, error) {
			return h.DB.CurrentState.GetMany(ctx, params.Keys)
		})
	case "paginate":
		return unmarshalAndCall(req.Data, func(params *paginateParams) (*PaginationResponse, error) {
			return h.Paginate(ctx, params.RoomID, params.MaxTimelineID, params.Limit)
		})
	case "paginate_server":
		return unmarshalAndCall(req.Data, func(params *paginateParams) (*PaginationResponse, error) {
			return h.PaginateServer(ctx, params.RoomID, params.Limit)
		})
	case "ensure_group_session_shared":
		return unmarshalAndCall(req.Data, func(params *ensureGroupSessionSharedParams) (bool, error) {
			return true, h.EnsureGroupSessionShared(ctx, params.RoomID)
		})
	case "resolve_alias":
		return unmarshalAndCall(req.Data, func(params *resolveAliasParams) (*mautrix.RespAliasResolve, error) {
			return h.Client.ResolveAlias(ctx, params.Alias)
		})
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
	RoomID    id.RoomID       `json:"room_id"`
	EventType event.Type      `json:"type"`
	Content   json.RawMessage `json:"content"`
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

type getProfileViewParams struct {
	RoomID id.RoomID `json:"room_id"`
	UserID id.UserID `json:"user_id"`
}

type getEventParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
}

type getEventsByRowIDsParams struct {
	RowIDs []database.EventRowID `json:"row_ids"`
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
