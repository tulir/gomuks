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
	"bytes"
	"compress/flate"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog"

	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type flateProxy struct {
	lock   sync.Mutex
	target io.Writer
	fw     *flate.Writer
}

func (fp *flateProxy) Write(p []byte) (n int, err error) {
	if fp.target == nil {
		panic(errors.New("flateProxy: target not set"))
	}
	return fp.target.Write(p)
}

func writeRaw(ctx context.Context, conn *websocket.Conn, fp *flateProxy, data []byte) (int, error) {
	var n int
	msgType := websocket.MessageText
	if fp != nil {
		msgType = websocket.MessageBinary
	}
	writer, err := conn.Writer(ctx, msgType)
	if err != nil {
		return n, err
	}
	if fp != nil {
		var buf bytes.Buffer
		fp.lock.Lock()
		fp.target = &buf
		_, err = fp.fw.Write(data)
		fp.target = nil
		fp.lock.Unlock()
		if err != nil {
			return n, fmt.Errorf("failed to write data to flate writer: %w", err)
		}
		data = buf.Bytes()
	}
	n, err = writer.Write(data)
	if err != nil {
		return n, err
	}
	return n, writer.Close()
}

func writeCmd[T any](ctx context.Context, conn *websocket.Conn, fp *flateProxy, cmd *jsoncmd.Container[T]) error {
	msgType := websocket.MessageText
	if fp != nil {
		msgType = websocket.MessageBinary
	}
	writer, err := conn.Writer(ctx, msgType)
	if err != nil {
		return err
	}
	jsonWriter := writer
	if fp != nil {
		fp.lock.Lock()
		fp.target = writer
		jsonWriter = fp.fw
		defer func() {
			fp.target = nil
			fp.lock.Unlock()
		}()
	}
	err = json.NewEncoder(jsonWriter).Encode(&cmd)
	if err != nil {
		return fmt.Errorf("failed to encode command to websocket: %w", err)
	}
	if fp != nil {
		err = fp.fw.Flush()
		if err != nil {
			return fmt.Errorf("failed to flush flate writer: %w", err)
		}
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close websocket writer: %w", err)
	}
	return nil
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
	compress, _ := strconv.ParseInt(r.URL.Query().Get("compress"), 10, 64)
	log.Info().
		Int64("resume_from", resumeFrom).
		Int64("resume_run_id", resumeRunID).
		Int64("current_run_id", runID).
		Int64("compress", compress).
		Msg("Accepted new websocket connection")
	var fp *flateProxy
	if compress == 1 {
		fp = &flateProxy{}
		var err error
		fp.fw, err = flate.NewWriter(fp, flate.DefaultCompression)
		if err != nil {
			log.Err(err).Msg("Failed to create flate writer for websocket messages")
			_ = conn.Close(websocket.StatusInternalError, "Failed to create flate writer")
			return
		}
		log.Debug().Msg("Enabled flate compression for websocket messages")
	}
	conn.SetReadLimit(128 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = log.WithContext(ctx)
	var listenerID uint64
	evts := make(chan *BufferedEvent, 512)
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
	var resumeData []*BufferedEvent
	listenerID, resumeData = gmx.EventBuffer.Subscribe(resumeFrom, closeManually, func(evt *BufferedEvent) {
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
		err := writeCmd(ctx, conn, fp, &BufferedEvent{
			Command: jsoncmd.EventImageAuthToken,
			Data:    gmx.generateImageToken(1 * time.Hour),
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
			err := writeCmd(ctx, conn, fp, cmd)
			if err != nil {
				log.Err(err).Int64("req_id", cmd.RequestID).Msg("Failed to write outgoing event from resume data")
				return
			} else {
				log.Trace().Int64("req_id", cmd.RequestID).Msg("Sent outgoing event from resume data")
			}
		}
		if resumeData != nil {
			err := writeCmd(ctx, conn, fp, &hicli.JSONCommand{
				Command:   jsoncmd.EventInitComplete,
				RequestID: 0,
			})
			if err != nil {
				log.Err(err).Msg("Failed to send init done event to client")
				return
			}
		}
		resumeData = nil
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		ctxDone := ctx.Done()
		for {
			select {
			case cmd := <-evts:
				err := writeCmd(ctx, conn, fp, cmd)
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
		err := writeCmd(ctx, conn, fp, resp)
		if err != nil && ctx.Err() == nil {
			log.Err(err).Int64("req_id", cmd.RequestID).Msg("Failed to write response")
			closeOnce.Do(forceClose)
		} else {
			log.Trace().Int64("req_id", cmd.RequestID).Msg("Sent response to command")
		}
	}
	initErr := writeCmd(ctx, conn, fp, &jsoncmd.Container[*jsoncmd.RunData]{
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
	initErr = writeCmd(ctx, conn, fp, &jsoncmd.Container[*jsoncmd.ClientState]{
		Command: jsoncmd.EventClientState,
		Data:    gmx.Client.State(),
	})
	if initErr != nil {
		log.Err(initErr).Msg("Failed to write init client state message")
		return
	}
	initErr = writeCmd(ctx, conn, fp, &jsoncmd.Container[*jsoncmd.SyncStatus]{
		Command: jsoncmd.EventSyncStatus,
		Data:    gmx.Client.SyncStatus.Load(),
	})
	if initErr != nil {
		log.Err(initErr).Msg("Failed to write init sync status message")
		return
	}
	go sendImageAuthToken()
	if gmx.Client.IsLoggedIn() && !didResume {
		go gmx.sendInitialData(ctx, fp, conn)
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

func (gmx *Gomuks) sendInitialData(ctx context.Context, fp *flateProxy, conn *websocket.Conn) {
	log := zerolog.Ctx(ctx)
	var roomCount int
	var totalSize int
	for payload := range gmx.Client.GetInitialSync(ctx, 100) {
		roomCount += len(payload.Rooms)
		marshaledPayload, err := json.Marshal(&jsoncmd.Container[*jsoncmd.SyncComplete]{
			Command:   jsoncmd.EventSyncComplete,
			RequestID: 0,
			Data:      payload,
		})
		if err != nil {
			log.Err(err).Msg("Failed to marshal initial rooms to send to client")
			return
		}
		var n int
		n, err = writeRaw(ctx, conn, fp, marshaledPayload)
		if err != nil {
			log.Err(err).Msg("Failed to send initial rooms to client")
			return
		}
		totalSize += n
	}
	if ctx.Err() != nil {
		return
	}
	err := writeCmd(ctx, conn, fp, &hicli.JSONCommand{
		Command:   jsoncmd.EventInitComplete,
		RequestID: 0,
	})
	if err != nil {
		log.Err(err).Msg("Failed to send initial rooms done event to client")
		return
	}
	log.Info().
		Int("room_count", roomCount).
		Int("total_payload_size", totalSize).
		Msg("Sent initial rooms to client")
}
