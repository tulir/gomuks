// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func ParseResponse[T any](resp json.RawMessage, err error) (T, error) {
	var data T
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return data, fmt.Errorf("failed to parse response JSON: %w", err)
	}
	return data, nil
}

func (gr *GomuksRPC) GetState(ctx context.Context) (*jsoncmd.ClientState, error) {
	return ParseResponse[*jsoncmd.ClientState](gr.Request(ctx, jsoncmd.ReqGetState, nil))
}

func (gr *GomuksRPC) SendMessage(ctx context.Context, params *jsoncmd.SendMessageParams) (*database.Event, error) {
	return ParseResponse[*database.Event](gr.Request(ctx, jsoncmd.ReqSendMessage, params))
}

func (gr *GomuksRPC) SendEvent(ctx context.Context, params *jsoncmd.SendEventParams) (*database.Event, error) {
	return ParseResponse[*database.Event](gr.Request(ctx, jsoncmd.ReqSendEvent, params))
}

func (gr *GomuksRPC) ResendEvent(ctx context.Context, params *jsoncmd.ResendEventParams) (*database.Event, error) {
	return ParseResponse[*database.Event](gr.Request(ctx, jsoncmd.ReqResendEvent, params))
}

func (gr *GomuksRPC) ReportEvent(ctx context.Context, params *jsoncmd.ReportEventParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqReportEvent, params))
}

func (gr *GomuksRPC) RedactEvent(ctx context.Context, params *jsoncmd.RedactEventParams) (*mautrix.RespSendEvent, error) {
	return ParseResponse[*mautrix.RespSendEvent](gr.Request(ctx, jsoncmd.ReqRedactEvent, params))
}

func (gr *GomuksRPC) SetState(ctx context.Context, params *jsoncmd.SendStateEventParams) (id.EventID, error) {
	return ParseResponse[id.EventID](gr.Request(ctx, jsoncmd.ReqSetState, params))
}

func (gr *GomuksRPC) UpdateDelayedEvent(ctx context.Context, params *jsoncmd.UpdateDelayedEventParams) (*mautrix.RespUpdateDelayedEvent, error) {
	return ParseResponse[*mautrix.RespUpdateDelayedEvent](gr.Request(ctx, jsoncmd.ReqUpdateDelayedEvent, params))
}

func (gr *GomuksRPC) SetMembership(ctx context.Context, params *jsoncmd.SetMembershipParams) (any, error) {
	return ParseResponse[any](gr.Request(ctx, jsoncmd.ReqSetMembership, params))
}

func (gr *GomuksRPC) SetAccountData(ctx context.Context, params *jsoncmd.SetAccountDataParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqSetAccountData, params))
}

func (gr *GomuksRPC) MarkRead(ctx context.Context, params *jsoncmd.MarkReadParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqMarkRead, params))
}

func (gr *GomuksRPC) SetTyping(ctx context.Context, params *jsoncmd.SetTypingParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqSetTyping, params))
}

func (gr *GomuksRPC) GetProfile(ctx context.Context, params *jsoncmd.GetProfileParams) (*mautrix.RespUserProfile, error) {
	return ParseResponse[*mautrix.RespUserProfile](gr.Request(ctx, jsoncmd.ReqGetProfile, params))
}

func (gr *GomuksRPC) SetProfileField(ctx context.Context, params *jsoncmd.SetProfileFieldParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqSetProfileField, params))
}

func (gr *GomuksRPC) GetMutualRooms(ctx context.Context, params *jsoncmd.GetProfileParams) ([]id.RoomID, error) {
	return ParseResponse[[]id.RoomID](gr.Request(ctx, jsoncmd.ReqGetMutualRooms, params))
}

func (gr *GomuksRPC) TrackUserDevices(ctx context.Context, params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
	return ParseResponse[*jsoncmd.ProfileEncryptionInfo](gr.Request(ctx, jsoncmd.ReqTrackUserDevices, params))
}

func (gr *GomuksRPC) GetProfileEncryptionInfo(ctx context.Context, params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
	return ParseResponse[*jsoncmd.ProfileEncryptionInfo](gr.Request(ctx, jsoncmd.ReqGetProfileEncryptionInfo, params))
}

func (gr *GomuksRPC) GetEvent(ctx context.Context, params *jsoncmd.GetEventParams) (*database.Event, error) {
	return ParseResponse[*database.Event](gr.Request(ctx, jsoncmd.ReqGetEvent, params))
}

func (gr *GomuksRPC) GetRelatedEvents(ctx context.Context, params *jsoncmd.GetRelatedEventsParams) ([]*database.Event, error) {
	return ParseResponse[[]*database.Event](gr.Request(ctx, jsoncmd.ReqGetRelatedEvents, params))
}

func (gr *GomuksRPC) GetRoomState(ctx context.Context, params *jsoncmd.GetRoomStateParams) ([]*database.Event, error) {
	return ParseResponse[[]*database.Event](gr.Request(ctx, jsoncmd.ReqGetRoomState, params))
}

func (gr *GomuksRPC) GetSpecificRoomState(ctx context.Context, params *jsoncmd.GetSpecificRoomStateParams) ([]*database.Event, error) {
	return ParseResponse[[]*database.Event](gr.Request(ctx, jsoncmd.ReqGetSpecificRoomState, params))
}

func (gr *GomuksRPC) GetReceipts(ctx context.Context, params *jsoncmd.GetReceiptsParams) (map[id.EventID][]*database.Receipt, error) {
	return ParseResponse[map[id.EventID][]*database.Receipt](gr.Request(ctx, jsoncmd.ReqGetReceipts, params))
}

func (gr *GomuksRPC) Paginate(ctx context.Context, params *jsoncmd.PaginateParams) (*jsoncmd.PaginationResponse, error) {
	return ParseResponse[*jsoncmd.PaginationResponse](gr.Request(ctx, jsoncmd.ReqPaginate, params))
}

func (gr *GomuksRPC) PaginateServer(ctx context.Context, params *jsoncmd.PaginateParams) (*jsoncmd.PaginationResponse, error) {
	return ParseResponse[*jsoncmd.PaginationResponse](gr.Request(ctx, jsoncmd.ReqPaginateServer, params))
}

func (gr *GomuksRPC) GetRoomSummary(ctx context.Context, params *jsoncmd.JoinRoomParams) (*mautrix.RespRoomSummary, error) {
	return ParseResponse[*mautrix.RespRoomSummary](gr.Request(ctx, jsoncmd.ReqGetRoomSummary, params))
}

func (gr *GomuksRPC) JoinRoom(ctx context.Context, params *jsoncmd.JoinRoomParams) (*mautrix.RespJoinRoom, error) {
	return ParseResponse[*mautrix.RespJoinRoom](gr.Request(ctx, jsoncmd.ReqJoinRoom, params))
}

func (gr *GomuksRPC) KnockRoom(ctx context.Context, params *jsoncmd.JoinRoomParams) (*mautrix.RespKnockRoom, error) {
	return ParseResponse[*mautrix.RespKnockRoom](gr.Request(ctx, jsoncmd.ReqKnockRoom, params))
}

func (gr *GomuksRPC) LeaveRoom(ctx context.Context, params *jsoncmd.LeaveRoomParams) (*mautrix.RespLeaveRoom, error) {
	return ParseResponse[*mautrix.RespLeaveRoom](gr.Request(ctx, jsoncmd.ReqLeaveRoom, params))
}

func (gr *GomuksRPC) CreateRoom(ctx context.Context, params *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error) {
	return ParseResponse[*mautrix.RespCreateRoom](gr.Request(ctx, jsoncmd.ReqCreateRoom, params))
}

func (gr *GomuksRPC) MuteRoom(ctx context.Context, params *jsoncmd.MuteRoomParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqMuteRoom, params))
}

func (gr *GomuksRPC) EnsureGroupSessionShared(ctx context.Context, params *jsoncmd.EnsureGroupSessionSharedParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqEnsureGroupSessionShared, params))
}

func (gr *GomuksRPC) SendToDevice(ctx context.Context, params *jsoncmd.SendToDeviceParams) (*mautrix.RespSendToDevice, error) {
	return ParseResponse[*mautrix.RespSendToDevice](gr.Request(ctx, jsoncmd.ReqSendToDevice, params))
}

func (gr *GomuksRPC) ResolveAlias(ctx context.Context, params *jsoncmd.ResolveAliasParams) (*mautrix.RespAliasResolve, error) {
	return ParseResponse[*mautrix.RespAliasResolve](gr.Request(ctx, jsoncmd.ReqResolveAlias, params))
}

func (gr *GomuksRPC) RequestOpenIDToken(ctx context.Context) (*mautrix.RespOpenIDToken, error) {
	return ParseResponse[*mautrix.RespOpenIDToken](gr.Request(ctx, jsoncmd.ReqRequestOpenIDToken, nil))
}

func (gr *GomuksRPC) Logout(ctx context.Context) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqLogout, nil))
}

func (gr *GomuksRPC) Login(ctx context.Context, params *jsoncmd.LoginParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqLogin, params))
}

func (gr *GomuksRPC) LoginCustom(ctx context.Context, params *jsoncmd.LoginCustomParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqLoginCustom, params))
}

func (gr *GomuksRPC) Verify(ctx context.Context, params *jsoncmd.VerifyParams) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqVerify, params))
}

func (gr *GomuksRPC) DiscoverHomeserver(ctx context.Context, params *jsoncmd.DiscoverHomeserverParams) (*mautrix.ClientWellKnown, error) {
	return ParseResponse[*mautrix.ClientWellKnown](gr.Request(ctx, jsoncmd.ReqDiscoverHomeserver, params))
}

func (gr *GomuksRPC) GetLoginFlows(ctx context.Context, params *jsoncmd.GetLoginFlowsParams) (*mautrix.RespLoginFlows, error) {
	return ParseResponse[*mautrix.RespLoginFlows](gr.Request(ctx, jsoncmd.ReqGetLoginFlows, params))
}

func (gr *GomuksRPC) RegisterPush(ctx context.Context, params *database.PushRegistration) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqRegisterPush, params))
}

func (gr *GomuksRPC) ListenToDevice(ctx context.Context, listen bool) (bool, error) {
	return ParseResponse[bool](gr.Request(ctx, jsoncmd.ReqListenToDevice, &listen))
}

func (gr *GomuksRPC) GetTurnServers(ctx context.Context) (*mautrix.RespTurnServer, error) {
	return ParseResponse[*mautrix.RespTurnServer](gr.Request(ctx, jsoncmd.ReqGetTurnServers, nil))
}

func (gr *GomuksRPC) GetMediaConfig(ctx context.Context) (*mautrix.RespMediaConfig, error) {
	return ParseResponse[*mautrix.RespMediaConfig](gr.Request(ctx, jsoncmd.ReqGetMediaConfig, nil))
}

func (gr *GomuksRPC) Ping(ctx context.Context, params *jsoncmd.PingParams) (struct{}, error) {
	return ParseResponse[struct{}](gr.Request(ctx, jsoncmd.ReqPing, params))
}
