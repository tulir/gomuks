// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"go.mau.fi/util/exstrings"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (h *HiClient) fetchFromKeyBackup(ctx context.Context, roomID id.RoomID, sessionID id.SessionID) (*crypto.InboundGroupSession, error) {
	data, err := h.Client.GetKeyBackupForRoomAndSession(ctx, h.KeyBackupVersion, roomID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch key from server: %w", err)
	} else if data == nil {
		return nil, nil
	}
	decrypted, err := data.SessionData.Decrypt(h.KeyBackupKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}
	sess, err := h.Crypto.ImportRoomKeyFromBackup(ctx, h.KeyBackupVersion, roomID, sessionID, decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to import decrypted key: %w", err)
	}
	return sess, nil
}

func (h *HiClient) handleReceivedMegolmSession(ctx context.Context, roomID id.RoomID, sessionID id.SessionID, firstKnownIndex uint32) {
	log := zerolog.Ctx(ctx)
	err := h.DB.SessionRequest.Remove(ctx, sessionID, firstKnownIndex)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to remove session request after receiving megolm session")
	}
	// When receiving megolm sessions in sync, wake up the request queue to ensure they get uploaded to key backup
	syncCtx, ok := ctx.Value(syncContextKey).(*syncContext)
	if ok {
		syncCtx.shouldWakeupRequestQueue = true
	}
	events, err := h.DB.Event.GetFailedByMegolmSessionID(ctx, roomID, sessionID)
	if err != nil {
		log.Err(err).Msg("Failed to get events that failed to decrypt to retry decryption")
		return
	} else if len(events) == 0 {
		log.Trace().Msg("No events to retry decryption for")
		return
	}
	decrypted := events[:0]
	for _, evt := range events {
		if evt.Decrypted != nil {
			continue
		}
		result := gjson.GetBytes(evt.Content, "ciphertext")
		idx, err := crypto.ParseMegolmMessageIndex(exstrings.UnsafeBytes(result.Str))
		if err != nil {
			log.Warn().Err(err).Stringer("event_id", evt.ID).Msg("Failed to parse megolm message index")
		} else if uint32(idx) < firstKnownIndex {
			log.Debug().Stringer("event_id", evt.ID).Msg("Skipping event with megolm message index lower than first known index")
			continue
		}

		var mautrixEvt *event.Event
		mautrixEvt, err = h.decryptEventInto(ctx, evt.AsRawMautrix(), evt)
		if err != nil {
			log.Warn().Err(err).Stringer("event_id", evt.ID).Msg("Failed to decrypt event even after receiving megolm session")
		} else {
			decrypted = append(decrypted, evt)
			h.postDecryptProcess(ctx, nil, evt, mautrixEvt)
		}
	}
	if len(decrypted) > 0 {
		var newPreview database.EventRowID
		err = h.DB.DoTxn(ctx, nil, func(ctx context.Context) error {
			for _, evt := range decrypted {
				err = h.DB.Event.UpdateDecrypted(ctx, evt)
				if err != nil {
					return fmt.Errorf("failed to save decrypted content for %s: %w", evt.ID, err)
				}
				if evt.CanUseForPreview() {
					var previewChanged bool
					previewChanged, err = h.DB.Room.UpdatePreviewIfLaterOnTimeline(ctx, evt.RoomID, evt.RowID)
					if err != nil {
						return fmt.Errorf("failed to update room %s preview to %d: %w", evt.RoomID, evt.RowID, err)
					} else if previewChanged {
						newPreview = evt.RowID
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Err(err).Msg("Failed to save decrypted events")
		} else {
			h.EventHandler(&jsoncmd.EventsDecrypted{Events: decrypted, PreviewEventRowID: newPreview, RoomID: roomID})
		}
	}
}

func (h *HiClient) WakeupRequestQueue() {
	select {
	case h.requestQueueWakeup <- struct{}{}:
	default:
	}
}

func (h *HiClient) RunRequestQueue(ctx context.Context) {
	log := zerolog.Ctx(ctx).With().Str("action", "request queue").Logger()
	ctx = log.WithContext(ctx)
	log.Info().Msg("Starting key request queue")
	defer func() {
		log.Info().Msg("Stopping key request queue")
	}()
	for {
		err := h.FetchKeysForOutdatedUsers(ctx)
		if err != nil {
			log.Err(err).Msg("Failed to fetch outdated device lists for tracked users")
		}
		h.uploadKeysToBackup(ctx)
		madeRequests, err := h.RequestQueuedSessions(ctx)
		if err != nil {
			log.Err(err).Msg("Failed to handle session request queue")
		} else if madeRequests {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-h.requestQueueWakeup:
		}
	}
}

func (h *HiClient) requestQueuedSession(ctx context.Context, req *database.SessionRequest, doneFunc func()) {
	defer doneFunc()
	log := zerolog.Ctx(ctx)
	if !req.BackupChecked {
		sess, err := h.fetchFromKeyBackup(ctx, req.RoomID, req.SessionID)
		if err != nil {
			log.Err(err).
				Stringer("session_id", req.SessionID).
				Msg("Failed to fetch session from key backup")

			// TODO should this have retries instead of just storing it's checked?
			req.BackupChecked = true
			err = h.DB.SessionRequest.Put(ctx, req)
			if err != nil {
				log.Err(err).Stringer("session_id", req.SessionID).Msg("Failed to update session request after trying to check backup")
			}
		} else if sess == nil || sess.Internal.FirstKnownIndex() > req.MinIndex {
			req.BackupChecked = true
			err = h.DB.SessionRequest.Put(ctx, req)
			if err != nil {
				log.Err(err).Stringer("session_id", req.SessionID).Msg("Failed to update session request after checking backup")
			}
		} else {
			log.Debug().Stringer("session_id", req.SessionID).
				Msg("Found session with sufficiently low first known index, removing from queue")
			err = h.DB.SessionRequest.Remove(ctx, req.SessionID, sess.Internal.FirstKnownIndex())
			if err != nil {
				log.Err(err).Stringer("session_id", req.SessionID).Msg("Failed to remove session from request queue")
			}
		}
	} else {
		err := h.Crypto.SendRoomKeyRequest(ctx, req.RoomID, "", req.SessionID, "", map[id.UserID][]id.DeviceID{
			h.Account.UserID: {"*"},
			req.Sender:       {"*"},
		})
		//var err error
		if err != nil {
			log.Err(err).
				Stringer("session_id", req.SessionID).
				Msg("Failed to send key request")
		} else {
			log.Debug().Stringer("session_id", req.SessionID).Msg("Sent key request")
			req.RequestSent = true
			err = h.DB.SessionRequest.Put(ctx, req)
			if err != nil {
				log.Err(err).Stringer("session_id", req.SessionID).Msg("Failed to update session request after sending request")
			}
		}
	}
}

const MaxParallelRequests = 5

func (h *HiClient) RequestQueuedSessions(ctx context.Context) (bool, error) {
	sessions, err := h.DB.SessionRequest.Next(ctx, MaxParallelRequests)
	if err != nil {
		return false, fmt.Errorf("failed to get next events to decrypt: %w", err)
	} else if len(sessions) == 0 {
		return false, nil
	}
	var wg sync.WaitGroup
	wg.Add(len(sessions))
	for _, req := range sessions {
		go h.requestQueuedSession(ctx, req, wg.Done)
	}
	wg.Wait()

	return true, err
}

func (h *HiClient) FetchKeysForOutdatedUsers(ctx context.Context) error {
	outdatedUsers, err := h.Crypto.CryptoStore.GetOutdatedTrackedUsers(ctx)
	if err != nil {
		return err
	} else if len(outdatedUsers) == 0 {
		return nil
	}
	_, err = h.Crypto.FetchKeys(ctx, outdatedUsers, false)
	if err != nil {
		return err
	}
	// TODO backoff for users that fail to be fetched?
	return nil
}
