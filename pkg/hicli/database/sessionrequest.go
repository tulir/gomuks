// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

const (
	putSessionRequestQueueEntry = `
		INSERT INTO session_request (room_id, session_id, sender, min_index, backup_checked, request_sent)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (session_id) DO UPDATE
			SET min_index = MIN(excluded.min_index, session_request.min_index),
			    backup_checked = excluded.backup_checked OR session_request.backup_checked,
			    request_sent = excluded.request_sent OR session_request.request_sent
	`
	removeSessionRequestQuery = `
		DELETE FROM session_request WHERE session_id = $1 AND min_index >= $2
	`
	getNextSessionsToRequestQuery = `
		SELECT room_id, session_id, sender, min_index, backup_checked, request_sent
		FROM session_request
		WHERE request_sent = false OR backup_checked = false
		ORDER BY backup_checked, rowid
		LIMIT $1
	`
)

type SessionRequestQuery struct {
	*dbutil.QueryHelper[*SessionRequest]
}

func (srq *SessionRequestQuery) Next(ctx context.Context, count int) ([]*SessionRequest, error) {
	return srq.QueryMany(ctx, getNextSessionsToRequestQuery, count)
}

func (srq *SessionRequestQuery) Remove(ctx context.Context, sessionID id.SessionID, minIndex uint32) error {
	return srq.Exec(ctx, removeSessionRequestQuery, sessionID, minIndex)
}

func (srq *SessionRequestQuery) Put(ctx context.Context, sr *SessionRequest) error {
	return srq.Exec(ctx, putSessionRequestQueueEntry, sr.sqlVariables()...)
}

type SessionRequest struct {
	RoomID        id.RoomID
	SessionID     id.SessionID
	Sender        id.UserID
	MinIndex      uint32
	BackupChecked bool
	RequestSent   bool
}

func (s *SessionRequest) Scan(row dbutil.Scannable) (*SessionRequest, error) {
	return dbutil.ValueOrErr(s, row.Scan(&s.RoomID, &s.SessionID, &s.Sender, &s.MinIndex, &s.BackupChecked, &s.RequestSent))
}

func (s *SessionRequest) sqlVariables() []any {
	return []any{s.RoomID, s.SessionID, s.Sender, s.MinIndex, s.BackupChecked, s.RequestSent}
}
