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

package gomuks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog"
	"go.mau.fi/util/exerrors"

	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func writeCmd[T any](ctx context.Context, conn *websocket.Conn, cmd *jsoncmd.Container[T]) error {
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

var runID = time.Now().UnixNano()

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
		OriginPatterns: gmx.Config.Web.OriginPatterns,
	})
	if acceptErr != nil {
		log.Warn().Err(acceptErr).Msg("Failed to accept websocket connection")
		return
	}
	resumeFrom, _ := strconv.ParseInt(r.URL.Query().Get("last_received_event"), 10, 64)
	resumeRunID, _ := strconv.ParseInt(r.URL.Query().Get("run_id"), 10, 64)
	log.Info().
		Int64("resume_from", resumeFrom).
		Int64("resume_run_id", resumeRunID).
		Int64("current_run_id", runID).
		Msg("Accepted new websocket connection")
	conn.SetReadLimit(128 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = log.WithContext(ctx)
	var listenerID uint64
	evts := make(chan *hicli.JSONCommand, 32)
	forceClose := func() {
		cancel()
		if listenerID != 0 {
			gmx.EventBuffer.Unsubscribe(listenerID)
		}
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
	if resumeRunID != runID {
		resumeFrom = 0
	}
	var resumeData []*hicli.JSONCommand
	listenerID, resumeData = gmx.EventBuffer.Subscribe(resumeFrom, closeManually, func(evt *hicli.JSONCommand) {
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
	didResume := resumeData != nil

	lastDataReceived := &atomic.Int64{}
	lastDataReceived.Store(time.Now().UnixMilli())
	const RecvTimeout = 60 * time.Second
	lastImageAuthTokenSent := time.Now()
	sendImageAuthToken := func() {
		err := writeCmd(ctx, conn, &hicli.JSONCommand{
			Command: jsoncmd.EventImageAuthToken,
			Data:    exerrors.Must(json.Marshal(gmx.generateImageToken(1 * time.Hour))),
		})
		if err != nil {
			log.Err(err).Msg("Failed to write image auth token message")
			return
		}
	}
	go func() {
		defer recoverPanic("event loop")
		defer closeOnce.Do(forceClose)
		for _, cmd := range resumeData {
			err := writeCmd(ctx, conn, cmd)
			if err != nil {
				log.Err(err).Int64("req_id", cmd.RequestID).Msg("Failed to write outgoing event from resume data")
				return
			} else {
				log.Trace().Int64("req_id", cmd.RequestID).Msg("Sent outgoing event from resume data")
			}
		}
		resumeData = nil
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
			Stringer("command", cmd.Command).
			RawJSON("data", cmd.Data).
			Msg("Received command")
		var resp *hicli.JSONCommand
		if cmd.Command == jsoncmd.ReqPing {
			resp = &hicli.JSONCommand{
				Command:   jsoncmd.RespPong,
				RequestID: cmd.RequestID,
			}
			var pingData jsoncmd.PingParams
			err := json.Unmarshal(cmd.Data, &pingData)
			if err != nil {
				log.Err(err).Msg("Failed to parse ping data")
			} else if pingData.LastReceivedID != 0 {
				gmx.EventBuffer.SetLastAckedID(listenerID, pingData.LastReceivedID)
			}
		} else {
			resp = gmx.Client.SubmitJSONCommand(ctx, cmd)
		}
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
	initErr := writeCmd(ctx, conn, &hicli.JSONCommandCustom[*jsoncmd.RunData]{
		Command: jsoncmd.EventRunID,
		Data: &jsoncmd.RunData{
			RunID: strconv.FormatInt(runID, 10),
			ETag:  gmx.frontendETag,
		},
	})
	if initErr != nil {
		log.Err(initErr).Msg("Failed to write init client state message")
		return
	}
	initErr = writeCmd(ctx, conn, &hicli.JSONCommandCustom[*jsoncmd.ClientState]{
		Command: jsoncmd.EventClientState,
		Data:    gmx.Client.State(),
	})
	if initErr != nil {
		log.Err(initErr).Msg("Failed to write init client state message")
		return
	}
	initErr = writeCmd(ctx, conn, &hicli.JSONCommandCustom[*jsoncmd.SyncStatus]{
		Command: jsoncmd.EventSyncStatus,
		Data:    gmx.Client.SyncStatus.Load(),
	})
	if initErr != nil {
		log.Err(initErr).Msg("Failed to write init sync status message")
		return
	}
	go sendImageAuthToken()
	if gmx.Client.IsLoggedIn() && !didResume {
		go gmx.sendInitialData(ctx, conn)
	}
	log.Debug().Bool("did_resume", didResume).Msg("Connection initialization complete")
	var closeErr websocket.CloseError
	for {
		msgType, reader, err := conn.Reader(ctx)
		if err != nil {
			if errors.As(err, &closeErr) {
				log.Debug().
					Stringer("status_code", closeErr.Code).
					Str("reason", closeErr.Reason).
					Msg("Connection closed")
				if closeErr.Code == websocket.StatusGoingAway {
					gmx.EventBuffer.ClearListenerLastAckedID(listenerID)
				}
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
		data, _ := io.ReadAll(reader)
		if len(data) > 0 {
			log.Warn().
				Bytes("data", data).
				Msg("Unexpected data in websocket reader")
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
			Command:   jsoncmd.EventSyncComplete,
			RequestID: 0,
			Data:      marshaledPayload,
		})
		if err != nil {
			log.Err(err).Msg("Failed to send initial rooms to client")
			return
		}
	}
	if ctx.Err() != nil {
		return
	}
	err := writeCmd(ctx, conn, &hicli.JSONCommand{
		Command:   jsoncmd.EventInitComplete,
		RequestID: 0,
	})
	if err != nil {
		log.Err(err).Msg("Failed to send initial rooms done event to client")
		return
	}
	log.Info().Int("room_count", roomCount).Msg("Sent initial rooms to client")
}
