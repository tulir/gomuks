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
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (h *HiClient) handleJSONCommand(ctx context.Context, req *JSONCommand) (any, error) {
	switch req.Command {
	case jsoncmd.ReqGetState:
		return h.State(), nil
	case jsoncmd.ReqCancel:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.CancelRequestParams) (bool, error) {
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
	case jsoncmd.ReqSendMessage:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SendMessageParams) (*database.Event, error) {
			return h.SendMessage(ctx, params.RoomID, params.BaseContent, params.Extra, params.Text, params.RelatesTo, params.Mentions, params.URLPreviews)
		})
	case jsoncmd.ReqSendEvent:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SendEventParams) (*database.Event, error) {
			return h.Send(ctx, params.RoomID, params.EventType, params.Content, params.DisableEncryption, params.Synchronous)
		})
	case jsoncmd.ReqResendEvent:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.ResendEventParams) (*database.Event, error) {
			return h.Resend(ctx, params.TransactionID)
		})
	case jsoncmd.ReqReportEvent:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.ReportEventParams) (bool, error) {
			return true, h.Client.ReportEvent(ctx, params.RoomID, params.EventID, params.Reason)
		})
	case jsoncmd.ReqRedactEvent:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.RedactEventParams) (*mautrix.RespSendEvent, error) {
			return h.Client.RedactEvent(ctx, params.RoomID, params.EventID, mautrix.ReqRedact{
				Reason: params.Reason,
			})
		})
	case jsoncmd.ReqSetState:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SendStateEventParams) (id.EventID, error) {
			return h.SetState(ctx, params.RoomID, params.EventType, params.StateKey, params.Content, mautrix.ReqSendEvent{
				UnstableDelay: time.Duration(params.DelayMS) * time.Millisecond,
			})
		})
	case jsoncmd.ReqUpdateDelayedEvent:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.UpdateDelayedEventParams) (*mautrix.RespUpdateDelayedEvent, error) {
			return h.Client.UpdateDelayedEvent(ctx, &mautrix.ReqUpdateDelayedEvent{
				DelayID: params.DelayID,
				Action:  params.Action,
			})
		})
	case jsoncmd.ReqSetMembership:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SetMembershipParams) (any, error) {
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
	case jsoncmd.ReqSetAccountData:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SetAccountDataParams) (bool, error) {
			if params.RoomID != "" {
				return true, h.Client.SetRoomAccountData(ctx, params.RoomID, params.Type, params.Content)
			} else {
				return true, h.Client.SetAccountData(ctx, params.Type, params.Content)
			}
		})
	case jsoncmd.ReqMarkRead:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.MarkReadParams) (bool, error) {
			return true, h.MarkRead(ctx, params.RoomID, params.EventID, params.ReceiptType)
		})
	case jsoncmd.ReqSetTyping:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SetTypingParams) (bool, error) {
			return true, h.SetTyping(ctx, params.RoomID, time.Duration(params.Timeout)*time.Millisecond)
		})
	case jsoncmd.ReqGetProfile:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetProfileParams) (*mautrix.RespUserProfile, error) {
			return h.Client.GetProfile(ctx, params.UserID)
		})
	case jsoncmd.ReqSetProfileField:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SetProfileFieldParams) (bool, error) {
			return true, h.Client.UnstableSetProfileField(ctx, params.Field, params.Value)
		})
	case jsoncmd.ReqGetMutualRooms:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetProfileParams) ([]id.RoomID, error) {
			return h.GetMutualRooms(ctx, params.UserID)
		})
	case jsoncmd.ReqTrackUserDevices:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
			err := h.TrackUserDevices(ctx, params.UserID)
			if err != nil {
				return nil, err
			}
			return h.GetProfileEncryptionInfo(ctx, params.UserID)
		})
	case jsoncmd.ReqGetProfileEncryptionInfo:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
			return h.GetProfileEncryptionInfo(ctx, params.UserID)
		})
	case jsoncmd.ReqGetEvent:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetEventParams) (*database.Event, error) {
			if params.Unredact {
				return h.GetUnredactedEvent(ctx, params.RoomID, params.EventID)
			}
			return h.GetEvent(ctx, params.RoomID, params.EventID)
		})
	case jsoncmd.ReqGetRelatedEvents:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetRelatedEventsParams) ([]*database.Event, error) {
			return h.DB.Event.GetRelatedEvents(ctx, params.RoomID, params.EventID, params.RelationType)
		})
	case jsoncmd.ReqGetRoomState:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetRoomStateParams) ([]*database.Event, error) {
			return h.GetRoomState(ctx, params.RoomID, params.IncludeMembers, params.FetchMembers, params.Refetch)
		})
	case jsoncmd.ReqGetSpecificRoomState:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetSpecificRoomStateParams) ([]*database.Event, error) {
			return h.DB.CurrentState.GetMany(ctx, params.Keys)
		})
	case jsoncmd.ReqGetReceipts:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetReceiptsParams) (map[id.EventID][]*database.Receipt, error) {
			return h.GetReceipts(ctx, params.RoomID, params.EventIDs)
		})
	case jsoncmd.ReqPaginate:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.PaginateParams) (*jsoncmd.PaginationResponse, error) {
			return h.Paginate(ctx, params.RoomID, params.MaxTimelineID, params.Limit)
		})
	case jsoncmd.ReqPaginateServer:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.PaginateParams) (*jsoncmd.PaginationResponse, error) {
			return h.PaginateServer(ctx, params.RoomID, params.Limit)
		})
	case jsoncmd.ReqGetRoomSummary:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.JoinRoomParams) (*mautrix.RespRoomSummary, error) {
			return h.Client.GetRoomSummary(ctx, params.RoomIDOrAlias, params.Via...)
		})
	case jsoncmd.ReqJoinRoom:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.JoinRoomParams) (*mautrix.RespJoinRoom, error) {
			return h.Client.JoinRoom(ctx, params.RoomIDOrAlias, &mautrix.ReqJoinRoom{
				Via:    params.Via,
				Reason: params.Reason,
			})
		})
	case jsoncmd.ReqKnockRoom:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.JoinRoomParams) (*mautrix.RespKnockRoom, error) {
			return h.Client.KnockRoom(ctx, params.RoomIDOrAlias, &mautrix.ReqKnockRoom{
				Via:    params.Via,
				Reason: params.Reason,
			})
		})
	case jsoncmd.ReqLeaveRoom:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.LeaveRoomParams) (*mautrix.RespLeaveRoom, error) {
			return h.Client.LeaveRoom(ctx, params.RoomID, &mautrix.ReqLeave{Reason: params.Reason})
		})
	case jsoncmd.ReqCreateRoom:
		return unmarshalAndCall(req.Data, func(params *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error) {
			return h.Client.CreateRoom(ctx, params)
		})
	case jsoncmd.ReqMuteRoom:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.MuteRoomParams) (bool, error) {
			if params.Muted {
				return true, h.Client.PutPushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID), &mautrix.ReqPutPushRule{
					Actions: []pushrules.PushActionType{},
				})
			} else {
				return false, h.Client.DeletePushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID))
			}
		})
	case jsoncmd.ReqEnsureGroupSessionShared:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.EnsureGroupSessionSharedParams) (bool, error) {
			return true, h.EnsureGroupSessionShared(ctx, params.RoomID)
		})
	case jsoncmd.ReqSendToDevice:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.SendToDeviceParams) (*mautrix.RespSendToDevice, error) {
			params.EventType.Class = event.ToDeviceEventType
			return h.SendToDevice(ctx, params.EventType, params.ReqSendToDevice, params.Encrypted)
		})
	case jsoncmd.ReqResolveAlias:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.ResolveAliasParams) (*mautrix.RespAliasResolve, error) {
			return h.Client.ResolveAlias(ctx, params.Alias)
		})
	case jsoncmd.ReqRequestOpenIDToken:
		return h.Client.RequestOpenIDToken(ctx)
	case jsoncmd.ReqLogout:
		if h.LogoutFunc == nil {
			return nil, errors.New("logout not supported")
		}
		return true, h.LogoutFunc(ctx)
	case jsoncmd.ReqLogin:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.LoginParams) (bool, error) {
			return true, h.LoginPassword(ctx, params.HomeserverURL, params.Username, params.Password)
		})
	case jsoncmd.ReqLoginCustom:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.LoginCustomParams) (bool, error) {
			var err error
			h.Client.HomeserverURL, err = url.Parse(params.HomeserverURL)
			if err != nil {
				return false, err
			}
			return true, h.Login(ctx, params.Request)
		})
	case jsoncmd.ReqVerify:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.VerifyParams) (bool, error) {
			return true, h.Verify(ctx, params.RecoveryKey)
		})
	case jsoncmd.ReqDiscoverHomeserver:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.DiscoverHomeserverParams) (*mautrix.ClientWellKnown, error) {
			_, homeserver, err := params.UserID.Parse()
			if err != nil {
				return nil, err
			}
			return mautrix.DiscoverClientAPI(ctx, homeserver)
		})
	case jsoncmd.ReqGetLoginFlows:
		return unmarshalAndCall(req.Data, func(params *jsoncmd.GetLoginFlowsParams) (*mautrix.RespLoginFlows, error) {
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
	case jsoncmd.ReqRegisterPush:
		return unmarshalAndCall(req.Data, func(params *database.PushRegistration) (bool, error) {
			return true, h.DB.PushRegistration.Put(ctx, params)
		})
	case jsoncmd.ReqListenToDevice:
		return unmarshalAndCall(req.Data, func(listen *bool) (bool, error) {
			return h.ToDeviceInSync.Swap(*listen), nil
		})
	case jsoncmd.ReqGetTurnServers:
		return h.Client.TurnServer(ctx)
	case jsoncmd.ReqGetMediaConfig:
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
