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

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/id"
)

func (c *HiClient) uploadKeysToBackup(ctx context.Context) {
	log := zerolog.Ctx(ctx)
	version := c.KeyBackupVersion
	key := c.KeyBackupKey
	if version == "" || key == nil {
		return
	}

	sessions, err := c.CryptoStore.GetGroupSessionsWithoutKeyBackupVersion(ctx, version).AsList()
	if err != nil {
		log.Err(err).Msg("Failed to get megolm sessions that aren't backed up")
		return
	} else if len(sessions) == 0 {
		return
	}
	log.Debug().Int("session_count", len(sessions)).Msg("Backing up megolm sessions")
	for chunk := range slices.Chunk(sessions, 100) {
		err = c.uploadKeyBackupBatch(ctx, version, key, chunk)
		if err != nil {
			log.Err(err).Msg("Failed to upload key backup batch")
			return
		}
		err = c.CryptoStore.DB.DoTxn(ctx, nil, func(ctx context.Context) error {
			for _, sess := range chunk {
				sess.KeyBackupVersion = version
				err := c.CryptoStore.PutGroupSession(ctx, sess)
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

func (c *HiClient) uploadKeyBackupBatch(ctx context.Context, version id.KeyBackupVersion, megolmBackupKey *backup.MegolmBackupKey, sessions []*crypto.InboundGroupSession) error {
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

	_, err := c.Client.PutKeysInBackup(ctx, version, &req)
	return err
}
