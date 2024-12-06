// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
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

type ProfileDevice struct {
	DeviceID    id.DeviceID   `json:"device_id"`
	Name        string        `json:"name"`
	IdentityKey id.Curve25519 `json:"identity_key"`
	SigningKey  id.Ed25519    `json:"signing_key"`
	Fingerprint string        `json:"fingerprint"`
	Trust       id.TrustState `json:"trust_state"`
}

type ProfileEncryptionInfo struct {
	DevicesTracked bool             `json:"devices_tracked"`
	Devices        []*ProfileDevice `json:"devices"`
	MasterKey      string           `json:"master_key"`
	FirstMasterKey string           `json:"first_master_key"`
	UserTrusted    bool             `json:"user_trusted"`
	Errors         []string         `json:"errors"`
}

func (h *HiClient) GetProfileEncryptionInfo(ctx context.Context, userID id.UserID) (*ProfileEncryptionInfo, error) {
	var resp ProfileEncryptionInfo
	log := zerolog.Ctx(ctx)
	userIDs, err := h.CryptoStore.FilterTrackedUsers(ctx, []id.UserID{userID})
	if err != nil {
		log.Err(err).Msg("Failed to check if user's devices are tracked")
		return nil, fmt.Errorf("failed to check if user's devices are tracked: %w", err)
	} else if len(userIDs) == 0 {
		return &resp, nil
	}
	ownKeys := h.Crypto.GetOwnCrossSigningPublicKeys(ctx)
	var ownUserSigningKey id.Ed25519
	if ownKeys != nil {
		ownUserSigningKey = ownKeys.UserSigningKey
	}
	resp.DevicesTracked = true
	csKeys, err := h.CryptoStore.GetCrossSigningKeys(ctx, userID)
	theirMasterKey := csKeys[id.XSUsageMaster]
	theirSelfSignKey := csKeys[id.XSUsageSelfSigning]
	if err != nil {
		log.Err(err).Msg("Failed to get cross-signing keys")
		return nil, fmt.Errorf("failed to get cross-signing keys: %w", err)
	} else if csKeys != nil && theirMasterKey.Key != "" {
		resp.MasterKey = theirMasterKey.Key.Fingerprint()
		resp.FirstMasterKey = theirMasterKey.First.Fingerprint()
		selfKeySigned, err := h.CryptoStore.IsKeySignedBy(ctx, userID, theirSelfSignKey.Key, userID, theirMasterKey.Key)
		if err != nil {
			log.Err(err).Msg("Failed to check if self-signing key is signed by master key")
			return nil, fmt.Errorf("failed to check if self-signing key is signed by master key: %w", err)
		} else if !selfKeySigned {
			theirSelfSignKey = id.CrossSigningKey{}
			resp.Errors = append(resp.Errors, "Self-signing key is not signed by master key")
		}
	} else {
		resp.Errors = append(resp.Errors, "Cross-signing keys not found")
	}
	devices, err := h.CryptoStore.GetDevices(ctx, userID)
	if err != nil {
		log.Err(err).Msg("Failed to get devices for user")
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	if userID == h.Account.UserID {
		resp.UserTrusted, err = h.CryptoStore.IsKeySignedBy(ctx, userID, theirMasterKey.Key, userID, h.Crypto.OwnIdentity().SigningKey)
	} else if ownUserSigningKey != "" && theirMasterKey.Key != "" {
		resp.UserTrusted, err = h.CryptoStore.IsKeySignedBy(ctx, userID, theirMasterKey.Key, h.Account.UserID, ownUserSigningKey)
	}
	if err != nil {
		log.Err(err).Msg("Failed to check if user is trusted")
		resp.Errors = append(resp.Errors, fmt.Sprintf("Failed to check if user is trusted: %v", err))
	}
	resp.Devices = make([]*ProfileDevice, len(devices))
	i := 0
	for _, device := range devices {
		signatures, err := h.CryptoStore.GetSignaturesForKeyBy(ctx, device.UserID, device.SigningKey, device.UserID)
		if err != nil {
			log.Err(err).Stringer("device_id", device.DeviceID).Msg("Failed to get signatures for device")
			resp.Errors = append(resp.Errors, fmt.Sprintf("Failed to get signatures for device %s: %v", device.DeviceID, err))
		} else if _, signed := signatures[theirSelfSignKey.Key]; signed && device.Trust == id.TrustStateUnset && theirSelfSignKey.Key != "" {
			if resp.UserTrusted {
				device.Trust = id.TrustStateCrossSignedVerified
			} else if theirMasterKey.Key == theirMasterKey.First {
				device.Trust = id.TrustStateCrossSignedTOFU
			} else {
				device.Trust = id.TrustStateCrossSignedUntrusted
			}
		}
		resp.Devices[i] = &ProfileDevice{
			DeviceID:    device.DeviceID,
			Name:        device.Name,
			IdentityKey: device.IdentityKey,
			SigningKey:  device.SigningKey,
			Fingerprint: device.Fingerprint(),
			Trust:       device.Trust,
		}
		i++
	}
	slices.SortFunc(resp.Devices, func(a, b *ProfileDevice) int {
		return strings.Compare(a.DeviceID.String(), b.DeviceID.String())
	})
	return &resp, nil
}
