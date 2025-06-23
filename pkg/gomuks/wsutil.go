// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
	"compress/flate"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"sync"

	"github.com/coder/websocket"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type flateProxy struct {
	lock   sync.Mutex
	target io.Writer
	fw     *flate.Writer
}

func (fp *flateProxy) Write(p []byte) (n int, err error) {
	if fp.target == nil {
		return 0, errors.New("flateProxy: target not set")
	}
	return fp.target.Write(p)
}

type sizeWriter struct {
	n int
	w io.Writer
}

func (sm *sizeWriter) Write(p []byte) (n int, err error) {
	n, err = sm.w.Write(p)
	sm.n += n
	return
}

func writeCmd[T any](
	ctx context.Context,
	conn *websocket.Conn,
	fp *flateProxy,
	cmd *jsoncmd.Container[T],
) error {
	_, err := writeCmdWithExtra(ctx, conn, fp, cmd, nil)
	return err
}

func writeCmdWithExtra[T any](
	ctx context.Context,
	conn *websocket.Conn,
	fp *flateProxy,
	cmd *jsoncmd.Container[T],
	extra iter.Seq[*jsoncmd.Container[T]],
) (int, error) {
	msgType := websocket.MessageText
	if fp != nil {
		msgType = websocket.MessageBinary
	}
	wsWriter, err := conn.Writer(ctx, msgType)
	if err != nil {
		return 0, err
	}
	writer := &sizeWriter{w: wsWriter}
	var jsonWriter io.Writer = writer
	if fp != nil {
		fp.lock.Lock()
		fp.target = writer
		jsonWriter = fp.fw
		defer func() {
			fp.target = nil
			fp.lock.Unlock()
		}()
	}
	jsonEnc := json.NewEncoder(jsonWriter)
	err = jsonEnc.Encode(&cmd)
	if err != nil {
		return writer.n, fmt.Errorf("failed to encode command to websocket: %w", err)
	}
	if extra != nil && msgType == websocket.MessageBinary {
		const preferredMaxFrameSize = 256 * 1024
		for extraCmd := range extra {
			err = jsonEnc.Encode(&extraCmd)
			if err != nil {
				return writer.n, fmt.Errorf("failed to encode command to websocket: %w", err)
			}
			if writer.n > preferredMaxFrameSize {
				break
			}
		}
	}
	if fp != nil {
		err = fp.fw.Flush()
		if err != nil {
			return writer.n, fmt.Errorf("failed to flush flate writer: %w", err)
		}
	}
	err = wsWriter.Close()
	if err != nil {
		return writer.n, fmt.Errorf("failed to close websocket writer: %w", err)
	}
	return writer.n, nil
}

func sliceToChan[T any](s []T) <-chan T {
	ch := make(chan T, len(s))
	for _, val := range s {
		ch <- val
	}
	close(ch)
	return ch
}

func chanToSeq[T any](ch <-chan T) iter.Seq[T] {
	return func(yield func(event T) bool) {
		for {
			select {
			case val, ok := <-ch:
				if !ok || !yield(val) {
					return
				}
			default:
				return
			}
		}
	}
}
