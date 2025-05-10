// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"errors"
	"slices"

	"github.com/rs/zerolog"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

const MutualRoomsBatchLimit = 5

func (h *HiClient) GetMutualRooms(ctx context.Context, userID id.UserID) (output []id.RoomID, err error) {
	var nextBatch string
	for i := 0; i < MutualRoomsBatchLimit; i++ {
		mutualRooms, err := h.Client.GetMutualRooms(ctx, userID, mautrix.ReqMutualRooms{From: nextBatch})
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Str("from_batch_token", nextBatch).Msg("Failed to get mutual rooms")
			return nil, err
		}
		output = append(output, mutualRooms.Joined...)
		nextBatch = mutualRooms.NextBatch
		if nextBatch == "" {
			break
		}
	}
	slices.Sort(output)
	output = slices.Compact(output)
	return
}

func (h *HiClient) GetProfileEncryptionInfo(ctx context.Context, userID id.UserID) (*jsoncmd.ProfileEncryptionInfo, error) {
	var resp jsoncmd.ProfileEncryptionInfo
	log := zerolog.Ctx(ctx)
	cachedDevices, err := h.Crypto.GetCachedDevices(ctx, userID)
	if errors.Is(err, crypto.ErrUserNotTracked) {
		return &resp, nil
	} else if err != nil {
		log.Err(err).Msg("Failed to get cached devices")
		return nil, err
	}
	resp.DevicesTracked = true
	if cachedDevices.MasterKey != nil {
		resp.MasterKey = cachedDevices.MasterKey.Key.Fingerprint()
		resp.FirstMasterKey = cachedDevices.MasterKey.First.Fingerprint()
		if !cachedDevices.HasValidSelfSigningKey {
			resp.Errors = append(resp.Errors, "Self-signing key is not signed by master key")
		}
	} else {
		resp.Errors = append(resp.Errors, "Cross-signing keys not found")
	}
	resp.UserTrusted = cachedDevices.MasterKeySignedByUs
	resp.Devices = make([]*jsoncmd.ProfileDevice, len(cachedDevices.Devices))
	for i, dev := range cachedDevices.Devices {
		resp.Devices[i] = &jsoncmd.ProfileDevice{
			DeviceID:    dev.DeviceID,
			Name:        dev.Name,
			IdentityKey: dev.IdentityKey,
			SigningKey:  dev.SigningKey,
			Fingerprint: dev.Fingerprint(),
			Trust:       dev.Trust,
		}
	}
	return &resp, nil
}

func (h *HiClient) TrackUserDevices(ctx context.Context, userID id.UserID) error {
	_, err := h.Crypto.FetchKeys(ctx, []id.UserID{userID}, true)
	return err
}
