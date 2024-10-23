// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"fmt"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

type hiSyncer HiClient

var _ mautrix.Syncer = (*hiSyncer)(nil)

type contextKey int

const (
	syncContextKey contextKey = iota
)

func (h *hiSyncer) ProcessResponse(ctx context.Context, resp *mautrix.RespSync, since string) error {
	c := (*HiClient)(h)
	ctx = context.WithValue(ctx, syncContextKey, &syncContext{evt: &SyncComplete{
		Rooms:     make(map[id.RoomID]*SyncRoom, len(resp.Rooms.Join)),
		LeftRooms: make([]id.RoomID, 0, len(resp.Rooms.Leave)),
	}})
	err := c.preProcessSyncResponse(ctx, resp, since)
	if err != nil {
		return err
	}
	err = c.DB.DoTxn(ctx, nil, func(ctx context.Context) error {
		return c.processSyncResponse(ctx, resp, since)
	})
	if err != nil {
		return err
	}
	c.postProcessSyncResponse(ctx, resp, since)
	return nil
}

func (h *hiSyncer) OnFailedSync(_ *mautrix.RespSync, err error) (time.Duration, error) {
	(*HiClient)(h).Log.Err(err).Msg("Sync failed, retrying in 1 second")
	return 1 * time.Second, nil
}

func (h *hiSyncer) GetFilterJSON(_ id.UserID) *mautrix.Filter {
	if !h.Verified {
		return &mautrix.Filter{
			Presence: mautrix.FilterPart{
				NotRooms: []id.RoomID{"*"},
			},
			Room: mautrix.RoomFilter{
				NotRooms: []id.RoomID{"*"},
			},
		}
	}
	return &mautrix.Filter{
		Presence: mautrix.FilterPart{
			NotRooms: []id.RoomID{"*"},
		},
		Room: mautrix.RoomFilter{
			State: mautrix.FilterPart{
				LazyLoadMembers: true,
			},
			Timeline: mautrix.FilterPart{
				Limit:           100,
				LazyLoadMembers: true,
			},
		},
	}
}

type hiStore HiClient

var _ mautrix.SyncStore = (*hiStore)(nil)

// Filter ID save and load are intentionally no-ops: we want to recreate filters when restarting syncing

func (h *hiStore) SaveFilterID(_ context.Context, _ id.UserID, _ string) error { return nil }
func (h *hiStore) LoadFilterID(_ context.Context, _ id.UserID) (string, error) { return "", nil }

func (h *hiStore) SaveNextBatch(ctx context.Context, userID id.UserID, nextBatchToken string) error {
	// This is intentionally a no-op: we don't want to save the next batch before processing the sync
	return nil
}

func (h *hiStore) LoadNextBatch(_ context.Context, userID id.UserID) (string, error) {
	if h.Account.UserID != userID {
		return "", fmt.Errorf("mismatching user ID")
	}
	return h.Account.NextBatch, nil
}
