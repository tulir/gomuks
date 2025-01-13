// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/exgjson"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const (
	getEventBaseQuery = `
		SELECT rowid, -1,
		       room_id, event_id, sender, type, state_key, timestamp, content, decrypted, decrypted_type,
		       unsigned, local_content, transaction_id, redacted_by, relates_to, relation_type,
		       megolm_session_id, decryption_error, send_error, reactions, last_edit_rowid, unread_type
		FROM event
	`
	getEventByRowID                  = getEventBaseQuery + `WHERE rowid = $1`
	getManyEventsByRowID             = getEventBaseQuery + `WHERE rowid IN (%s)`
	getEventByID                     = getEventBaseQuery + `WHERE event_id = $1`
	getEventByTransactionID          = getEventBaseQuery + `WHERE transaction_id = $1`
	getFailedEventsByMegolmSessionID = getEventBaseQuery + `WHERE room_id = $1 AND megolm_session_id = $2 AND decryption_error IS NOT NULL`
	insertEventBaseQuery             = `
		INSERT INTO event (
			room_id, event_id, sender, type, state_key, timestamp, content, decrypted, decrypted_type,
			unsigned, local_content, transaction_id, redacted_by, relates_to, relation_type,
			megolm_session_id, decryption_error, send_error, reactions, last_edit_rowid, unread_type
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`
	insertEventQuery = insertEventBaseQuery + `RETURNING rowid`
	upsertEventQuery = insertEventBaseQuery + `
		ON CONFLICT (event_id) DO UPDATE
			SET decrypted=COALESCE(event.decrypted, excluded.decrypted),
			    decrypted_type=COALESCE(event.decrypted_type, excluded.decrypted_type),
			    redacted_by=COALESCE(event.redacted_by, excluded.redacted_by),
			    decryption_error=CASE WHEN COALESCE(event.decrypted, excluded.decrypted) IS NULL THEN COALESCE(excluded.decryption_error, event.decryption_error) END,
			    send_error=excluded.send_error,
				timestamp=excluded.timestamp,
				unsigned=COALESCE(excluded.unsigned, event.unsigned),
				local_content=COALESCE(excluded.local_content, event.local_content)
		ON CONFLICT (transaction_id) DO UPDATE
			SET event_id=excluded.event_id,
				timestamp=excluded.timestamp,
				unsigned=excluded.unsigned
		RETURNING rowid
	`
	updateEventSendErrorQuery        = `UPDATE event SET send_error = $2 WHERE rowid = $1`
	updateEventIDQuery               = `UPDATE event SET event_id = $2, send_error = NULL WHERE rowid=$1`
	updateEventDecryptedQuery        = `UPDATE event SET decrypted = $2, decrypted_type = $3, decryption_error = NULL, unread_type = $4, local_content = $5 WHERE rowid = $1`
	updateEventLocalContentQuery     = `UPDATE event SET local_content = $2 WHERE rowid = $1`
	updateEventEncryptedContentQuery = `UPDATE event SET content = $2, megolm_session_id = $3 WHERE rowid = $1`
	getEventReactionsQuery           = getEventBaseQuery + `
		WHERE room_id = ?
		  AND type = 'm.reaction'
		  AND relation_type = 'm.annotation'
		  AND redacted_by IS NULL
		  AND relates_to IN (%s)
	`
	getEventEditRowIDsQuery = `
		SELECT main.event_id, edit.rowid
		FROM event main
		JOIN event edit ON
			edit.room_id = main.room_id
			AND edit.relates_to = main.event_id
			AND edit.relation_type = 'm.replace'
		AND edit.type = main.type
		AND edit.sender = main.sender
		AND edit.redacted_by IS NULL
		WHERE main.event_id IN (%s)
		ORDER BY main.event_id, edit.timestamp
	`
	setLastEditRowIDQuery = `
		UPDATE event SET last_edit_rowid = $2 WHERE event_id = $1
	`
	updateReactionCountsQuery = `UPDATE event SET reactions = $2 WHERE event_id = $1`
)

type EventQuery struct {
	*dbutil.QueryHelper[*Event]
}

func (eq *EventQuery) GetFailedByMegolmSessionID(ctx context.Context, roomID id.RoomID, sessionID id.SessionID) ([]*Event, error) {
	return eq.QueryMany(ctx, getFailedEventsByMegolmSessionID, roomID, sessionID)
}

func (eq *EventQuery) GetByID(ctx context.Context, eventID id.EventID) (*Event, error) {
	return eq.QueryOne(ctx, getEventByID, eventID)
}

func (eq *EventQuery) GetByTransactionID(ctx context.Context, txnID string) (*Event, error) {
	return eq.QueryOne(ctx, getEventByTransactionID, txnID)
}

func (eq *EventQuery) GetByRowID(ctx context.Context, rowID EventRowID) (*Event, error) {
	return eq.QueryOne(ctx, getEventByRowID, rowID)
}

func (eq *EventQuery) GetByRowIDs(ctx context.Context, rowIDs ...EventRowID) ([]*Event, error) {
	query, params := buildMultiEventGetFunction(nil, rowIDs, getManyEventsByRowID)
	return eq.QueryMany(ctx, query, params...)
}

func (eq *EventQuery) Upsert(ctx context.Context, evt *Event) (rowID EventRowID, err error) {
	err = eq.GetDB().QueryRow(ctx, upsertEventQuery, evt.sqlVariables()...).Scan(&rowID)
	if err == nil {
		evt.RowID = rowID
	}
	return
}

func (eq *EventQuery) Insert(ctx context.Context, evt *Event) (rowID EventRowID, err error) {
	err = eq.GetDB().QueryRow(ctx, insertEventQuery, evt.sqlVariables()...).Scan(&rowID)
	if err == nil {
		evt.RowID = rowID
	}
	return
}

var stateEventMassInserter = dbutil.NewMassInsertBuilder[*Event, [1]any](
	strings.ReplaceAll(upsertEventQuery, "($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)", "($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"),
	"($1, $%d, $%d, $%d, $%d, $%d, $%d, NULL, NULL, $%d, NULL, $%d, $%d, NULL, NULL, NULL, NULL, NULL, '{}', 0, 0)",
)

var massInsertConverter = dbutil.ConvertRowFn[EventRowID](dbutil.ScanSingleColumn[EventRowID])

func (e *Event) GetMassInsertValues() [9]any {
	return [9]any{
		e.ID, e.Sender, e.Type, e.StateKey, e.Timestamp.UnixMilli(),
		unsafeJSONString(e.Content), unsafeJSONString(e.Unsigned),
		dbutil.StrPtr(e.TransactionID), dbutil.StrPtr(e.RedactedBy),
	}
}

func (eq *EventQuery) MassUpsertState(ctx context.Context, evts []*Event) error {
	for chunk := range slices.Chunk(evts, 500) {
		query, params := stateEventMassInserter.Build([1]any{chunk[0].RoomID}, chunk)
		i := 0
		err := massInsertConverter.
			NewRowIter(eq.GetDB().Query(ctx, query, params...)).
			Iter(func(t EventRowID) (bool, error) {
				chunk[i].RowID = t
				i++
				return true, nil
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func (eq *EventQuery) UpdateID(ctx context.Context, rowID EventRowID, newID id.EventID) error {
	return eq.Exec(ctx, updateEventIDQuery, rowID, newID)
}

func (eq *EventQuery) UpdateSendError(ctx context.Context, rowID EventRowID, sendError string) error {
	return eq.Exec(ctx, updateEventSendErrorQuery, rowID, sendError)
}

func (eq *EventQuery) UpdateDecrypted(ctx context.Context, evt *Event) error {
	return eq.Exec(
		ctx,
		updateEventDecryptedQuery,
		evt.RowID,
		unsafeJSONString(evt.Decrypted),
		evt.DecryptedType,
		evt.UnreadType,
		dbutil.JSONPtr(evt.LocalContent),
	)
}

func (eq *EventQuery) UpdateLocalContent(ctx context.Context, evt *Event) error {
	return eq.Exec(ctx, updateEventLocalContentQuery, evt.RowID, dbutil.JSONPtr(evt.LocalContent))
}

func (eq *EventQuery) UpdateEncryptedContent(ctx context.Context, evt *Event) error {
	return eq.Exec(ctx, updateEventEncryptedContentQuery, evt.RowID, unsafeJSONString(evt.Content), evt.MegolmSessionID)
}

func (eq *EventQuery) FillReactionCounts(ctx context.Context, roomID id.RoomID, events []*Event) error {
	eventIDs := make([]id.EventID, 0, len(events))
	eventMap := make(map[id.EventID]*Event)
	for _, evt := range events {
		if evt.Reactions == nil {
			eventIDs = append(eventIDs, evt.ID)
			eventMap[evt.ID] = evt
		}
	}
	if len(eventIDs) == 0 {
		return nil
	}
	result, err := eq.GetReactions(ctx, roomID, eventIDs...)
	if err != nil {
		return err
	}
	for evtID, res := range result {
		eventMap[evtID].Reactions = res.Counts
	}
	return nil
}

func (eq *EventQuery) FillLastEditRowIDs(ctx context.Context, roomID id.RoomID, events []*Event) error {
	eventIDs := make([]id.EventID, len(events))
	eventMap := make(map[id.EventID]*Event)
	for i, evt := range events {
		if evt.LastEditRowID == nil {
			eventIDs[i] = evt.ID
			eventMap[evt.ID] = evt
		}
	}
	return eq.GetDB().DoTxn(ctx, nil, func(ctx context.Context) error {
		result, err := eq.GetEditRowIDs(ctx, roomID, eventIDs...)
		if err != nil {
			return err
		}
		for evtID, res := range result {
			lastEditRowID := res[len(res)-1]
			eventMap[evtID].LastEditRowID = &lastEditRowID
			delete(eventMap, evtID)
			err = eq.Exec(ctx, setLastEditRowIDQuery, evtID, lastEditRowID)
			if err != nil {
				return err
			}
		}
		var zero EventRowID
		for evtID, evt := range eventMap {
			evt.LastEditRowID = &zero
			err = eq.Exec(ctx, setLastEditRowIDQuery, evtID, zero)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

var reactionKeyPath = exgjson.Path("m.relates_to", "key")

type GetReactionsResult struct {
	Events []*Event
	Counts map[string]int
}

func buildMultiEventGetFunction[T any](preParams []any, eventIDs []T, query string) (string, []any) {
	params := make([]any, len(preParams)+len(eventIDs))
	copy(params, preParams)
	for i, evtID := range eventIDs {
		params[i+len(preParams)] = evtID
	}
	placeholders := strings.Repeat("?,", len(eventIDs))
	placeholders = placeholders[:len(placeholders)-1]
	return fmt.Sprintf(query, placeholders), params
}

type editRowIDTuple struct {
	eventID   id.EventID
	editRowID EventRowID
}

func (eq *EventQuery) GetEditRowIDs(ctx context.Context, roomID id.RoomID, eventIDs ...id.EventID) (map[id.EventID][]EventRowID, error) {
	query, params := buildMultiEventGetFunction([]any{roomID}, eventIDs, getEventEditRowIDsQuery)
	rows, err := eq.GetDB().Query(ctx, query, params...)
	output := make(map[id.EventID][]EventRowID)
	return output, dbutil.NewRowIterWithError(rows, func(row dbutil.Scannable) (tuple editRowIDTuple, err error) {
		err = row.Scan(&tuple.eventID, &tuple.editRowID)
		return
	}, err).Iter(func(tuple editRowIDTuple) (bool, error) {
		output[tuple.eventID] = append(output[tuple.eventID], tuple.editRowID)
		return true, nil
	})
}

func (eq *EventQuery) GetReactions(ctx context.Context, roomID id.RoomID, eventIDs ...id.EventID) (map[id.EventID]*GetReactionsResult, error) {
	result := make(map[id.EventID]*GetReactionsResult, len(eventIDs))
	for _, evtID := range eventIDs {
		result[evtID] = &GetReactionsResult{Counts: make(map[string]int)}
	}
	return result, eq.GetDB().DoTxn(ctx, nil, func(ctx context.Context) error {
		query, params := buildMultiEventGetFunction([]any{roomID}, eventIDs, getEventReactionsQuery)
		events, err := eq.QueryMany(ctx, query, params...)
		if err != nil {
			return err
		} else if len(events) == 0 {
			return nil
		}
		for _, evt := range events {
			dest := result[evt.RelatesTo]
			dest.Events = append(dest.Events, evt)
			keyRes := gjson.GetBytes(evt.Content, reactionKeyPath)
			if keyRes.Type == gjson.String {
				dest.Counts[keyRes.Str]++
			}
		}
		for evtID, res := range result {
			if len(res.Counts) > 0 {
				err = eq.Exec(ctx, updateReactionCountsQuery, evtID, dbutil.JSON{Data: &res.Counts})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

type EventRowID int64

func (m EventRowID) GetMassInsertValues() [1]any {
	return [1]any{m}
}

type LocalContent struct {
	SanitizedHTML        string `json:"sanitized_html,omitempty"`
	HTMLVersion          int    `json:"html_version,omitempty"`
	WasPlaintext         bool   `json:"was_plaintext,omitempty"`
	BigEmoji             bool   `json:"big_emoji,omitempty"`
	HasMath              bool   `json:"has_math,omitempty"`
	EditSource           string `json:"edit_source,omitempty"`
	ReplyFallbackRemoved bool   `json:"reply_fallback_removed,omitempty"`
}

func (c *LocalContent) GetReplyFallbackRemoved() bool {
	return c != nil && c.ReplyFallbackRemoved
}

type Event struct {
	RowID         EventRowID    `json:"rowid"`
	TimelineRowID TimelineRowID `json:"timeline_rowid"`

	RoomID    id.RoomID          `json:"room_id"`
	ID        id.EventID         `json:"event_id"`
	Sender    id.UserID          `json:"sender"`
	Type      string             `json:"type"`
	StateKey  *string            `json:"state_key,omitempty"`
	Timestamp jsontime.UnixMilli `json:"timestamp"`

	Content       json.RawMessage `json:"content"`
	Decrypted     json.RawMessage `json:"decrypted,omitempty"`
	DecryptedType string          `json:"decrypted_type,omitempty"`
	Unsigned      json.RawMessage `json:"unsigned,omitempty"`
	LocalContent  *LocalContent   `json:"local_content,omitempty"`

	TransactionID string `json:"transaction_id,omitempty"`

	RedactedBy   id.EventID         `json:"redacted_by,omitempty"`
	RelatesTo    id.EventID         `json:"relates_to,omitempty"`
	RelationType event.RelationType `json:"relation_type,omitempty"`

	MegolmSessionID id.SessionID `json:"-"`
	DecryptionError string       `json:"decryption_error,omitempty"`
	SendError       string       `json:"send_error,omitempty"`

	Reactions     map[string]int `json:"reactions,omitempty"`
	LastEditRowID *EventRowID    `json:"last_edit_rowid,omitempty"`
	UnreadType    UnreadType     `json:"unread_type,omitempty"`
}

func MautrixToEvent(evt *event.Event) *Event {
	dbEvt := &Event{
		RoomID:          evt.RoomID,
		ID:              evt.ID,
		Sender:          evt.Sender,
		Type:            evt.Type.Type,
		StateKey:        evt.StateKey,
		Timestamp:       jsontime.UM(time.UnixMilli(evt.Timestamp)),
		Content:         evt.Content.VeryRaw,
		MegolmSessionID: getMegolmSessionID(evt),
		TransactionID:   evt.Unsigned.TransactionID,
		Reactions:       make(map[string]int),
	}
	if !strings.HasPrefix(dbEvt.TransactionID, "hicli-mautrix-go_") {
		dbEvt.TransactionID = ""
	}
	dbEvt.RelatesTo, dbEvt.RelationType = getRelatesToFromEvent(evt)
	dbEvt.Unsigned, _ = json.Marshal(&evt.Unsigned)
	if evt.Unsigned.RedactedBecause != nil {
		dbEvt.RedactedBy = evt.Unsigned.RedactedBecause.ID
	}
	return dbEvt
}

func (e *Event) AsRawMautrix() *event.Event {
	if e == nil {
		return nil
	}
	evt := &event.Event{
		RoomID:    e.RoomID,
		ID:        e.ID,
		Sender:    e.Sender,
		Type:      event.Type{Type: e.Type, Class: event.MessageEventType},
		StateKey:  e.StateKey,
		Timestamp: e.Timestamp.UnixMilli(),
		Content:   event.Content{VeryRaw: e.Content},
	}
	if e.Decrypted != nil {
		evt.Content.VeryRaw = e.Decrypted
		evt.Type.Type = e.DecryptedType
		evt.Mautrix.WasEncrypted = true
	}
	if e.StateKey != nil {
		evt.Type.Class = event.StateEventType
	}
	_ = json.Unmarshal(e.Unsigned, &evt.Unsigned)
	return evt
}

func (e *Event) Scan(row dbutil.Scannable) (*Event, error) {
	var timestamp int64
	var transactionID, redactedBy, relatesTo, relationType, megolmSessionID, decryptionError, sendError, decryptedType sql.NullString
	err := row.Scan(
		&e.RowID,
		&e.TimelineRowID,
		&e.RoomID,
		&e.ID,
		&e.Sender,
		&e.Type,
		&e.StateKey,
		&timestamp,
		(*[]byte)(&e.Content),
		(*[]byte)(&e.Decrypted),
		&decryptedType,
		(*[]byte)(&e.Unsigned),
		dbutil.JSON{Data: &e.LocalContent},
		&transactionID,
		&redactedBy,
		&relatesTo,
		&relationType,
		&megolmSessionID,
		&decryptionError,
		&sendError,
		dbutil.JSON{Data: &e.Reactions},
		&e.LastEditRowID,
		&e.UnreadType,
	)
	if err != nil {
		return nil, err
	}
	e.Timestamp = jsontime.UM(time.UnixMilli(timestamp))
	e.TransactionID = transactionID.String
	e.RedactedBy = id.EventID(redactedBy.String)
	e.RelatesTo = id.EventID(relatesTo.String)
	e.RelationType = event.RelationType(relationType.String)
	e.MegolmSessionID = id.SessionID(megolmSessionID.String)
	e.DecryptedType = decryptedType.String
	e.DecryptionError = decryptionError.String
	e.SendError = sendError.String
	return e, nil
}

var relatesToPath = exgjson.Path("m.relates_to", "event_id")
var relationTypePath = exgjson.Path("m.relates_to", "rel_type")
var replyToPath = exgjson.Path("m.relates_to", "m.in_reply_to", "event_id")

func getRelatesToFromEvent(evt *event.Event) (id.EventID, event.RelationType) {
	if evt.StateKey != nil {
		return "", ""
	}
	return GetRelatesToFromBytes(evt.Content.VeryRaw)
}

func GetRelatesToFromBytes(content []byte) (id.EventID, event.RelationType) {
	results := gjson.GetManyBytes(content, relatesToPath, relationTypePath)
	if len(results) == 2 && results[0].Exists() && results[1].Exists() && results[0].Type == gjson.String && results[1].Type == gjson.String {
		return id.EventID(results[0].Str), event.RelationType(results[1].Str)
	}
	return "", ""
}

func getMegolmSessionID(evt *event.Event) id.SessionID {
	if evt.Type != event.EventEncrypted {
		return ""
	}
	res := gjson.GetBytes(evt.Content.VeryRaw, "session_id")
	if res.Exists() && res.Type == gjson.String {
		return id.SessionID(res.Str)
	}
	return ""
}

func (e *Event) GetReplyTo() id.EventID {
	content := e.Content
	if e.Decrypted != nil {
		content = e.Decrypted
	}
	result := gjson.GetBytes(content, replyToPath)
	if result.Type == gjson.String {
		return id.EventID(result.Str)
	}
	return ""
}

func (e *Event) sqlVariables() []any {
	var reactions any
	if e.Reactions != nil {
		reactions = e.Reactions
	}
	return []any{
		e.RoomID,
		e.ID,
		e.Sender,
		e.Type,
		e.StateKey,
		e.Timestamp.UnixMilli(),
		unsafeJSONString(e.Content),
		unsafeJSONString(e.Decrypted),
		dbutil.StrPtr(e.DecryptedType),
		unsafeJSONString(e.Unsigned),
		dbutil.JSONPtr(e.LocalContent),
		dbutil.StrPtr(e.TransactionID),
		dbutil.StrPtr(e.RedactedBy),
		dbutil.StrPtr(e.RelatesTo),
		dbutil.StrPtr(e.RelationType),
		dbutil.StrPtr(e.MegolmSessionID),
		dbutil.StrPtr(e.DecryptionError),
		dbutil.StrPtr(e.SendError),
		dbutil.JSON{Data: reactions},
		e.LastEditRowID,
		e.UnreadType,
	}
}

func (e *Event) GetNonPushUnreadType() UnreadType {
	if e.RelationType == event.RelReplace || e.RedactedBy != "" {
		return UnreadTypeNone
	}
	switch e.Type {
	case event.EventMessage.Type, event.EventSticker.Type, event.EventUnstablePollStart.Type:
		return UnreadTypeNormal
	case event.EventEncrypted.Type:
		switch e.DecryptedType {
		case event.EventMessage.Type, event.EventSticker.Type, event.EventUnstablePollStart.Type:
			return UnreadTypeNormal
		}
	}
	return UnreadTypeNone
}

func (e *Event) CanUseForPreview() bool {
	return (e.Type == event.EventMessage.Type || e.Type == event.EventSticker.Type ||
		(e.Type == event.EventEncrypted.Type &&
			(e.DecryptedType == event.EventMessage.Type || e.DecryptedType == event.EventSticker.Type))) &&
		e.RelationType != event.RelReplace && e.RedactedBy == ""
}

func (e *Event) BumpsSortingTimestamp() bool {
	return (e.Type == event.EventMessage.Type || e.Type == event.EventSticker.Type || e.Type == event.EventEncrypted.Type) &&
		e.RelationType != event.RelReplace
}

func (e *Event) MarkReplyFallbackRemoved() {
	if e.LocalContent == nil {
		e.LocalContent = &LocalContent{}
	}
	e.LocalContent.ReplyFallbackRemoved = true
}
