// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/id"
)

func (h *HiClient) uploadKeysToBackup(ctx context.Context) {
	log := zerolog.Ctx(ctx)
	version := h.KeyBackupVersion
	key := h.KeyBackupKey
	if version == "" || key == nil {
		return
	}

	sessions, err := h.CryptoStore.GetGroupSessionsWithoutKeyBackupVersion(ctx, version).AsList()
	if err != nil {
		log.Err(err).Msg("Failed to get megolm sessions that aren't backed up")
		return
	} else if len(sessions) == 0 {
		return
	}
	log.Debug().Int("session_count", len(sessions)).Msg("Backing up megolm sessions")
	for chunk := range slices.Chunk(sessions, 100) {
		err = h.uploadKeyBackupBatch(ctx, version, key, chunk)
		if err != nil {
			log.Err(err).Msg("Failed to upload key backup batch")
			return
		}
		err = h.CryptoStore.DB.DoTxn(ctx, nil, func(ctx context.Context) error {
			for _, sess := range chunk {
				sess.KeyBackupVersion = version
				err := h.CryptoStore.PutGroupSession(ctx, sess)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Err(err).Msg("Failed to update key backup version of uploaded megolm sessions in database")
			return
		}
	}
	log.Info().Int("session_count", len(sessions)).Msg("Successfully uploaded megolm sessions to key backup")
}

func (h *HiClient) uploadKeyBackupBatch(ctx context.Context, version id.KeyBackupVersion, megolmBackupKey *backup.MegolmBackupKey, sessions []*crypto.InboundGroupSession) error {
	if len(sessions) == 0 {
		return nil
	}

	req := mautrix.ReqKeyBackup{
		Rooms: map[id.RoomID]mautrix.ReqRoomKeyBackup{},
	}

	for _, session := range sessions {
		sessionKey, err := session.Internal.Export(session.Internal.FirstKnownIndex())
		if err != nil {
			return fmt.Errorf("failed to export session data: %w", err)
		}

		sessionData, err := backup.EncryptSessionData(megolmBackupKey, &backup.MegolmSessionData{
			Algorithm:          id.AlgorithmMegolmV1,
			ForwardingKeyChain: session.ForwardingChains,
			SenderClaimedKeys: backup.SenderClaimedKeys{
				Ed25519: session.SigningKey,
			},
			SenderKey:  session.SenderKey,
			SessionKey: string(sessionKey),
		})
		if err != nil {
			return fmt.Errorf("failed to encrypt session data: %w", err)
		}

		jsonSessionData, err := json.Marshal(sessionData)
		if err != nil {
			return fmt.Errorf("failed to marshal session data: %w", err)
		}

		roomData, ok := req.Rooms[session.RoomID]
		if !ok {
			roomData = mautrix.ReqRoomKeyBackup{
				Sessions: map[id.SessionID]mautrix.ReqKeyBackupData{},
			}
			req.Rooms[session.RoomID] = roomData
		}

		roomData.Sessions[session.ID()] = mautrix.ReqKeyBackupData{
			FirstMessageIndex: int(session.Internal.FirstKnownIndex()),
			ForwardedCount:    len(session.ForwardingChains),
			IsVerified:        session.Internal.IsVerified(),
			SessionData:       jsonSessionData,
		}
	}

	_, err := h.Client.PutKeysInBackup(ctx, version, &req)
	return err
}

type KeyBackupRestoreProgress struct {
	CurrentRoomID id.RoomID `json:"current_room_id"`
	Stage         string    `json:"stage"`

	Decrypted        int `json:"decrypted"`
	DecryptionFailed int `json:"decryption_failed"`
	ImportFailed     int `json:"import_failed"`
	Saved            int `json:"saved"`
	PostProcessed    int `json:"post_processed"`

	Total int `json:"total"`
}

type keyBackupEntry struct {
	RoomID    id.RoomID
	SessionID id.SessionID
	Entry     *crypto.InboundGroupSession
}

func (h *HiClient) RestoreKeyBackup(
	ctx context.Context,
	onlyRoomID id.RoomID,
	progressCallback func(progress KeyBackupRestoreProgress),
) error {
	var progress KeyBackupRestoreProgress
	if onlyRoomID != "" {
		progress.CurrentRoomID = onlyRoomID
	}
	progress.Stage = "fetching"
	progressCallback(progress)
	var rooms map[id.RoomID]mautrix.RespRoomKeyBackup[backup.EncryptedSessionData[backup.MegolmSessionData]]
	if onlyRoomID != "" {
		resp, err := h.Client.GetKeyBackupForRoom(ctx, h.KeyBackupVersion, onlyRoomID)
		if err != nil {
			return err
		}
		rooms = map[id.RoomID]mautrix.RespRoomKeyBackup[backup.EncryptedSessionData[backup.MegolmSessionData]]{
			onlyRoomID: *resp,
		}
	} else {
		resp, err := h.Client.GetKeyBackup(ctx, h.KeyBackupVersion)
		if err != nil {
			return err
		}
		rooms = resp.Rooms
	}
	for _, keys := range rooms {
		progress.Total += len(keys.Sessions)
	}
	progress.Stage = "decrypting"
	progressCallback(progress)
	const callbackInterval = 100 * time.Millisecond
	const persistChunkSize = 100
	lastCallback := time.Now()
	debouncedProgressCallback := func() {
		if time.Since(lastCallback) > callbackInterval {
			lastCallback = time.Now()
			progressCallback(progress)
		}
	}
	log := zerolog.Ctx(ctx)
	entries := make([]keyBackupEntry, 0, progress.Total)
	for roomID, keys := range rooms {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		progress.CurrentRoomID = roomID
		encryptionEvent, err := h.Crypto.StateStore.GetEncryptionEvent(ctx, roomID)
		if err != nil {
			log.Err(err).
				Stringer("room_id", roomID).
				Msg("Failed to get encryption event for room")
			return fmt.Errorf("failed to get encryption event for room %s: %w", roomID, err)
		}
		for sessionID, key := range keys.Sessions {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			decrypted, err := key.SessionData.Decrypt(h.KeyBackupKey)
			if err != nil {
				log.Err(err).
					Stringer("key_backup_version", h.KeyBackupVersion).
					Stringer("room_id", roomID).
					Stringer("session_id", sessionID).
					Msg("Failed to decrypt session data")
				progress.DecryptionFailed++
			} else if imported, err := h.Crypto.ImportRoomKeyFromBackupWithoutSaving(ctx, h.KeyBackupVersion, roomID, encryptionEvent, sessionID, decrypted); err != nil {
				log.Err(err).
					Stringer("key_backup_version", h.KeyBackupVersion).
					Stringer("room_id", roomID).
					Stringer("session_id", sessionID).
					Msg("Failed to import session data")
				progress.ImportFailed++
			} else {
				progress.Decrypted++
				entries = append(entries, keyBackupEntry{
					RoomID:    roomID,
					SessionID: sessionID,
					Entry:     imported,
				})
			}
			debouncedProgressCallback()
		}
	}
	log.Debug().Any("progress", progress).Msg("Finished decrypting key backup, storing entries")
	progress.Stage = "saving"
	progressCallback(progress)
	for chunk := range slices.Chunk(entries, persistChunkSize) {
		err := h.DB.DoTxn(ctx, nil, func(ctx context.Context) error {
			for _, entry := range chunk {
				progress.CurrentRoomID = entry.RoomID
				err := h.CryptoStore.PutGroupSession(ctx, entry.Entry)
				if err != nil {
					log.Err(err).
						Stringer("key_backup_version", h.KeyBackupVersion).
						Stringer("room_id", entry.RoomID).
						Stringer("session_id", entry.SessionID).
						Msg("Failed to save session data")
					return err
				} else {
					progress.Saved++
				}
				debouncedProgressCallback()
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to persist decrypted entries: %w", err)
		}
	}
	log.Debug().Any("progress", progress).Msg("Finished saving key backup, retrying decryption")
	// Don't allow cancelling this step, decrypting should be retried even if the client disappears
	noCancelCtx := context.WithoutCancel(ctx)
	progress.Stage = "postprocessing"
	progressCallback(progress)
	for _, entry := range entries {
		progress.CurrentRoomID = entry.RoomID
		h.Crypto.MarkSessionReceived(noCancelCtx, entry.RoomID, entry.SessionID, entry.Entry.Internal.FirstKnownIndex())
		progress.PostProcessed++
		if ctx.Err() == nil {
			debouncedProgressCallback()
		}
	}
	progress.Stage = "done"
	progressCallback(progress)
	log.Debug().Any("progress", progress).Msg("Finished importing key backup")
	return nil
}
