// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog"
	"go.mau.fi/util/exerrors"

	"go.mau.fi/gomuks/pkg/hicli"
)

func writeCmd(ctx context.Context, conn *websocket.Conn, cmd *hicli.JSONCommand) error {
	writer, err := conn.Writer(ctx, websocket.MessageText)
	if err != nil {
		return err
	}
	err = json.NewEncoder(writer).Encode(&cmd)
	if err != nil {
		return err
	}
	return writer.Close()
}

const (
	StatusEventsStuck = 4001
	StatusPingTimeout = 4002
)

var emptyObject = json.RawMessage("{}")

func (gmx *Gomuks) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	var conn *websocket.Conn
	log := zerolog.Ctx(r.Context())
	recoverPanic := func(context string) bool {
		err := recover()
		if err != nil {
			logEvt := log.Error().
				Bytes(zerolog.ErrorStackFieldName, debug.Stack()).
				Str("goroutine", context)
			if realErr, ok := err.(error); ok {
				logEvt = logEvt.Err(realErr)
			} else {
				logEvt = logEvt.Any(zerolog.ErrorFieldName, err)
			}
			logEvt.Msg("Panic in websocket handler")
			return true
		}
		return false
	}
	defer recoverPanic("read loop")

	conn, acceptErr := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*"},
	})
	if acceptErr != nil {
		log.Warn().Err(acceptErr).Msg("Failed to accept websocket connection")
		return
	}
	log.Info().Msg("Accepted new websocket connection")
	conn.SetReadLimit(128 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = log.WithContext(ctx)
	unsubscribe := func() {}
	evts := make(chan *hicli.JSONCommand, 32)
	forceClose := func() {
		cancel()
		unsubscribe()
		_ = conn.CloseNow()
		close(evts)
	}
	var closeOnce sync.Once
	defer closeOnce.Do(forceClose)
	closeManually := func(statusCode websocket.StatusCode, reason string) {
		log.Debug().Stringer("status_code", statusCode).Str("reason", reason).Msg("Closing connection manually")
		_ = conn.Close(statusCode, reason)
		closeOnce.Do(forceClose)
	}
	unsubscribe = gmx.SubscribeEvents(closeManually, func(evt *hicli.JSONCommand) {
		if ctx.Err() != nil {
			return
		}
		select {
		case evts <- evt:
		default:
			log.Warn().Msg("Event queue full, closing connection")
			cancel()
			go func() {
				defer recoverPanic("closing connection after error in event handler")
				_ = conn.Close(StatusEventsStuck, "Event queue full")
				closeOnce.Do(forceClose)
			}()
		}
	})

	lastDataReceived := &atomic.Int64{}
	lastDataReceived.Store(time.Now().UnixMilli())
	const RecvTimeout = 60 * time.Second
	lastImageAuthTokenSent := time.Now()
	sendImageAuthToken := func() {
		err := writeCmd(ctx, conn, &hicli.JSONCommand{
			Command: "image_auth_token",
			Data:    exerrors.Must(json.Marshal(gmx.generateImageToken())),
		})
		if err != nil {
			log.Err(err).Msg("Failed to write image auth token message")
			return
		}
	}
	go func() {
		defer recoverPanic("event loop")
		defer closeOnce.Do(forceClose)
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		ctxDone := ctx.Done()
		for {
			select {
			case cmd := <-evts:
				err := writeCmd(ctx, conn, cmd)
				if err != nil {
					log.Err(err).Int64("req_id", cmd.RequestID).Msg("Failed to write outgoing event")
					return
				} else {
					log.Trace().Int64("req_id", cmd.RequestID).Msg("Sent outgoing event")
				}
			case <-ticker.C:
				if time.Since(lastImageAuthTokenSent) > 30*time.Minute {
					sendImageAuthToken()
					lastImageAuthTokenSent = time.Now()
				}
				if time.Now().UnixMilli()-lastDataReceived.Load() > RecvTimeout.Milliseconds() {
					log.Warn().Msg("No data received in a minute, closing connection")
					_ = conn.Close(StatusPingTimeout, "Ping timeout")
					return
				}
			case <-ctxDone:
				return
			}
		}
	}()
	submitCmd := func(cmd *hicli.JSONCommand) {
		defer func() {
			if recoverPanic("command handler") {
				_ = conn.Close(websocket.StatusInternalError, "Command handler panicked")
				closeOnce.Do(forceClose)
			}
		}()
		if cmd.Data == nil {
			cmd.Data = emptyObject
		}
		log.Trace().
			Int64("req_id", cmd.RequestID).
			Str("command", cmd.Command).
			RawJSON("data", cmd.Data).
			Msg("Received command")
		resp := gmx.Client.SubmitJSONCommand(ctx, cmd)
		if ctx.Err() != nil {
			return
		}
		err := writeCmd(ctx, conn, resp)
		if err != nil && ctx.Err() == nil {
			log.Err(err).Int64("req_id", cmd.RequestID).Msg("Failed to write response")
			closeOnce.Do(forceClose)
		} else {
			log.Trace().Int64("req_id", cmd.RequestID).Msg("Sent response to command")
		}
	}
	initData, initErr := json.Marshal(gmx.Client.State())
	if initErr != nil {
		log.Err(initErr).Msg("Failed to marshal init message")
		return
	}
	initErr = writeCmd(ctx, conn, &hicli.JSONCommand{
		Command: "client_state",
		Data:    initData,
	})
	if initErr != nil {
		log.Err(initErr).Msg("Failed to write init message")
		return
	}
	go sendImageAuthToken()
	if gmx.Client.IsLoggedIn() {
		go gmx.sendInitialData(ctx, conn)
	}
	log.Debug().Msg("Connection initialization complete")
	var closeErr websocket.CloseError
	for {
		msgType, reader, err := conn.Reader(ctx)
		if err != nil {
			if errors.As(err, &closeErr) {
				log.Debug().
					Stringer("status_code", closeErr.Code).
					Str("reason", closeErr.Reason).
					Msg("Connection closed")
			} else {
				log.Err(err).Msg("Failed to read message")
			}
			return
		} else if msgType != websocket.MessageText {
			log.Error().Stringer("message_type", msgType).Msg("Unexpected message type")
			_ = conn.Close(websocket.StatusUnsupportedData, "Non-text message")
			return
		}
		lastDataReceived.Store(time.Now().UnixMilli())
		var cmd hicli.JSONCommand
		err = json.NewDecoder(reader).Decode(&cmd)
		if err != nil {
			log.Err(err).Msg("Failed to parse message")
			_ = conn.Close(websocket.StatusUnsupportedData, "Invalid JSON")
			return
		}
		go submitCmd(&cmd)
	}
}

func (gmx *Gomuks) sendInitialData(ctx context.Context, conn *websocket.Conn) {
	log := zerolog.Ctx(ctx)
	var roomCount int
	for payload := range gmx.Client.GetInitialSync(ctx, 100) {
		roomCount += len(payload.Rooms)
		marshaledPayload, err := json.Marshal(&payload)
		if err != nil {
			log.Err(err).Msg("Failed to marshal initial rooms to send to client")
			return
		}
		err = writeCmd(ctx, conn, &hicli.JSONCommand{
			Command:   "sync_complete",
			RequestID: 0,
			Data:      marshaledPayload,
		})
		if err != nil {
			log.Err(err).Msg("Failed to send initial rooms to client")
			return
		}
	}
	log.Info().Int("room_count", roomCount).Msg("Sent initial rooms to client")
}
