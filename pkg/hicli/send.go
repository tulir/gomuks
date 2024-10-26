// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/yuin/goldmark"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/rainbow"
)

var (
	rainbowWithHTML = goldmark.New(format.Extensions, format.HTMLOptions, goldmark.WithExtensions(rainbow.Extension))
)

func (h *HiClient) SendMessage(
	ctx context.Context,
	roomID id.RoomID,
	base *event.MessageEventContent,
	text string,
	relatesTo *event.RelatesTo,
	mentions *event.Mentions,
) (*database.Event, error) {
	var content event.MessageEventContent
	if strings.HasPrefix(text, "/rainbow ") {
		text = strings.TrimPrefix(text, "/rainbow ")
		content = format.RenderMarkdownCustom(text, rainbowWithHTML)
		content.FormattedBody = rainbow.ApplyColor(content.FormattedBody)
	} else if strings.HasPrefix(text, "/plain ") {
		text = strings.TrimPrefix(text, "/plain ")
		content = format.RenderMarkdown(text, false, false)
	} else if strings.HasPrefix(text, "/html ") {
		text = strings.TrimPrefix(text, "/html ")
		content = format.RenderMarkdown(text, false, true)
	} else if text != "" {
		content = format.RenderMarkdown(text, true, true)
	}
	if base != nil {
		if text != "" {
			base.Body = content.Body
			base.Format = content.Format
			base.FormattedBody = content.FormattedBody
		}
		content = *base
	}
	if content.Mentions == nil {
		content.Mentions = &event.Mentions{}
	}
	if mentions != nil {
		content.Mentions.Room = mentions.Room
		for _, userID := range mentions.UserIDs {
			if userID != h.Account.UserID {
				content.Mentions.Add(userID)
			}
		}
	}
	if relatesTo != nil {
		if relatesTo.Type == event.RelReplace {
			contentCopy := content
			content = event.MessageEventContent{
				Body:       "",
				MsgType:    contentCopy.MsgType,
				URL:        contentCopy.URL,
				NewContent: &contentCopy,
				RelatesTo:  relatesTo,
			}
			if contentCopy.File != nil {
				content.URL = contentCopy.File.URL
			}
		} else {
			content.RelatesTo = relatesTo
		}
	}
	return h.Send(ctx, roomID, event.EventMessage, &content)
}

func (h *HiClient) MarkRead(ctx context.Context, roomID id.RoomID, eventID id.EventID, receiptType event.ReceiptType) error {
	content := &mautrix.ReqSetReadMarkers{
		FullyRead: eventID,
	}
	if receiptType == event.ReceiptTypeRead {
		content.Read = eventID
	} else if receiptType == event.ReceiptTypeReadPrivate {
		content.ReadPrivate = eventID
	} else {
		return fmt.Errorf("invalid receipt type: %v", receiptType)
	}
	err := h.Client.SetReadMarkers(ctx, roomID, content)
	if err != nil {
		return fmt.Errorf("failed to mark event as read: %w", err)
	}
	return nil
}

func (h *HiClient) SetTyping(ctx context.Context, roomID id.RoomID, timeout time.Duration) error {
	_, err := h.Client.UserTyping(ctx, roomID, timeout > 0, timeout)
	return err
}

func (h *HiClient) SetState(
	ctx context.Context,
	roomID id.RoomID,
	evtType event.Type,
	stateKey string,
	content any,
) (id.EventID, error) {
	room, err := h.DB.Room.Get(ctx, roomID)
	if err != nil {
		return "", fmt.Errorf("failed to get room metadata: %w", err)
	} else if room == nil {
		return "", fmt.Errorf("unknown room")
	}
	resp, err := h.Client.SendStateEvent(ctx, room.ID, evtType, stateKey, content)
	if err != nil {
		return "", err
	}
	return resp.EventID, nil
}

func (h *HiClient) Send(
	ctx context.Context,
	roomID id.RoomID,
	evtType event.Type,
	content any,
) (*database.Event, error) {
	room, err := h.DB.Room.Get(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room metadata: %w", err)
	} else if room == nil {
		return nil, fmt.Errorf("unknown room")
	}
	txnID := "hicli-" + h.Client.TxnID()
	dbEvt := &database.Event{
		RoomID:          room.ID,
		ID:              id.EventID(fmt.Sprintf("~%s", txnID)),
		Sender:          h.Account.UserID,
		Timestamp:       jsontime.UnixMilliNow(),
		Unsigned:        []byte("{}"),
		TransactionID:   txnID,
		DecryptionError: "",
		SendError:       "not sent",
		Reactions:       map[string]int{},
		LastEditRowID:   ptr.Ptr(database.EventRowID(0)),
	}
	if room.EncryptionEvent != nil && evtType != event.EventReaction {
		dbEvt.Type = event.EventEncrypted.Type
		dbEvt.DecryptedType = evtType.Type
		dbEvt.Decrypted, err = json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event content: %w", err)
		}
		dbEvt.Content = json.RawMessage("{}")
		dbEvt.RelatesTo, dbEvt.RelationType = database.GetRelatesToFromBytes(dbEvt.Decrypted)
	} else {
		dbEvt.Type = evtType.Type
		dbEvt.Content, err = json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event content: %w", err)
		}
		dbEvt.RelatesTo, dbEvt.RelationType = database.GetRelatesToFromBytes(dbEvt.Content)
	}
	var inlineImages []id.ContentURI
	mautrixEvt := dbEvt.AsRawMautrix()
	dbEvt.LocalContent, inlineImages = h.calculateLocalContent(ctx, dbEvt, mautrixEvt)
	_, err = h.DB.Event.Insert(ctx, dbEvt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert event into database: %w", err)
	}
	h.cacheMedia(ctx, mautrixEvt, dbEvt.RowID)
	for _, uri := range inlineImages {
		h.addMediaCache(ctx, dbEvt.RowID, uri.CUString(), nil, nil, "")
	}
	ctx = context.WithoutCancel(ctx)
	go func() {
		err := h.SetTyping(ctx, room.ID, 0)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to stop typing while sending message")
		}
	}()
	go func() {
		var err error
		defer func() {
			if dbEvt.SendError != "" {
				err2 := h.DB.Event.UpdateSendError(ctx, dbEvt.RowID, dbEvt.SendError)
				if err2 != nil {
					zerolog.Ctx(ctx).Err(err2).AnErr("send_error", err).
						Msg("Failed to update send error in database after sending failed")
				}
			}
			h.EventHandler(&SendComplete{
				Event: dbEvt,
				Error: err,
			})
		}()
		if dbEvt.Decrypted != nil {
			var encryptedContent *event.EncryptedEventContent
			encryptedContent, err = h.Encrypt(ctx, room, evtType, dbEvt.Decrypted)
			if err != nil {
				dbEvt.SendError = fmt.Sprintf("failed to encrypt: %v", err)
				zerolog.Ctx(ctx).Err(err).Msg("Failed to encrypt event")
				return
			}
			evtType = event.EventEncrypted
			dbEvt.MegolmSessionID = encryptedContent.SessionID
			dbEvt.Content, err = json.Marshal(encryptedContent)
			if err != nil {
				dbEvt.SendError = fmt.Sprintf("failed to marshal encrypted content: %v", err)
				zerolog.Ctx(ctx).Err(err).Msg("Failed to marshal encrypted content")
				return
			}
			err = h.DB.Event.UpdateEncryptedContent(ctx, dbEvt)
			if err != nil {
				dbEvt.SendError = fmt.Sprintf("failed to save event after encryption: %v", err)
				zerolog.Ctx(ctx).Err(err).Msg("Failed to save event after encryption")
				return
			}
		}
		var resp *mautrix.RespSendEvent
		resp, err = h.Client.SendMessageEvent(ctx, room.ID, evtType, dbEvt.Content, mautrix.ReqSendEvent{
			Timestamp:     dbEvt.Timestamp.UnixMilli(),
			TransactionID: txnID,
			DontEncrypt:   true,
		})
		if err != nil {
			dbEvt.SendError = err.Error()
			err = fmt.Errorf("failed to send event: %w", err)
			return
		}
		dbEvt.ID = resp.EventID
		err = h.DB.Event.UpdateID(ctx, dbEvt.RowID, dbEvt.ID)
		if err != nil {
			err = fmt.Errorf("failed to update event ID in database: %w", err)
		}
	}()
	return dbEvt, nil
}

func (h *HiClient) Encrypt(ctx context.Context, room *database.Room, evtType event.Type, content any) (encrypted *event.EncryptedEventContent, err error) {
	h.encryptLock.Lock()
	defer h.encryptLock.Unlock()
	encrypted, err = h.Crypto.EncryptMegolmEvent(ctx, room.ID, evtType, content)
	if errors.Is(err, crypto.SessionExpired) || errors.Is(err, crypto.NoGroupSession) || errors.Is(err, crypto.SessionNotShared) {
		if err = h.shareGroupSession(ctx, room); err != nil {
			err = fmt.Errorf("failed to share group session: %w", err)
		} else if encrypted, err = h.Crypto.EncryptMegolmEvent(ctx, room.ID, evtType, content); err != nil {
			err = fmt.Errorf("failed to encrypt event after re-sharing group session: %w", err)
		}
	}
	return
}

func (h *HiClient) EnsureGroupSessionShared(ctx context.Context, roomID id.RoomID) error {
	h.encryptLock.Lock()
	defer h.encryptLock.Unlock()
	if session, err := h.CryptoStore.GetOutboundGroupSession(ctx, roomID); err != nil {
		return fmt.Errorf("failed to get previous outbound group session: %w", err)
	} else if session != nil && session.Shared && !session.Expired() {
		return nil
	} else if roomMeta, err := h.DB.Room.Get(ctx, roomID); err != nil {
		return fmt.Errorf("failed to get room metadata: %w", err)
	} else if roomMeta == nil {
		return fmt.Errorf("unknown room")
	} else {
		return h.shareGroupSession(ctx, roomMeta)
	}
}

func (h *HiClient) loadMembers(ctx context.Context, room *database.Room) error {
	if room.HasMemberList {
		return nil
	}
	resp, err := h.Client.Members(ctx, room.ID)
	if err != nil {
		return fmt.Errorf("failed to get room member list: %w", err)
	}
	err = h.DB.DoTxn(ctx, nil, func(ctx context.Context) error {
		entries := make([]*database.CurrentStateEntry, len(resp.Chunk))
		for i, evt := range resp.Chunk {
			dbEvt, err := h.processEvent(ctx, evt, nil, nil, true)
			if err != nil {
				return err
			}
			entries[i] = &database.CurrentStateEntry{
				EventType:  evt.Type,
				StateKey:   *evt.StateKey,
				EventRowID: dbEvt.RowID,
				Membership: event.Membership(evt.Content.Raw["membership"].(string)),
			}
		}
		err := h.DB.CurrentState.AddMany(ctx, room.ID, false, entries)
		if err != nil {
			return err
		}
		return h.DB.Room.Upsert(ctx, &database.Room{
			ID:            room.ID,
			HasMemberList: true,
		})
	})
	if err != nil {
		return fmt.Errorf("failed to process room member list: %w", err)
	}
	return nil
}

func (h *HiClient) shareGroupSession(ctx context.Context, room *database.Room) error {
	err := h.loadMembers(ctx, room)
	if err != nil {
		return err
	}
	shareToInvited := h.shouldShareKeysToInvitedUsers(ctx, room.ID)
	var users []id.UserID
	if shareToInvited {
		users, err = h.ClientStore.GetRoomJoinedOrInvitedMembers(ctx, room.ID)
	} else {
		users, err = h.ClientStore.GetRoomJoinedMembers(ctx, room.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to get room member list: %w", err)
	} else if err = h.Crypto.ShareGroupSession(ctx, room.ID, users); err != nil {
		return fmt.Errorf("failed to share group session: %w", err)
	}
	return nil
}

func (h *HiClient) shouldShareKeysToInvitedUsers(ctx context.Context, roomID id.RoomID) bool {
	historyVisibility, err := h.DB.CurrentState.Get(ctx, roomID, event.StateHistoryVisibility, "")
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get history visibility event")
		return false
	}
	mautrixEvt := historyVisibility.AsRawMautrix()
	err = mautrixEvt.Content.ParseRaw(mautrixEvt.Type)
	if err != nil && !errors.Is(err, event.ErrContentAlreadyParsed) {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to parse history visibility event")
		return false
	}
	hv, ok := mautrixEvt.Content.Parsed.(*event.HistoryVisibilityEventContent)
	if !ok {
		zerolog.Ctx(ctx).Warn().Msg("Unexpected parsed content type for history visibility event")
		return false
	}
	return hv.HistoryVisibility == event.HistoryVisibilityInvited ||
		hv.HistoryVisibility == event.HistoryVisibilityShared ||
		hv.HistoryVisibility == event.HistoryVisibilityWorldReadable
}
