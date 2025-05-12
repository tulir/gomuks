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
	"unsafe"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const (
	upsertAccountDataQuery = `
		INSERT INTO account_data (user_id, type, content) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, type) DO UPDATE SET content = excluded.content
	`
	upsertRoomAccountDataQuery = `
		INSERT INTO room_account_data (user_id, room_id, type, content) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, room_id, type) DO UPDATE SET content = excluded.content
	`
	getGlobalAccountDataQuery = `
		SELECT user_id, '', type, content FROM account_data WHERE user_id = $1
	`
	getRoomAccountDataQuery = `
		SELECT user_id, room_id, type, content FROM room_account_data WHERE user_id = $1 AND room_id = $2
	`
)

type AccountDataQuery struct {
	*dbutil.QueryHelper[*AccountData]
}

func unsafeJSONString(content json.RawMessage) *string {
	if content == nil {
		return nil
	}
	str := unsafe.String(unsafe.SliceData(content), len(content))
	return &str
}

func (adq *AccountDataQuery) Put(ctx context.Context, userID id.UserID, eventType event.Type, content json.RawMessage) (*AccountData, error) {
	ad := &AccountData{
		UserID:  userID,
		Type:    eventType.Type,
		Content: content,
	}
	return ad, adq.Exec(ctx, upsertAccountDataQuery, userID, eventType.Type, unsafeJSONString(content))
}

func (adq *AccountDataQuery) PutRoom(ctx context.Context, userID id.UserID, roomID id.RoomID, eventType event.Type, content json.RawMessage) (*AccountData, error) {
	ad := &AccountData{
		UserID:  userID,
		RoomID:  roomID,
		Type:    eventType.Type,
		Content: content,
	}
	return ad, adq.Exec(ctx, upsertRoomAccountDataQuery, userID, roomID, eventType.Type, unsafeJSONString(content))
}

func (adq *AccountDataQuery) GetAllGlobal(ctx context.Context, userID id.UserID) ([]*AccountData, error) {
	return adq.QueryMany(ctx, getGlobalAccountDataQuery, userID)
}

func (adq *AccountDataQuery) GetAllRoom(ctx context.Context, userID id.UserID, roomID id.RoomID) ([]*AccountData, error) {
	return adq.QueryMany(ctx, getRoomAccountDataQuery, userID, roomID)
}

type AccountData struct {
	UserID  id.UserID       `json:"user_id"`
	RoomID  id.RoomID       `json:"room_id,omitempty"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

func (a *AccountData) Scan(row dbutil.Scannable) (*AccountData, error) {
	var roomID sql.NullString
	err := row.Scan(&a.UserID, &roomID, &a.Type, (*[]byte)(&a.Content))
	if err != nil {
		return nil, err
	}
	a.RoomID = id.RoomID(roomID.String)
	return a, nil
}
