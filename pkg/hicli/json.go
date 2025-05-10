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
	"sync/atomic"

	"go.mau.fi/util/exerrors"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type JSONCommandCustom[T any] = jsoncmd.Container[T]
type JSONCommand = jsoncmd.Container[json.RawMessage]

type JSONEventHandler func(*JSONCommand)

var outgoingEventCounter atomic.Int64

func (jeh JSONEventHandler) HandleEvent(evt any) {
	data, err := json.Marshal(evt)
	if err != nil {
		panic(fmt.Errorf("failed to marshal event %T: %w", evt, err))
	}
	jeh(&JSONCommand{
		Command:   jsoncmd.EventTypeName(evt),
		RequestID: -outgoingEventCounter.Add(1),
		Data:      data,
	})
}

func (h *HiClient) State() *jsoncmd.ClientState {
	state := &jsoncmd.ClientState{}
	if acc := h.Account; acc != nil {
		state.IsLoggedIn = true
		state.UserID = acc.UserID
		state.DeviceID = acc.DeviceID
		state.HomeserverURL = acc.HomeserverURL
		state.IsVerified = h.Verified
	}
	return state
}

func (h *HiClient) dispatchCurrentState() {
	h.EventHandler(h.State())
}

func (h *HiClient) SubmitJSONCommand(ctx context.Context, req *JSONCommand) *JSONCommand {
	log := h.Log.With().Int64("request_id", req.RequestID).Stringer("command", req.Command).Logger()
	ctx, cancel := context.WithCancelCause(ctx)
	defer func() {
		cancel(nil)
		h.jsonRequestsLock.Lock()
		delete(h.jsonRequests, req.RequestID)
		h.jsonRequestsLock.Unlock()
	}()
	ctx = log.WithContext(ctx)
	h.jsonRequestsLock.Lock()
	h.jsonRequests[req.RequestID] = cancel
	h.jsonRequestsLock.Unlock()
	resp, err := h.handleJSONCommand(ctx, req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			causeErr := context.Cause(ctx)
			if causeErr != ctx.Err() {
				err = fmt.Errorf("%w: %w", err, causeErr)
			}
		}
		return &JSONCommand{
			Command:   jsoncmd.RespError,
			RequestID: req.RequestID,
			Data:      exerrors.Must(json.Marshal(err.Error())),
		}
	}
	var respData json.RawMessage
	respData, err = json.Marshal(resp)
	if err != nil {
		return &JSONCommand{
			Command:   jsoncmd.RespError,
			RequestID: req.RequestID,
			Data:      exerrors.Must(json.Marshal(fmt.Sprintf("failed to marshal response json: %v", err))),
		}
	}
	return &JSONCommand{
		Command:   jsoncmd.RespSuccess,
		RequestID: req.RequestID,
		Data:      respData,
	}
}
