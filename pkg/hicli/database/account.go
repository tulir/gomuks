// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"database/sql"
	"errors"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

const (
	getAccountQuery    = `SELECT user_id, device_id, access_token, homeserver_url, next_batch FROM account WHERE user_id = $1`
	putNextBatchQuery  = `UPDATE account SET next_batch = $1 WHERE user_id = $2`
	upsertAccountQuery = `
		INSERT INTO account (user_id, device_id, access_token, homeserver_url, next_batch)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_id)
			DO UPDATE SET device_id = excluded.device_id,
			              access_token = excluded.access_token,
			              homeserver_url = excluded.homeserver_url,
			              next_batch = excluded.next_batch
	`
)

type AccountQuery struct {
	*dbutil.QueryHelper[*Account]
}

func (aq *AccountQuery) GetFirstUserID(ctx context.Context) (userID id.UserID, err error) {
	var exists bool
	if exists, err = aq.GetDB().TableExists(ctx, "account"); err != nil || !exists {
		return
	}
	err = aq.GetDB().QueryRow(ctx, `SELECT user_id FROM account LIMIT 1`).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (aq *AccountQuery) Get(ctx context.Context, userID id.UserID) (*Account, error) {
	return aq.QueryOne(ctx, getAccountQuery, userID)
}

func (aq *AccountQuery) PutNextBatch(ctx context.Context, userID id.UserID, nextBatch string) error {
	return aq.Exec(ctx, putNextBatchQuery, nextBatch, userID)
}

func (aq *AccountQuery) Put(ctx context.Context, account *Account) error {
	return aq.Exec(ctx, upsertAccountQuery, account.sqlVariables()...)
}

type Account struct {
	UserID        id.UserID
	DeviceID      id.DeviceID
	AccessToken   string
	HomeserverURL string
	NextBatch     string
}

func (a *Account) Scan(row dbutil.Scannable) (*Account, error) {
	return dbutil.ValueOrErr(a, row.Scan(&a.UserID, &a.DeviceID, &a.AccessToken, &a.HomeserverURL, &a.NextBatch))
}

func (a *Account) sqlVariables() []any {
	return []any{a.UserID, a.DeviceID, a.AccessToken, a.HomeserverURL, a.NextBatch}
}
