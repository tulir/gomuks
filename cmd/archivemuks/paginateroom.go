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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

var fileNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

func findLastRowID(file *os.File) (database.TimelineRowID, error) {
	const chunkSize = 2048
	info, err := file.Stat()
	if err != nil {
		return 0, err
	} else if info.Size() == 0 {
		return 0, nil
	}
	buf := make([]byte, chunkSize)
	ptr := info.Size() - chunkSize
	isFirstChunk := true
	lastLineStart := 0
	for {
		if ptr < 0 {
			ptr = 0
		}
		buf = buf[:chunkSize]
		n, err := file.ReadAt(buf, ptr)
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		buf = buf[:n]
		if isFirstChunk {
			buf = bytes.TrimRight(buf, "\n")
		}
		idx := bytes.LastIndexByte(buf, '\n')
		if idx >= 0 {
			lastLineStart = int(ptr) + idx + 1
			break
		}
		if ptr <= 0 {
			break
		}
		ptr -= chunkSize
		isFirstChunk = false
	}
	_, err = file.Seek(int64(lastLineStart), io.SeekStart)
	if err != nil {
		return 0, err
	}
	var dbEvent *database.Event
	err = json.NewDecoder(file).Decode(&dbEvent)
	if err != nil {
		return 0, err
	}
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	return dbEvent.TimelineRowID, nil
}

func paginateRoom(ctx context.Context, roomID id.RoomID) {
	meta, ok := roomMeta[roomID]
	if !ok {
		fmt.Println("Room", roomID, "not found")
		return
	}
	name := ptr.Val(meta.Name)
	if name == "" {
		name = roomID.String()
	}
	fileName := fileNameSanitizer.ReplaceAllString(name, "_") + ".json"
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to open file")
		return
	}
	maxTimelineID, err := findLastRowID(file)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to read last event row ID from file")
		return
	}
	fmt.Printf("Paginating room %s (%s) into %s\n", roomID, name, fileName)
	if maxTimelineID != 0 {
		fmt.Println("Continuing from row ID", maxTimelineID)
	} else {
		fmt.Println("File is empty, starting from the beginning")
	}
	enc := json.NewEncoder(file)
	hasMore := true
	for hasMore {
		fmt.Println("Sending pagination request", roomID, maxTimelineID)
		resp, err := cli.Paginate(ctx, &jsoncmd.PaginateParams{
			RoomID:        roomID,
			MaxTimelineID: maxTimelineID,
			Limit:         100,
		})
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to paginate room")
			if ctx.Err() != nil {
				return
			}
			fmt.Println("Retrying in 20 seconds")
			select {
			case <-ctx.Done():
				return
			case <-time.After(20 * time.Second):
			}
			continue
		}
		fmt.Println("Got", len(resp.Events), "more events, have more:", resp.HasMore)
		hasMore = resp.HasMore
		for _, evt := range resp.Events {
			if evt.StateKey != nil {
				continue
			}
			err = enc.Encode(evt)
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("Failed to write event to file")
				return
			}
		}
		if len(resp.Events) > 0 {
			maxTimelineID = resp.Events[len(resp.Events)-1].TimelineRowID
		}
		if resp.FromServer {
			// Wait for the decryption queue
			// TODO remove this after the deadlock is fixed
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
			}
		}
	}
}
