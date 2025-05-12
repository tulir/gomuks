// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.mau.fi/util/emojirunes"
	"go.mau.fi/util/exzerolog"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/olm"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type syncContext struct {
	shouldWakeupRequestQueue bool

	evt *jsoncmd.SyncComplete
}

func (h *HiClient) markSyncErrored(err error, permanent bool) {
	stat := &jsoncmd.SyncStatus{
		Type:       jsoncmd.SyncStatusErroring,
		Error:      err.Error(),
		ErrorCount: h.syncErrors,
		LastSync:   jsontime.UM(h.lastSync),
	}
	if permanent {
		stat.Type = jsoncmd.SyncStatusFailed
	}
	h.SyncStatus.Store(stat)
	h.EventHandler(stat)
}

var (
	syncOK      = &jsoncmd.SyncStatus{Type: jsoncmd.SyncStatusOK}
	syncWaiting = &jsoncmd.SyncStatus{Type: jsoncmd.SyncStatusWaiting}
)

func (h *HiClient) markSyncOK() {
	if h.SyncStatus.Swap(syncOK) != syncOK {
		h.EventHandler(syncOK)
	}
}

func (h *HiClient) preProcessSyncResponse(ctx context.Context, resp *mautrix.RespSync, since string) error {
	log := zerolog.Ctx(ctx)
	listenToDevice := h.ToDeviceInSync.Load()
	var syncTD []*jsoncmd.SyncToDevice

	postponedToDevices := resp.ToDevice.Events[:0]
	for _, evt := range resp.ToDevice.Events {
		evt.Type.Class = event.ToDeviceEventType
		err := evt.Content.ParseRaw(evt.Type)
		if err != nil && !errors.Is(err, event.ErrContentAlreadyParsed) {
			log.Warn().Err(err).
				Stringer("event_type", &evt.Type).
				Stringer("sender", evt.Sender).
				Msg("Failed to parse to-device event, skipping")
			continue
		}

		switch content := evt.Content.Parsed.(type) {
		case *event.EncryptedEventContent:
			unhandledDecrypted := h.Crypto.HandleEncryptedEvent(ctx, evt)
			if unhandledDecrypted != nil && listenToDevice {
				syncTD = append(syncTD, &jsoncmd.SyncToDevice{
					Sender:    evt.Sender,
					Type:      unhandledDecrypted.Type,
					Content:   unhandledDecrypted.Content.VeryRaw,
					Encrypted: true,
				})
			}
		case *event.RoomKeyWithheldEventContent:
			// TODO move this check to mautrix-go?
			if evt.Sender == h.Account.UserID && content.Code == event.RoomKeyWithheldUnavailable {
				log.Debug().Any("withheld_content", content).Msg("Ignoring m.unavailable megolm session withheld event")
			} else {
				h.Crypto.HandleRoomKeyWithheld(ctx, content)
			}
		case *event.SecretRequestEventContent, *event.RoomKeyRequestEventContent:
			postponedToDevices = append(postponedToDevices, evt)
		default:
			if listenToDevice {
				syncTD = append(syncTD, &jsoncmd.SyncToDevice{
					Sender:  evt.Sender,
					Type:    evt.Type,
					Content: evt.Content.VeryRaw,
				})
			}
		}
	}
	resp.ToDevice.Events = postponedToDevices
	if len(syncTD) > 0 {
		ctx.Value(syncContextKey).(*syncContext).evt.ToDevice = syncTD
	}
	h.Crypto.MarkOlmHashSavePoint(ctx)

	return nil
}

func (h *HiClient) maybeDiscardOutboundSession(ctx context.Context, newMembership event.Membership, evt *event.Event) bool {
	var prevMembership event.Membership = "unknown"
	if evt.Unsigned.PrevContent != nil {
		prevMembership = event.Membership(gjson.GetBytes(evt.Unsigned.PrevContent.VeryRaw, "membership").Str)
	}
	if prevMembership == "unknown" || prevMembership == "" {
		cs, err := h.DB.CurrentState.Get(ctx, evt.RoomID, event.StateMember, evt.GetStateKey())
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).
				Stringer("room_id", evt.RoomID).
				Str("user_id", evt.GetStateKey()).
				Msg("Failed to get previous membership")
			return false
		} else if cs != nil {
			prevMembership = event.Membership(gjson.GetBytes(cs.Content, "membership").Str)
		}
	}
	if prevMembership == newMembership ||
		(prevMembership == event.MembershipInvite && newMembership == event.MembershipJoin && h.shouldShareKeysToInvitedUsers(ctx, evt.RoomID)) ||
		(prevMembership == event.MembershipJoin && newMembership == event.MembershipInvite) ||
		(prevMembership == event.MembershipBan && newMembership == event.MembershipLeave) ||
		(prevMembership == event.MembershipLeave && newMembership == event.MembershipBan) {
		return false
	}
	zerolog.Ctx(ctx).Debug().
		Stringer("room_id", evt.RoomID).
		Str("user_id", evt.GetStateKey()).
		Str("prev_membership", string(prevMembership)).
		Str("new_membership", string(newMembership)).
		Msg("Got membership state change, invalidating group session in room")
	err := h.CryptoStore.RemoveOutboundGroupSession(ctx, evt.RoomID)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Stringer("room_id", evt.RoomID).Msg("Failed to invalidate outbound group session")
		return false
	}
	return true
}

func (h *HiClient) postProcessSyncResponse(ctx context.Context, resp *mautrix.RespSync, since string) {
	h.Crypto.HandleOTKCounts(ctx, &resp.DeviceOTKCount)
	go h.asyncPostProcessSyncResponse(ctx, resp, since)
	syncCtx := ctx.Value(syncContextKey).(*syncContext)
	if syncCtx.shouldWakeupRequestQueue {
		h.WakeupRequestQueue()
	}
	if !h.firstSyncReceived {
		h.firstSyncReceived = true
		h.Client.Client.Transport.(*http.Transport).ResponseHeaderTimeout = 60 * time.Second
		h.Client.Client.Timeout = 180 * time.Second
	}
	if !syncCtx.evt.IsEmpty() {
		h.EventHandler(syncCtx.evt)
	}
}

func (h *HiClient) asyncPostProcessSyncResponse(ctx context.Context, resp *mautrix.RespSync, since string) {
	for _, evt := range resp.ToDevice.Events {
		switch content := evt.Content.Parsed.(type) {
		case *event.SecretRequestEventContent:
			h.Crypto.HandleSecretRequest(ctx, evt.Sender, content)
		case *event.RoomKeyRequestEventContent:
			h.Crypto.HandleRoomKeyRequest(ctx, evt.Sender, content)
		}
	}
}

func (h *HiClient) processSyncResponse(ctx context.Context, resp *mautrix.RespSync, since string) error {
	if len(resp.DeviceLists.Changed) > 0 {
		zerolog.Ctx(ctx).Debug().
			Array("users", exzerolog.ArrayOfStringers(resp.DeviceLists.Changed)).
			Msg("Marking changed device lists for tracked users as outdated")
		err := h.Crypto.CryptoStore.MarkTrackedUsersOutdated(ctx, resp.DeviceLists.Changed)
		if err != nil {
			return fmt.Errorf("failed to mark changed device lists as outdated: %w", err)
		}
		ctx.Value(syncContextKey).(*syncContext).shouldWakeupRequestQueue = true
	}

	accountData := make(map[event.Type]*database.AccountData, len(resp.AccountData.Events))
	var err error
	for _, evt := range resp.AccountData.Events {
		evt.Type.Class = event.AccountDataEventType
		accountData[evt.Type], err = h.DB.AccountData.Put(ctx, h.Account.UserID, evt.Type, evt.Content.VeryRaw)
		if err != nil {
			return fmt.Errorf("failed to save account data event %s: %w", evt.Type.Type, err)
		}
		if evt.Type == event.AccountDataPushRules {
			err = evt.Content.ParseRaw(evt.Type)
			if err != nil && !errors.Is(err, event.ErrContentAlreadyParsed) {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to parse push rules in sync")
			} else if pushRules, ok := evt.Content.Parsed.(*pushrules.EventContent); ok {
				h.receiveNewPushRules(ctx, pushRules.Ruleset)
				zerolog.Ctx(ctx).Debug().Msg("Updated push rules from sync")
			}
		}
	}
	ctx.Value(syncContextKey).(*syncContext).evt.AccountData = accountData
	for roomID, room := range resp.Rooms.Invite {
		err = h.processSyncInvitedRoom(ctx, roomID, room)
		if err != nil {
			return fmt.Errorf("failed to process invited room %s: %w", roomID, err)
		}
	}
	for roomID, room := range resp.Rooms.Join {
		err = h.processSyncJoinedRoom(ctx, roomID, room)
		if err != nil {
			return fmt.Errorf("failed to process joined room %s: %w", roomID, err)
		}
	}
	for roomID, room := range resp.Rooms.Leave {
		err = h.processSyncLeftRoom(ctx, roomID, room)
		if err != nil {
			return fmt.Errorf("failed to process left room %s: %w", roomID, err)
		}
	}
	h.Account.NextBatch = resp.NextBatch
	err = h.DB.Account.PutNextBatch(ctx, h.Account.UserID, resp.NextBatch)
	if err != nil {
		return fmt.Errorf("failed to save next_batch: %w", err)
	}
	return nil
}

func (h *HiClient) receiptsToList(content *event.ReceiptEventContent) ([]*database.Receipt, []id.EventID) {
	receiptList := make([]*database.Receipt, 0)
	var newOwnReceipts []id.EventID
	for eventID, receipts := range *content {
		for receiptType, users := range receipts {
			for userID, receiptInfo := range users {
				if userID == h.Account.UserID {
					newOwnReceipts = append(newOwnReceipts, eventID)
				}
				if receiptInfo.ThreadID == event.ReadReceiptThreadMain {
					receiptInfo.ThreadID = ""
				}
				receiptList = append(receiptList, &database.Receipt{
					UserID:      userID,
					ReceiptType: receiptType,
					ThreadID:    receiptInfo.ThreadID,
					EventID:     eventID,
					Timestamp:   jsontime.UM(receiptInfo.Timestamp),
				})
			}
		}
	}
	return receiptList, newOwnReceipts
}

func (h *HiClient) processSyncInvitedRoom(ctx context.Context, roomID id.RoomID, room *mautrix.SyncInvitedRoom) error {
	ir := &database.InvitedRoom{
		ID:          roomID,
		CreatedAt:   jsontime.UnixMilliNow(),
		InviteState: room.State.Events,
	}
	if len(ir.InviteState) == 0 {
		zerolog.Ctx(ctx).Warn().Stringer("room_id", roomID).Msg("Got invited room with no state, ignoring")
		return nil
	}
	for _, evt := range room.State.Events {
		if evt.Type == event.StateMember && evt.GetStateKey() == h.Account.UserID.String() && evt.Timestamp != 0 {
			ir.CreatedAt = jsontime.UM(time.UnixMilli(evt.Timestamp))
			break
		}
	}
	err := h.DB.InvitedRoom.Upsert(ctx, ir)
	if err != nil {
		return fmt.Errorf("failed to save invited room: %w", err)
	}
	syncEvt := ctx.Value(syncContextKey).(*syncContext).evt
	syncEvt.InvitedRooms = append(syncEvt.InvitedRooms, ir)
	return nil
}

func (h *HiClient) processSyncJoinedRoom(ctx context.Context, roomID id.RoomID, room *mautrix.SyncJoinedRoom) error {
	existingRoomData, err := h.DB.Room.Get(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to get room data: %w", err)
	} else if existingRoomData == nil {
		err = h.DB.Room.CreateRow(ctx, roomID)
		if err != nil {
			return fmt.Errorf("failed to ensure room row exists: %w", err)
		}
		existingRoomData = &database.Room{
			ID: roomID,
			// Hack to set a default value for SortingTimestamp which is before all existing rooms,
			// but not the same for all rooms without a timestamp.
			SortingTimestamp: jsontime.UM(time.UnixMilli(time.Now().Unix())),
		}
	}

	accountData := make(map[event.Type]*database.AccountData, len(room.AccountData.Events))
	for _, evt := range room.AccountData.Events {
		evt.Type.Class = event.AccountDataEventType
		evt.RoomID = roomID
		accountData[evt.Type], err = h.DB.AccountData.PutRoom(ctx, h.Account.UserID, roomID, evt.Type, evt.Content.VeryRaw)
		if err != nil {
			return fmt.Errorf("failed to save account data event %s: %w", evt.Type.Type, err)
		}
	}
	var receiptsList []*database.Receipt
	var newOwnReceipts []id.EventID
	for _, evt := range room.Ephemeral.Events {
		evt.Type.Class = event.EphemeralEventType
		err = evt.Content.ParseRaw(evt.Type)
		if err != nil && !errors.Is(err, event.ErrContentAlreadyParsed) {
			zerolog.Ctx(ctx).Debug().Err(err).Msg("Failed to parse ephemeral event content")
			continue
		}
		switch evt.Type {
		case event.EphemeralEventReceipt:
			list, ownList := h.receiptsToList(evt.Content.AsReceipt())
			receiptsList = append(receiptsList, list...)
			newOwnReceipts = append(newOwnReceipts, ownList...)
		case event.EphemeralEventTyping:
			go h.EventHandler(&jsoncmd.Typing{
				RoomID:             roomID,
				TypingEventContent: *evt.Content.AsTyping(),
			})
		}
	}
	err = h.processStateAndTimeline(
		ctx,
		existingRoomData,
		&room.State,
		&room.Timeline,
		&room.Summary,
		receiptsList,
		newOwnReceipts,
		accountData,
	)
	if err != nil {
		return err
	}
	return nil
}

func (h *HiClient) processSyncLeftRoom(ctx context.Context, roomID id.RoomID, room *mautrix.SyncLeftRoom) error {
	zerolog.Ctx(ctx).Debug().Stringer("room_id", roomID).Msg("Deleting left room")
	err := h.DB.Room.Delete(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}
	err = h.DB.InvitedRoom.Delete(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete invited room: %w", err)
	}
	err = h.CryptoStore.RemoveOutboundGroupSession(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to remove outbound group session: %w", err)
	}
	payload := ctx.Value(syncContextKey).(*syncContext).evt
	payload.LeftRooms = append(payload.LeftRooms, roomID)
	return nil
}

func isDecryptionErrorRetryable(err error) bool {
	return errors.Is(err, crypto.NoSessionFound) || errors.Is(err, olm.UnknownMessageIndex) || errors.Is(err, crypto.ErrGroupSessionWithheld)
}

func removeReplyFallback(evt *event.Event) []byte {
	if evt.Type != event.EventMessage && evt.Type != event.EventSticker {
		return nil
	}
	_ = evt.Content.ParseRaw(evt.Type)
	content, ok := evt.Content.Parsed.(*event.MessageEventContent)
	if ok && content.RelatesTo.GetReplyTo() != "" {
		prevFormattedBody := content.FormattedBody
		content.RemoveReplyFallback()
		if content.FormattedBody != prevFormattedBody {
			bytes, err := sjson.SetBytes(evt.Content.VeryRaw, "formatted_body", content.FormattedBody)
			bytes, err2 := sjson.SetBytes(bytes, "body", content.Body)
			if err == nil && err2 == nil {
				return bytes
			}
		}
	}
	return nil
}

func (h *HiClient) decryptEvent(ctx context.Context, evt *event.Event) (*event.Event, []byte, bool, string, error) {
	err := evt.Content.ParseRaw(evt.Type)
	if err != nil && !errors.Is(err, event.ErrContentAlreadyParsed) {
		return nil, nil, false, "", err
	}
	decrypted, err := h.Crypto.DecryptMegolmEvent(ctx, evt)
	if err != nil {
		return nil, nil, false, "", err
	}
	withoutFallback := removeReplyFallback(decrypted)
	if withoutFallback != nil {
		return decrypted, withoutFallback, true, decrypted.Type.Type, nil
	}
	return decrypted, decrypted.Content.VeryRaw, false, decrypted.Type.Type, nil
}

func (h *HiClient) decryptEventInto(ctx context.Context, evt *event.Event, dbEvt *database.Event) (*event.Event, error) {
	decryptedEvt, rawContent, fallbackRemoved, decryptedType, err := h.decryptEvent(ctx, evt)
	if err != nil {
		dbEvt.DecryptionError = err.Error()
		return nil, err
	}
	dbEvt.Decrypted = rawContent
	if fallbackRemoved {
		dbEvt.MarkReplyFallbackRemoved()
	}
	dbEvt.DecryptedType = decryptedType
	return decryptedEvt, nil
}

func (h *HiClient) addMediaCache(
	ctx context.Context,
	eventRowID database.EventRowID,
	uri id.ContentURIString,
	file *event.EncryptedFileInfo,
	info *event.FileInfo,
	fileName string,
) {
	parsedMXC := uri.ParseOrIgnore()
	if !parsedMXC.IsValid() {
		return
	}
	cm := &database.Media{
		MXC:      parsedMXC,
		FileName: fileName,
	}
	if file != nil {
		cm.EncFile = &file.EncryptedFile
	}
	if info != nil {
		cm.MimeType = info.MimeType
	}
	err := h.DB.Media.Put(ctx, cm)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).
			Stringer("mxc", parsedMXC).
			Int64("event_rowid", int64(eventRowID)).
			Msg("Failed to add database media entry")
		return
	}
	err = h.DB.Media.AddReference(ctx, eventRowID, parsedMXC)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).
			Stringer("mxc", parsedMXC).
			Int64("event_rowid", int64(eventRowID)).
			Msg("Failed to add database media reference")
	}
}

func (h *HiClient) cacheMedia(ctx context.Context, evt *event.Event, rowID database.EventRowID) {
	cacheMessageEventContent := func(content *event.MessageEventContent) {
		if content.File != nil {
			h.addMediaCache(ctx, rowID, content.File.URL, content.File, content.Info, content.GetFileName())
		} else if content.URL != "" {
			h.addMediaCache(ctx, rowID, content.URL, nil, content.Info, content.GetFileName())
		}
		if content.GetInfo().ThumbnailFile != nil {
			h.addMediaCache(ctx, rowID, content.Info.ThumbnailFile.URL, content.Info.ThumbnailFile, content.Info.ThumbnailInfo, "")
		} else if content.GetInfo().ThumbnailURL != "" {
			h.addMediaCache(ctx, rowID, content.Info.ThumbnailURL, nil, content.Info.ThumbnailInfo, "")
		}

		for _, image := range content.BeeperGalleryImages {
			h.cacheMedia(ctx, &event.Event{
				Type:    event.EventMessage,
				Content: event.Content{Parsed: image},
			}, rowID)
		}

		for _, preview := range content.BeeperLinkPreviews {
			info := &event.FileInfo{MimeType: preview.ImageType}
			if preview.ImageEncryption != nil {
				h.addMediaCache(ctx, rowID, preview.ImageEncryption.URL, preview.ImageEncryption, info, "")
			} else if preview.ImageURL != "" {
				h.addMediaCache(ctx, rowID, preview.ImageURL, nil, info, "")
			}
		}
	}

	switch evt.Type {
	case event.EventMessage, event.EventSticker:
		content, ok := evt.Content.Parsed.(*event.MessageEventContent)
		if !ok {
			return
		}

		cacheMessageEventContent(content)
		if content.NewContent != nil {
			cacheMessageEventContent(content.NewContent)
		}
	case event.StateRoomAvatar:
		_ = evt.Content.ParseRaw(evt.Type)
		content, ok := evt.Content.Parsed.(*event.RoomAvatarEventContent)
		if !ok {
			return
		}
		h.addMediaCache(ctx, rowID, content.URL, nil, nil, "")
	case event.StateMember:
		_ = evt.Content.ParseRaw(evt.Type)
		content, ok := evt.Content.Parsed.(*event.MemberEventContent)
		if !ok {
			return
		}
		h.addMediaCache(ctx, rowID, content.AvatarURL, nil, nil, "")
	}
}

func (h *HiClient) calculateLocalContent(ctx context.Context, dbEvt *database.Event, evt *event.Event) (*database.LocalContent, []id.ContentURI) {
	if evt.Type != event.EventMessage {
		return nil, nil
	}
	_ = evt.Content.ParseRaw(evt.Type)
	content, ok := evt.Content.Parsed.(*event.MessageEventContent)
	if !ok {
		return nil, nil
	}
	if dbEvt.RelationType == event.RelReplace && content.NewContent != nil {
		content = content.NewContent
	}
	if content != nil {
		var sanitizedHTML, editSource string
		var wasPlaintext, hasMath, bigEmoji bool
		var inlineImages []id.ContentURI
		if content.Format == event.FormatHTML && content.FormattedBody != "" {
			var err error
			sanitizedHTML, inlineImages, err = sanitizeAndLinkifyHTML(content.FormattedBody)
			if err != nil {
				zerolog.Ctx(ctx).Warn().Err(err).
					Stringer("event_id", dbEvt.ID).
					Msg("Failed to sanitize HTML")
			}
			hasMath = strings.Contains(sanitizedHTML, "<hicli-math")
			if len(inlineImages) > 0 && dbEvt.RowID != 0 {
				for _, uri := range inlineImages {
					h.addMediaCache(ctx, dbEvt.RowID, uri.CUString(), nil, nil, "")
				}
				inlineImages = nil
			}
			if dbEvt.LocalContent != nil && dbEvt.LocalContent.EditSource != "" {
				editSource = dbEvt.LocalContent.EditSource
			} else if evt.Sender == h.Account.UserID {
				editSource, _ = format.HTMLToMarkdownFull(htmlToMarkdownForInput, content.FormattedBody)
				if content.MsgType == event.MsgEmote {
					editSource = "/me " + editSource
				} else if content.MsgType == event.MsgNotice {
					editSource = "/notice " + editSource
				}
			}
		} else {
			hasSpecialCharacters := false
			for _, char := range content.Body {
				if char == '<' || char == '>' || char == '&' || char == '.' || char == ':' {
					hasSpecialCharacters = true
					break
				}
			}
			if hasSpecialCharacters {
				var builder strings.Builder
				builder.Grow(len(content.Body) + builderPreallocBuffer)
				linkifyAndWriteBytes(&builder, []byte(content.Body))
				sanitizedHTML = builder.String()
			} else if len(content.Body) < 100 && emojirunes.IsOnlyEmojis(content.Body) {
				bigEmoji = true
			}
			if content.MsgType == event.MsgEmote {
				editSource = "/me " + content.Body
			} else if content.MsgType == event.MsgNotice {
				editSource = "/notice " + content.Body
			}
			wasPlaintext = true
		}
		return &database.LocalContent{
			SanitizedHTML:        sanitizedHTML,
			HTMLVersion:          CurrentHTMLSanitizerVersion,
			WasPlaintext:         wasPlaintext,
			BigEmoji:             bigEmoji,
			HasMath:              hasMath,
			EditSource:           editSource,
			ReplyFallbackRemoved: dbEvt.LocalContent.GetReplyFallbackRemoved(),
		}, inlineImages
	}
	return nil, nil
}

const CurrentHTMLSanitizerVersion = 10

func (h *HiClient) ReprocessExistingEvent(ctx context.Context, evt *database.Event) {
	if (evt.Type != event.EventMessage.Type && evt.DecryptedType != event.EventMessage.Type) ||
		evt.LocalContent == nil || evt.LocalContent.HTMLVersion >= CurrentHTMLSanitizerVersion {
		return
	}
	evt.LocalContent, _ = h.calculateLocalContent(ctx, evt, evt.AsRawMautrix())
	err := h.DB.Event.UpdateLocalContent(ctx, evt)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).
			Stringer("event_id", evt.ID).
			Msg("Failed to update local content")
	}
}

func (h *HiClient) postDecryptProcess(ctx context.Context, llSummary *mautrix.LazyLoadSummary, dbEvt *database.Event, evt *event.Event) (inlineImages []id.ContentURI) {
	if dbEvt.RowID != 0 {
		h.cacheMedia(ctx, evt, dbEvt.RowID)
	}
	if evt.Sender != h.Account.UserID {
		dbEvt.UnreadType = h.evaluatePushRules(ctx, llSummary, dbEvt.GetNonPushUnreadType(), evt)
	}
	dbEvt.LocalContent, inlineImages = h.calculateLocalContent(ctx, dbEvt, evt)
	return
}

func (h *HiClient) fillPrevContent(ctx context.Context, evt *event.Event) error {
	if evt.StateKey != nil && evt.Unsigned.PrevContent == nil && evt.Unsigned.ReplacesState != "" {
		replacesState, err := h.DB.Event.GetByID(ctx, evt.Unsigned.ReplacesState)
		if err != nil {
			return fmt.Errorf("failed to get prev content for %s from %s: %w", evt.ID, evt.Unsigned.ReplacesState, err)
		} else if replacesState != nil {
			evt.Unsigned.PrevContent = &event.Content{VeryRaw: replacesState.Content}
		}
	}
	return nil
}

func (h *HiClient) processEvent(
	ctx context.Context,
	evt *event.Event,
	llSummary *mautrix.LazyLoadSummary,
	decryptionQueue map[id.SessionID]*database.SessionRequest,
	checkDB bool,
) (*database.Event, error) {
	if checkDB {
		dbEvt, err := h.DB.Event.GetByID(ctx, evt.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check if event %s exists: %w", evt.ID, err)
		} else if dbEvt != nil {
			return dbEvt, nil
		}
	}
	if err := h.fillPrevContent(ctx, evt); err != nil {
		return nil, err
	}
	dbEvt := database.MautrixToEvent(evt)
	contentWithoutFallback := removeReplyFallback(evt)
	if contentWithoutFallback != nil {
		dbEvt.Content = contentWithoutFallback
		dbEvt.MarkReplyFallbackRemoved()
	}
	var decryptionErr error
	var decryptedMautrixEvt *event.Event
	if evt.Type == event.EventEncrypted && (dbEvt.RedactedBy == "" || len(dbEvt.Content) > 2) {
		decryptedMautrixEvt, decryptionErr = h.decryptEventInto(ctx, evt, dbEvt)
	} else if evt.Type == event.EventRedaction {
		if evt.Redacts != "" && gjson.GetBytes(evt.Content.VeryRaw, "redacts").Str != evt.Redacts.String() {
			var err error
			evt.Content.VeryRaw, err = sjson.SetBytes(evt.Content.VeryRaw, "redacts", evt.Redacts)
			if err != nil {
				return dbEvt, fmt.Errorf("failed to set redacts field: %w", err)
			}
			dbEvt.Content = evt.Content.VeryRaw
		} else if evt.Redacts == "" {
			evt.Redacts = id.EventID(gjson.GetBytes(evt.Content.VeryRaw, "redacts").Str)
		}
	}
	var inlineImages []id.ContentURI
	if decryptedMautrixEvt != nil {
		inlineImages = h.postDecryptProcess(ctx, llSummary, dbEvt, decryptedMautrixEvt)
	} else {
		inlineImages = h.postDecryptProcess(ctx, llSummary, dbEvt, evt)
	}
	_, err := h.DB.Event.Upsert(ctx, dbEvt)
	if err != nil {
		return dbEvt, fmt.Errorf("failed to save event %s: %w", evt.ID, err)
	}
	if decryptedMautrixEvt != nil {
		h.cacheMedia(ctx, decryptedMautrixEvt, dbEvt.RowID)
	} else {
		h.cacheMedia(ctx, evt, dbEvt.RowID)
	}
	for _, uri := range inlineImages {
		h.addMediaCache(ctx, dbEvt.RowID, uri.CUString(), nil, nil, "")
	}
	if decryptionErr != nil && isDecryptionErrorRetryable(decryptionErr) {
		req, ok := decryptionQueue[dbEvt.MegolmSessionID]
		if !ok {
			req = &database.SessionRequest{
				RoomID:    evt.RoomID,
				SessionID: dbEvt.MegolmSessionID,
				Sender:    evt.Sender,
			}
		}
		minIndex, _ := crypto.ParseMegolmMessageIndex(evt.Content.AsEncrypted().MegolmCiphertext)
		req.MinIndex = min(uint32(minIndex), req.MinIndex)
		if decryptionQueue != nil {
			decryptionQueue[dbEvt.MegolmSessionID] = req
		} else {
			err = h.DB.SessionRequest.Put(ctx, req)
			if err != nil {
				zerolog.Ctx(ctx).Err(err).
					Stringer("session_id", dbEvt.MegolmSessionID).
					Msg("Failed to save session request")
			} else {
				h.WakeupRequestQueue()
			}
		}
	}
	return dbEvt, err
}

var unsetSortingTimestamp = time.UnixMilli(1000000000000)

func (h *HiClient) processStateAndTimeline(
	ctx context.Context,
	room *database.Room,
	state *mautrix.SyncEventsList,
	timeline *mautrix.SyncTimeline,
	summary *mautrix.LazyLoadSummary,
	receipts []*database.Receipt,
	newOwnReceipts []id.EventID,
	accountData map[event.Type]*database.AccountData,
) error {
	updatedRoom := &database.Room{
		ID: room.ID,

		SortingTimestamp: room.SortingTimestamp,
		NameQuality:      room.NameQuality,
		UnreadCounts:     room.UnreadCounts,
	}
	heroesChanged := false
	if summary.Heroes == nil && summary.JoinedMemberCount == nil && summary.InvitedMemberCount == nil {
		summary = room.LazyLoadSummary
	} else if room.LazyLoadSummary == nil ||
		!slices.Equal(summary.Heroes, room.LazyLoadSummary.Heroes) ||
		!intPtrEqual(summary.JoinedMemberCount, room.LazyLoadSummary.JoinedMemberCount) ||
		!intPtrEqual(summary.InvitedMemberCount, room.LazyLoadSummary.InvitedMemberCount) {
		updatedRoom.LazyLoadSummary = summary
		heroesChanged = true
	}
	sdc := &spaceDataCollector{}
	decryptionQueue := make(map[id.SessionID]*database.SessionRequest)
	allNewEvents := make([]*database.Event, 0, len(state.Events)+len(timeline.Events))
	addedEvents := make(map[database.EventRowID]struct{})
	newNotifications := make([]jsoncmd.SyncNotification, 0)
	var recalculatePreviewEvent, unreadMessagesWereMaybeRedacted bool
	var newUnreadCounts database.UnreadCounts
	addOldEvent := func(rowID database.EventRowID, evtID id.EventID) (dbEvt *database.Event, err error) {
		if rowID != 0 {
			dbEvt, err = h.DB.Event.GetByRowID(ctx, rowID)
		} else {
			dbEvt, err = h.DB.Event.GetByID(ctx, evtID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get redaction target: %w", err)
		} else if dbEvt == nil {
			return nil, nil
		}
		_, alreadyAdded := addedEvents[dbEvt.RowID]
		if !alreadyAdded {
			addedEvents[dbEvt.RowID] = struct{}{}
			allNewEvents = append(allNewEvents, dbEvt)
		}
		return dbEvt, nil
	}
	processRedaction := func(evt *event.Event) error {
		dbEvt, err := addOldEvent(0, evt.Redacts)
		if err != nil {
			return fmt.Errorf("failed to get redaction target: %w", err)
		}
		if dbEvt == nil {
			return nil
		}
		if dbEvt.UnreadType > 0 {
			unreadMessagesWereMaybeRedacted = true
		}
		if dbEvt.RelationType == event.RelReplace || dbEvt.RelationType == event.RelAnnotation {
			_, err = addOldEvent(0, dbEvt.RelatesTo)
			if err != nil {
				return fmt.Errorf("failed to get relation target of redaction target: %w", err)
			}
		}
		if updatedRoom.PreviewEventRowID == dbEvt.RowID || (updatedRoom.PreviewEventRowID == 0 && room.PreviewEventRowID == dbEvt.RowID) {
			updatedRoom.PreviewEventRowID = 0
			recalculatePreviewEvent = true
		}
		return nil
	}
	megolmSessionDiscarded := false
	processNewEvent := func(evt *event.Event, isTimeline, isUnread bool) (database.EventRowID, error) {
		evt.RoomID = room.ID
		dbEvt, err := h.processEvent(ctx, evt, summary, decryptionQueue, evt.Unsigned.TransactionID != "")
		if err != nil {
			return -1, err
		}
		if isUnread {
			if dbEvt.UnreadType.Is(database.UnreadTypeNotify) && h.firstSyncReceived {
				newNotifications = append(newNotifications, jsoncmd.SyncNotification{
					RowID:     dbEvt.RowID,
					Sound:     dbEvt.UnreadType.Is(database.UnreadTypeSound),
					Highlight: dbEvt.UnreadType.Is(database.UnreadTypeHighlight),
					Event:     dbEvt,
					Room:      room,
				})
			}
			newUnreadCounts.AddOne(dbEvt.UnreadType)
		}
		if isTimeline {
			if dbEvt.CanUseForPreview() {
				updatedRoom.PreviewEventRowID = dbEvt.RowID
				recalculatePreviewEvent = false
			}
			updatedRoom.BumpSortingTimestamp(dbEvt)
		}
		if evt.StateKey != nil {
			var membership event.Membership
			if evt.Type == event.StateMember {
				membership = event.Membership(gjson.GetBytes(evt.Content.VeryRaw, "membership").Str)
				if summary != nil && slices.Contains(summary.Heroes, id.UserID(*evt.StateKey)) {
					heroesChanged = true
				}
				if !megolmSessionDiscarded && room.EncryptionEvent != nil {
					megolmSessionDiscarded = h.maybeDiscardOutboundSession(ctx, membership, evt)
				}
			} else if evt.Type == event.StateElementFunctionalMembers {
				heroesChanged = true
			}
			err = h.DB.CurrentState.Set(ctx, room.ID, evt.Type, *evt.StateKey, dbEvt.RowID, membership)
			if err != nil {
				return -1, fmt.Errorf("failed to save current state event ID %s for %s/%s: %w", evt.ID, evt.Type.Type, *evt.StateKey, err)
			}
			processImportantEvent(ctx, evt, room, updatedRoom, dbEvt.RowID, sdc)
		}
		allNewEvents = append(allNewEvents, dbEvt)
		addedEvents[dbEvt.RowID] = struct{}{}
		if evt.Type == event.EventRedaction && evt.Redacts != "" {
			err = processRedaction(evt)
			if err != nil {
				return -1, fmt.Errorf("failed to process redaction: %w", err)
			}
		} else if dbEvt.RelationType == event.RelReplace || dbEvt.RelationType == event.RelAnnotation {
			_, err = addOldEvent(0, dbEvt.RelatesTo)
			if err != nil {
				return -1, fmt.Errorf("failed to get relation target of event: %w", err)
			}
		} else if replyTo := dbEvt.GetReplyTo(); replyTo != "" {
			_, err = addOldEvent(0, replyTo)
			if err != nil {
				return -1, fmt.Errorf("failed to get reply target of event: %w", err)
			}
		}
		return dbEvt.RowID, nil
	}
	changedState := make(map[event.Type]map[string]database.EventRowID)
	setNewState := func(evtType event.Type, stateKey string, rowID database.EventRowID) {
		if _, ok := changedState[evtType]; !ok {
			changedState[evtType] = make(map[string]database.EventRowID)
		}
		changedState[evtType][stateKey] = rowID
	}
	for _, evt := range state.Events {
		evt.Type.Class = event.StateEventType
		rowID, err := processNewEvent(evt, false, false)
		if err != nil {
			return err
		}
		setNewState(evt.Type, *evt.StateKey, rowID)
	}
	var timelineRowTuples []database.TimelineRowTuple
	receiptMap := make(map[id.EventID][]*database.Receipt)
	for _, receipt := range receipts {
		if receipt.UserID != h.Account.UserID {
			receiptMap[receipt.EventID] = append(receiptMap[receipt.EventID], receipt)
		}
	}
	var err error
	if len(timeline.Events) > 0 {
		timelineIDs := make([]database.EventRowID, len(timeline.Events))
		encounteredReceiptUsers := make(map[id.UserID]struct{})
		readUpToIndex := -1
		for i := len(timeline.Events) - 1; i >= 0; i-- {
			evt := timeline.Events[i]
			for _, receipt := range receiptMap[evt.ID] {
				encounteredReceiptUsers[receipt.UserID] = struct{}{}
			}
			isRead := slices.Contains(newOwnReceipts, evt.ID)
			isOwnEvent := evt.Sender == h.Account.UserID
			_, alreadyEncountered := encounteredReceiptUsers[evt.Sender]
			if !isOwnEvent && !alreadyEncountered {
				encounteredReceiptUsers[evt.Sender] = struct{}{}
				injectedReceipt := &database.Receipt{
					RoomID:      room.ID,
					UserID:      evt.Sender,
					ReceiptType: event.ReceiptTypeRead,
					EventID:     evt.ID,
					Timestamp:   jsontime.UM(time.UnixMilli(evt.Timestamp)),
				}
				receipts = append(receipts, injectedReceipt)
				receiptMap[evt.ID] = append(receiptMap[evt.ID], injectedReceipt)
			}
			if readUpToIndex == -1 && (isRead || isOwnEvent) {
				readUpToIndex = i
				// Reset unread counts if we see our own read receipt in the timeline.
				// It'll be updated with new unreads (if any) at the end.
				updatedRoom.UnreadCounts = database.UnreadCounts{}
				if !isRead {
					receipts = append(receipts, &database.Receipt{
						RoomID:      room.ID,
						UserID:      h.Account.UserID,
						ReceiptType: event.ReceiptTypeRead,
						EventID:     evt.ID,
						Timestamp:   jsontime.UM(time.UnixMilli(evt.Timestamp)),
					})
					newOwnReceipts = append(newOwnReceipts, evt.ID)
				}
			}
		}
		for i, evt := range timeline.Events {
			if evt.StateKey != nil {
				evt.Type.Class = event.StateEventType
			} else {
				evt.Type.Class = event.MessageEventType
			}
			timelineIDs[i], err = processNewEvent(evt, true, i > readUpToIndex)
			if err != nil {
				return err
			}
			if evt.StateKey != nil {
				setNewState(evt.Type, *evt.StateKey, timelineIDs[i])
			}
		}
		if updatedRoom.SortingTimestamp.Before(unsetSortingTimestamp) && len(timeline.Events) > 0 {
			updatedRoom.SortingTimestamp = jsontime.UM(time.UnixMilli(timeline.Events[len(timeline.Events)-1].Timestamp))
		}
		for _, entry := range decryptionQueue {
			err = h.DB.SessionRequest.Put(ctx, entry)
			if err != nil {
				return fmt.Errorf("failed to save session request for %s: %w", entry.SessionID, err)
			}
		}
		if len(decryptionQueue) > 0 {
			ctx.Value(syncContextKey).(*syncContext).shouldWakeupRequestQueue = true
		}
		if timeline.Limited {
			err = h.DB.Timeline.Clear(ctx, room.ID)
			if err != nil {
				return fmt.Errorf("failed to clear old timeline: %w", err)
			}
			updatedRoom.PrevBatch = timeline.PrevBatch
			h.paginationInterrupterLock.Lock()
			if interrupt, ok := h.paginationInterrupter[room.ID]; ok {
				interrupt(ErrTimelineReset)
			}
			h.paginationInterrupterLock.Unlock()
		}
		timelineRowTuples, err = h.DB.Timeline.Append(ctx, room.ID, timelineIDs)
		if err != nil {
			return fmt.Errorf("failed to append timeline: %w", err)
		}
	} else {
		timelineRowTuples = make([]database.TimelineRowTuple, 0)
	}
	if recalculatePreviewEvent && updatedRoom.PreviewEventRowID == 0 {
		updatedRoom.PreviewEventRowID, err = h.DB.Room.RecalculatePreview(ctx, room.ID)
		if err != nil {
			return fmt.Errorf("failed to recalculate preview event: %w", err)
		} else if updatedRoom.PreviewEventRowID != 0 {
			_, err = addOldEvent(updatedRoom.PreviewEventRowID, "")
			if err != nil {
				return fmt.Errorf("failed to get preview event: %w", err)
			}
		}
	}
	// Calculate name from participants if participants changed and current name was generated from participants, or if the room name was unset
	if (heroesChanged && updatedRoom.NameQuality <= database.NameQualityParticipants) || updatedRoom.NameQuality == database.NameQualityNil {
		name, dmAvatarURL, dmUserID, err := h.calculateRoomParticipantName(ctx, room.ID, summary)
		if err != nil {
			return fmt.Errorf("failed to calculate room name: %w", err)
		}
		updatedRoom.DMUserID = &dmUserID
		updatedRoom.Name = &name
		updatedRoom.NameQuality = database.NameQualityParticipants
		if !dmAvatarURL.IsEmpty() && !room.ExplicitAvatar {
			updatedRoom.Avatar = &dmAvatarURL
		}
	}
	mu, ok := accountData[event.AccountDataMarkedUnread]
	if ok {
		updatedRoom.MarkedUnread = ptr.Ptr(gjson.GetBytes(mu.Content, "unread").Bool())
	}

	if len(receipts) > 0 {
		err = h.DB.Receipt.PutMany(ctx, room.ID, receipts...)
		if err != nil {
			return fmt.Errorf("failed to save receipts: %w", err)
		}
	}
	if !room.UnreadCounts.IsZero() && ((len(newOwnReceipts) > 0 && newUnreadCounts.IsZero()) || unreadMessagesWereMaybeRedacted) {
		updatedRoom.UnreadCounts, err = h.DB.Room.CalculateUnreads(ctx, room.ID, h.Account.UserID)
		if err != nil {
			return fmt.Errorf("failed to recalculate unread counts: %w", err)
		}
	} else {
		updatedRoom.UnreadCounts.Add(newUnreadCounts)
	}
	dismissNotifications := room.UnreadNotifications > 0 && updatedRoom.UnreadNotifications == 0 && len(newNotifications) == 0
	if timeline.PrevBatch != "" && (room.PrevBatch == "" || timeline.Limited) {
		updatedRoom.PrevBatch = timeline.PrevBatch
	}
	roomChanged := updatedRoom.CheckChangesAndCopyInto(room)
	if roomChanged {
		err = h.DB.Room.Upsert(ctx, updatedRoom)
		if err != nil {
			return fmt.Errorf("failed to save room data: %w", err)
		}
	}
	err = sdc.Apply(ctx, room, h.DB.SpaceEdge)
	if err != nil {
		return err
	}
	// TODO why is *old* unread count sometimes zero when processing the read receipt that is making it zero?
	if roomChanged || len(accountData) > 0 || len(newOwnReceipts) > 0 || len(receipts) > 0 || len(timelineRowTuples) > 0 || len(allNewEvents) > 0 {
		for _, receipt := range receipts {
			receipt.RoomID = ""
		}
		ctx.Value(syncContextKey).(*syncContext).evt.Rooms[room.ID] = &jsoncmd.SyncRoom{
			Meta:        room,
			Timeline:    timelineRowTuples,
			AccountData: accountData,
			State:       changedState,
			Reset:       timeline.Limited,
			Events:      allNewEvents,
			Receipts:    receiptMap,

			Notifications:        newNotifications,
			DismissNotifications: dismissNotifications,
		}
	}
	return nil
}

func joinMemberNames(names []string, totalCount int) string {
	if len(names) == 1 {
		return names[0]
	} else if len(names) < 5 || (len(names) == 5 && totalCount <= 6) {
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	} else {
		return fmt.Sprintf("%s and %d others", strings.Join(names[:4], ", "), totalCount-5)
	}
}

func (h *HiClient) calculateRoomParticipantName(ctx context.Context, roomID id.RoomID, summary *mautrix.LazyLoadSummary) (string, id.ContentURI, id.UserID, error) {
	var primaryAvatarURL id.ContentURI
	if summary == nil || len(summary.Heroes) == 0 {
		return "Empty room", primaryAvatarURL, "", nil
	}
	var functionalMembers []id.UserID
	functionalMembersEvt, err := h.DB.CurrentState.Get(ctx, roomID, event.StateElementFunctionalMembers, "")
	if err != nil {
		return "", primaryAvatarURL, "", fmt.Errorf("failed to get %s event: %w", event.StateElementFunctionalMembers.Type, err)
	} else if functionalMembersEvt != nil {
		mautrixEvt := functionalMembersEvt.AsRawMautrix()
		_ = mautrixEvt.Content.ParseRaw(mautrixEvt.Type)
		content, ok := mautrixEvt.Content.Parsed.(*event.ElementFunctionalMembersContent)
		if ok {
			functionalMembers = content.ServiceMembers
		}
	}
	var members, leftMembers []string
	var memberCount int
	if summary.JoinedMemberCount != nil && *summary.JoinedMemberCount > 0 {
		memberCount = *summary.JoinedMemberCount
	} else if summary.InvitedMemberCount != nil {
		memberCount = *summary.InvitedMemberCount
	}
	var dmUserID id.UserID
	for _, hero := range summary.Heroes {
		if slices.Contains(functionalMembers, hero) {
			// TODO save member count so push rule evaluation would use the subtracted one?
			memberCount--
			continue
		} else if len(members) >= 5 {
			break
		}
		if dmUserID == "" {
			dmUserID = hero
		}
		heroEvt, err := h.DB.CurrentState.Get(ctx, roomID, event.StateMember, hero.String())
		if err != nil {
			return "", primaryAvatarURL, "", fmt.Errorf("failed to get %s's member event: %w", hero, err)
		} else if heroEvt == nil {
			leftMembers = append(leftMembers, hero.String())
			continue
		}
		membership := gjson.GetBytes(heroEvt.Content, "membership").Str
		name := gjson.GetBytes(heroEvt.Content, "displayname").Str
		if name == "" {
			name = hero.String()
		}
		avatarURL := gjson.GetBytes(heroEvt.Content, "avatar_url").Str
		if avatarURL != "" {
			primaryAvatarURL = id.ContentURIString(avatarURL).ParseOrIgnore()
		}
		if membership == "join" || membership == "invite" {
			members = append(members, name)
			dmUserID = hero
		} else {
			leftMembers = append(leftMembers, name)
		}
	}
	if !primaryAvatarURL.IsValid() {
		primaryAvatarURL = id.ContentURI{}
	}
	if len(members) > 0 {
		if len(members) > 1 {
			primaryAvatarURL = id.ContentURI{}
			dmUserID = ""
		}
		return joinMemberNames(members, memberCount), primaryAvatarURL, dmUserID, nil
	} else if len(leftMembers) > 0 {
		if len(leftMembers) > 1 {
			primaryAvatarURL = id.ContentURI{}
			dmUserID = ""
		}
		return fmt.Sprintf("Empty room (was %s)", joinMemberNames(leftMembers, memberCount)), primaryAvatarURL, "", nil
	} else {
		return "Empty room", primaryAvatarURL, "", nil
	}
}

func intPtrEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

type spaceDataCollector struct {
	Children          []database.SpaceChildEntry
	Parents           []database.SpaceParentEntry
	RemovedChildren   []id.RoomID
	RemovedParents    []id.RoomID
	PowerLevelChanged bool
	IsFullState       bool
}

func (sdc *spaceDataCollector) Collect(evt *event.Event, rowID database.EventRowID) {
	switch evt.Type {
	case event.StatePowerLevels:
		sdc.PowerLevelChanged = true
	case event.StateCreate:
		sdc.IsFullState = true
	case event.StateSpaceChild:
		content := evt.Content.AsSpaceChild()
		if len(content.Via) == 0 {
			sdc.RemovedChildren = append(sdc.RemovedChildren, id.RoomID(*evt.StateKey))
		} else {
			sdc.Children = append(sdc.Children, database.SpaceChildEntry{
				ChildID:    id.RoomID(*evt.StateKey),
				EventRowID: rowID,
				Order:      content.Order,
				Suggested:  content.Suggested,
			})
		}
	case event.StateSpaceParent:
		content := evt.Content.AsSpaceParent()
		if len(content.Via) == 0 {
			sdc.RemovedParents = append(sdc.RemovedParents, id.RoomID(*evt.StateKey))
		} else {
			sdc.Parents = append(sdc.Parents, database.SpaceParentEntry{
				ParentID:   id.RoomID(*evt.StateKey),
				EventRowID: rowID,
				Canonical:  content.Canonical,
			})
		}
	}
}

func (sdc *spaceDataCollector) Apply(ctx context.Context, room *database.Room, seq *database.SpaceEdgeQuery) error {
	if room.CreationContent == nil || room.CreationContent.Type != event.RoomTypeSpace {
		sdc.Children = nil
		sdc.RemovedChildren = nil
		sdc.PowerLevelChanged = false
	}
	if len(sdc.Children) == 0 && len(sdc.RemovedChildren) == 0 &&
		len(sdc.Parents) == 0 && len(sdc.RemovedParents) == 0 &&
		!sdc.PowerLevelChanged {
		return nil
	}
	return seq.GetDB().DoTxn(ctx, nil, func(ctx context.Context) error {
		if len(sdc.Children) > 0 || len(sdc.RemovedChildren) > 0 {
			err := seq.SetChildren(ctx, room.ID, sdc.Children, sdc.RemovedChildren, sdc.IsFullState)
			if err != nil {
				return fmt.Errorf("failed to set space children: %w", err)
			}
		}
		if len(sdc.Parents) > 0 || len(sdc.RemovedParents) > 0 {
			err := seq.SetParents(ctx, room.ID, sdc.Parents, sdc.RemovedParents, sdc.IsFullState)
			if err != nil {
				return fmt.Errorf("failed to set space parents: %w", err)
			}
			if len(sdc.Parents) > 0 {
				err = seq.RevalidateAllParentsOfRoomValidity(ctx, room.ID)
				if err != nil {
					return fmt.Errorf("failed to revalidate own parent references: %w", err)
				}
			}
		}
		if sdc.PowerLevelChanged {
			err := seq.RevalidateAllChildrenOfParentValidity(ctx, room.ID)
			if err != nil {
				return fmt.Errorf("failed to revalidate child parent references to self: %w", err)
			}
		}
		return nil
	})
}

func processImportantEvent(
	ctx context.Context,
	evt *event.Event,
	existingRoomData, updatedRoom *database.Room,
	rowID database.EventRowID,
	sdc *spaceDataCollector,
) (roomDataChanged bool) {
	if evt.StateKey == nil {
		return
	}
	switch evt.Type {
	case event.StateCreate, event.StateTombstone, event.StateRoomName, event.StateCanonicalAlias,
		event.StateRoomAvatar, event.StateTopic, event.StateEncryption, event.StatePowerLevels:
		if *evt.StateKey != "" {
			return
		}
	case event.StateSpaceChild, event.StateSpaceParent:
		if !strings.HasPrefix(*evt.StateKey, "!") {
			return
		}
	default:
		return
	}
	err := evt.Content.ParseRaw(evt.Type)
	sdc.Collect(evt, rowID)
	if err != nil && !errors.Is(err, event.ErrContentAlreadyParsed) {
		zerolog.Ctx(ctx).Warn().Err(err).
			Stringer("event_type", &evt.Type).
			Stringer("event_id", evt.ID).
			Msg("Failed to parse state event, skipping")
		return
	}
	switch evt.Type {
	case event.StateCreate:
		updatedRoom.CreationContent, _ = evt.Content.Parsed.(*event.CreateEventContent)
	case event.StateTombstone:
		updatedRoom.Tombstone, _ = evt.Content.Parsed.(*event.TombstoneEventContent)
	case event.StateEncryption:
		newEncryption, _ := evt.Content.Parsed.(*event.EncryptionEventContent)
		if existingRoomData.EncryptionEvent == nil || existingRoomData.EncryptionEvent.Algorithm == newEncryption.Algorithm {
			updatedRoom.EncryptionEvent = newEncryption
		}
	case event.StateRoomName:
		content, ok := evt.Content.Parsed.(*event.RoomNameEventContent)
		if ok {
			updatedRoom.Name = &content.Name
			updatedRoom.NameQuality = database.NameQualityExplicit
			if content.Name == "" {
				if updatedRoom.CanonicalAlias != nil && *updatedRoom.CanonicalAlias != "" {
					updatedRoom.Name = (*string)(updatedRoom.CanonicalAlias)
					updatedRoom.NameQuality = database.NameQualityCanonicalAlias
				} else if existingRoomData.CanonicalAlias != nil && *existingRoomData.CanonicalAlias != "" {
					updatedRoom.Name = (*string)(existingRoomData.CanonicalAlias)
					updatedRoom.NameQuality = database.NameQualityCanonicalAlias
				} else {
					updatedRoom.NameQuality = database.NameQualityNil
				}
			}
		}
	case event.StateCanonicalAlias:
		content, ok := evt.Content.Parsed.(*event.CanonicalAliasEventContent)
		if ok {
			updatedRoom.CanonicalAlias = &content.Alias
			if updatedRoom.NameQuality <= database.NameQualityCanonicalAlias {
				updatedRoom.Name = (*string)(&content.Alias)
				updatedRoom.NameQuality = database.NameQualityCanonicalAlias
				if content.Alias == "" {
					updatedRoom.NameQuality = database.NameQualityNil
				}
			}
		}
	case event.StateRoomAvatar:
		content, ok := evt.Content.Parsed.(*event.RoomAvatarEventContent)
		if ok {
			url, _ := content.URL.Parse()
			updatedRoom.Avatar = &url
			updatedRoom.ExplicitAvatar = true
		}
	case event.StateTopic:
		content, ok := evt.Content.Parsed.(*event.TopicEventContent)
		if ok {
			updatedRoom.Topic = &content.Topic
		}
	}
	return
}
