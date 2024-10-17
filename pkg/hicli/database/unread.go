// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"

	"maunium.net/go/mautrix/id"
)

const (
	// TODO find out if this needs to be wrapped in another query that limits the number of events it evaluates
	//      (or maybe the timeline store just shouldn't be allowed to grow that big?)
	calculateUnreadsQuery = `
		SELECT
			COALESCE(SUM(CASE WHEN unread_type & 0100 THEN 1 ELSE 0 END), 0) AS highlights,
			COALESCE(SUM(CASE WHEN unread_type & 0010 THEN 1 ELSE 0 END), 0) AS notifications,
			COALESCE(SUM(CASE WHEN unread_type & 0001 THEN 1 ELSE 0 END), 0) AS messages
		FROM timeline
		JOIN event ON event.rowid = timeline.event_rowid
		WHERE timeline.room_id = $1 AND timeline.rowid > (
			SELECT MAX(rowid)
			FROM timeline
			WHERE room_id = $1 AND event_rowid IN (
				SELECT event.rowid
				FROM receipt
				JOIN event ON receipt.event_id=event.event_id
				WHERE receipt.room_id = $1 AND receipt.user_id = $2
			)
		)
	`
)

func (rq *RoomQuery) CalculateUnreads(ctx context.Context, roomID id.RoomID, userID id.UserID) (uc UnreadCounts, err error) {
	err = rq.GetDB().QueryRow(ctx, calculateUnreadsQuery, roomID, userID).
		Scan(&uc.UnreadHighlights, &uc.UnreadNotifications, &uc.UnreadMessages)
	return
}

type UnreadType int

func (ut UnreadType) Is(flag UnreadType) bool {
	return ut&flag != 0
}

const (
	UnreadTypeNone      UnreadType = 0b0000
	UnreadTypeNormal    UnreadType = 0b0001
	UnreadTypeNotify    UnreadType = 0b0010
	UnreadTypeHighlight UnreadType = 0b0100
	UnreadTypeSound     UnreadType = 0b1000
)

type UnreadCounts struct {
	UnreadHighlights    int `json:"unread_highlights"`
	UnreadNotifications int `json:"unread_notifications"`
	UnreadMessages      int `json:"unread_messages"`
}

func (uc *UnreadCounts) IsZero() bool {
	return uc.UnreadHighlights == 0 && uc.UnreadNotifications == 0 && uc.UnreadMessages == 0
}

func (uc *UnreadCounts) Add(other UnreadCounts) {
	uc.UnreadHighlights += other.UnreadHighlights
	uc.UnreadNotifications += other.UnreadNotifications
	uc.UnreadMessages += other.UnreadMessages
}

func (uc *UnreadCounts) AddOne(ut UnreadType) {
	if ut.Is(UnreadTypeNormal) {
		uc.UnreadMessages++
	}
	if ut.Is(UnreadTypeNotify) {
		uc.UnreadNotifications++
	}
	if ut.Is(UnreadTypeHighlight) {
		uc.UnreadHighlights++
	}
}
