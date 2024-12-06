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
	"sync"

	"github.com/rs/zerolog"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

type ProfileViewDevice struct {
	DeviceID    id.DeviceID   `json:"device_id"`
	Name        string        `json:"name"`
	IdentityKey id.Curve25519 `json:"identity_key"`
	SigningKey  id.Ed25519    `json:"signing_key"`
	Fingerprint string        `json:"fingerprint"`
	Trust       id.TrustState `json:"trust_state"`
}

type ProfileViewData struct {
	GlobalProfile *mautrix.RespUserProfile `json:"global_profile"`

	DevicesTracked bool                 `json:"devices_tracked"`
	Devices        []*ProfileViewDevice `json:"devices"`
	MasterKey      string               `json:"master_key"`
	FirstMasterKey string               `json:"first_master_key"`
	UserTrusted    bool                 `json:"user_trusted"`

	MutualRooms []id.RoomID `json:"mutual_rooms"`

	Errors []string `json:"errors"`
}

const MutualRoomsBatchLimit = 5

func (h *HiClient) GetProfileView(ctx context.Context, roomID id.RoomID, userID id.UserID) (*ProfileViewData, error) {
	log := zerolog.Ctx(ctx).With().
		Stringer("room_id", roomID).
		Stringer("target_user_id", userID).
		Logger()
	var resp ProfileViewData
	resp.Devices = make([]*ProfileViewDevice, 0)
	resp.GlobalProfile = &mautrix.RespUserProfile{}
	resp.Errors = make([]string, 0)

	var wg sync.WaitGroup
	wg.Add(3)

	var errorsLock sync.Mutex
	addError := func(err error) {
		errorsLock.Lock()
		resp.Errors = append(resp.Errors, err.Error())
		errorsLock.Unlock()
	}

	go func() {
		defer wg.Done()
		profile, err := h.Client.GetProfile(ctx, userID)
		if err != nil {
			log.Err(err).Msg("Failed to get global profile")
			addError(fmt.Errorf("failed to get global profile: %w", err))
		} else {
			resp.GlobalProfile = profile
		}
	}()
	go func() {
		defer wg.Done()
		if userID == h.Account.UserID {
			return
		}
		var nextBatch string
		for i := 0; i < MutualRoomsBatchLimit; i++ {
			mutualRooms, err := h.Client.GetMutualRooms(ctx, userID, mautrix.ReqMutualRooms{From: nextBatch})
			if err != nil {
				log.Err(err).Str("from_batch_token", nextBatch).Msg("Failed to get mutual rooms")
				addError(fmt.Errorf("failed to get mutual rooms: %w", err))
				break
			} else {
				resp.MutualRooms = mutualRooms.Joined
				nextBatch = mutualRooms.NextBatch
				if nextBatch == "" {
					break
				}
			}
		}
		slices.Sort(resp.MutualRooms)
		resp.MutualRooms = slices.Compact(resp.MutualRooms)
	}()
	go func() {
		defer wg.Done()
		userIDs, err := h.CryptoStore.FilterTrackedUsers(ctx, []id.UserID{userID})
		if err != nil {
			log.Err(err).Msg("Failed to check if user's devices are tracked")
			addError(fmt.Errorf("failed to check if user's devices are tracked: %w", err))
			return
		} else if len(userIDs) == 0 {
			return
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
			addError(fmt.Errorf("failed to get cross-signing keys: %w", err))
			return
		} else if csKeys != nil && theirMasterKey.Key != "" {
			resp.MasterKey = theirMasterKey.Key.Fingerprint()
			resp.FirstMasterKey = theirMasterKey.First.Fingerprint()
			selfKeySigned, err := h.CryptoStore.IsKeySignedBy(ctx, userID, theirSelfSignKey.Key, userID, theirMasterKey.Key)
			if err != nil {
				log.Err(err).Msg("Failed to check if self-signing key is signed by master key")
				addError(fmt.Errorf("failed to check if self-signing key is signed by master key: %w", err))
			} else if !selfKeySigned {
				theirSelfSignKey = id.CrossSigningKey{}
				addError(fmt.Errorf("self-signing key is not signed by master key"))
			}
		} else {
			addError(fmt.Errorf("cross-signing keys not found"))
		}
		devices, err := h.CryptoStore.GetDevices(ctx, userID)
		if err != nil {
			log.Err(err).Msg("Failed to get devices for user")
			addError(fmt.Errorf("failed to get devices: %w", err))
			return
		}
		if userID == h.Account.UserID {
			resp.UserTrusted, err = h.CryptoStore.IsKeySignedBy(ctx, userID, theirMasterKey.Key, userID, h.Crypto.OwnIdentity().SigningKey)
		} else if ownUserSigningKey != "" && theirMasterKey.Key != "" {
			resp.UserTrusted, err = h.CryptoStore.IsKeySignedBy(ctx, userID, theirMasterKey.Key, h.Account.UserID, ownUserSigningKey)
		}
		if err != nil {
			log.Err(err).Msg("Failed to check if user is trusted")
			addError(fmt.Errorf("failed to check if user is trusted: %w", err))
		}
		resp.Devices = make([]*ProfileViewDevice, len(devices))
		i := 0
		for _, device := range devices {
			signatures, err := h.CryptoStore.GetSignaturesForKeyBy(ctx, device.UserID, device.SigningKey, device.UserID)
			if err != nil {
				log.Err(err).Stringer("device_id", device.DeviceID).Msg("Failed to get signatures for device")
				addError(fmt.Errorf("failed to get signatures for device %s: %w", device.DeviceID, err))
			} else if _, signed := signatures[theirSelfSignKey.Key]; signed && device.Trust == id.TrustStateUnset && theirSelfSignKey.Key != "" {
				if resp.UserTrusted {
					device.Trust = id.TrustStateCrossSignedVerified
				} else if theirMasterKey.Key == theirMasterKey.First {
					device.Trust = id.TrustStateCrossSignedTOFU
				} else {
					device.Trust = id.TrustStateCrossSignedUntrusted
				}
			}
			resp.Devices[i] = &ProfileViewDevice{
				DeviceID:    device.DeviceID,
				Name:        device.Name,
				IdentityKey: device.IdentityKey,
				SigningKey:  device.SigningKey,
				Fingerprint: device.Fingerprint(),
				Trust:       device.Trust,
			}
			i++
		}
		slices.SortFunc(resp.Devices, func(a, b *ProfileViewDevice) int {
			return strings.Compare(a.DeviceID.String(), b.DeviceID.String())
		})
	}()

	wg.Wait()
	return &resp, nil
}
