// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const (
	upsertReceiptQuery = `
		INSERT INTO receipt (room_id, user_id, receipt_type, thread_id, event_id, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (room_id, user_id, receipt_type, thread_id) DO UPDATE
			SET event_id = excluded.event_id,
			    timestamp = excluded.timestamp
	`
	getReadReceiptsQuery = `SELECT room_id, user_id, receipt_type, thread_id, event_id, timestamp FROM receipt WHERE room_id = $1 AND receipt_type='m.read' AND event_id IN ($2)`
)

var receiptMassInserter = dbutil.NewMassInsertBuilder[*Receipt, [1]any](upsertReceiptQuery, "($1, $%d, $%d, $%d, $%d, $%d)")

type ReceiptQuery struct {
	*dbutil.QueryHelper[*Receipt]
}

func (rq *ReceiptQuery) Put(ctx context.Context, receipt *Receipt) error {
	return rq.Exec(ctx, upsertReceiptQuery, receipt.sqlVariables()...)
}

func (rq *ReceiptQuery) PutMany(ctx context.Context, roomID id.RoomID, receipts ...*Receipt) error {
	if len(receipts) > 1000 {
		return rq.GetDB().DoTxn(ctx, nil, func(ctx context.Context) error {
			for receiptChunk := range slices.Chunk(receipts, 200) {
				err := rq.PutMany(ctx, roomID, receiptChunk...)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	query, params := receiptMassInserter.Build([1]any{roomID}, receipts)
	return rq.Exec(ctx, query, params...)
}

func (rq *ReceiptQuery) GetManyRead(ctx context.Context, roomID id.RoomID, eventIDs []id.EventID) (map[id.EventID][]*Receipt, error) {
	args := make([]any, len(eventIDs)+1)
	placeholders := make([]string, len(eventIDs)+1)
	args[0] = roomID
	placeholders[0] = "?1"
	for i, evtID := range eventIDs {
		args[i+1] = evtID
		placeholders[i+1] = fmt.Sprintf("?%d", i+2)
	}
	query := strings.Replace(getReadReceiptsQuery, "$2", strings.Join(placeholders, ", "), 1)
	output := make(map[id.EventID][]*Receipt)
	err := rq.QueryManyIter(ctx, query, args...).Iter(func(receipt *Receipt) (bool, error) {
		output[receipt.EventID] = append(output[receipt.EventID], receipt)
		return true, nil
	})
	return output, err
}

type Receipt struct {
	RoomID      id.RoomID          `json:"room_id,omitempty"`
	UserID      id.UserID          `json:"user_id"`
	ReceiptType event.ReceiptType  `json:"receipt_type"`
	ThreadID    event.ThreadID     `json:"thread_id,omitempty"`
	EventID     id.EventID         `json:"event_id"`
	Timestamp   jsontime.UnixMilli `json:"timestamp"`
}

func (r *Receipt) Scan(row dbutil.Scannable) (*Receipt, error) {
	var ts int64
	err := row.Scan(&r.RoomID, &r.UserID, &r.ReceiptType, &r.ThreadID, &r.EventID, &ts)
	if err != nil {
		return nil, err
	}
	r.Timestamp = jsontime.UM(time.UnixMilli(ts))
	return r, nil
}

func (r *Receipt) sqlVariables() []any {
	return []any{r.RoomID, r.UserID, r.ReceiptType, r.ThreadID, r.EventID, r.Timestamp.UnixMilli()}
}

func (r *Receipt) GetMassInsertValues() [5]any {
	return [5]any{r.UserID, r.ReceiptType, r.ThreadID, r.EventID, r.Timestamp.UnixMilli()}
}
