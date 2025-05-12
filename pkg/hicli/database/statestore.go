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
	"fmt"
	"slices"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const (
	getMembershipQuery = `
		SELECT membership FROM current_state
		WHERE room_id = $1 AND event_type = 'm.room.member' AND state_key = $2
	`
	getStateEventContentQuery = `
		SELECT event.content FROM current_state cs
		LEFT JOIN event ON event.rowid = cs.event_rowid
		WHERE cs.room_id = $1 AND cs.event_type = $2 AND cs.state_key = $3
	`
	getRoomJoinedMembersQuery = `
		SELECT state_key FROM current_state
		WHERE room_id = $1 AND event_type = 'm.room.member' AND membership = 'join'
	`
	getRoomJoinedOrInvitedMembersQuery = `
		SELECT state_key FROM current_state
		WHERE room_id = $1 AND event_type = 'm.room.member' AND membership IN ('join', 'invite')
	`
	getHasFetchedMembersQuery = `
		SELECT has_member_list FROM room WHERE room_id = $1
	`
	isRoomEncryptedQuery = `
		SELECT room.encryption_event IS NOT NULL FROM room WHERE room_id = $1
	`
	getRoomEncryptionEventQuery = `
		SELECT room.encryption_event FROM room WHERE room_id = $1
	`
	findSharedRoomsQuery = `
		SELECT room_id FROM current_state
		WHERE event_type = 'm.room.member' AND state_key = $1 AND membership = 'join'
	`
)

type ClientStateStore struct {
	*Database
}

func (c *ClientStateStore) IsInRoom(ctx context.Context, roomID id.RoomID, userID id.UserID) bool {
	return c.IsMembership(ctx, roomID, userID, event.MembershipJoin)
}

func (c *ClientStateStore) IsInvited(ctx context.Context, roomID id.RoomID, userID id.UserID) bool {
	return c.IsMembership(ctx, roomID, userID, event.MembershipInvite, event.MembershipJoin)
}

func (c *ClientStateStore) IsMembership(ctx context.Context, roomID id.RoomID, userID id.UserID, allowedMemberships ...event.Membership) bool {
	var membership event.Membership
	err := c.QueryRow(ctx, getMembershipQuery, roomID, userID).Scan(&membership)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
		membership = event.MembershipLeave
	}
	return slices.Contains(allowedMemberships, membership)
}

func (c *ClientStateStore) GetMember(ctx context.Context, roomID id.RoomID, userID id.UserID) (*event.MemberEventContent, error) {
	content, err := c.TryGetMember(ctx, roomID, userID)
	if content == nil {
		content = &event.MemberEventContent{Membership: event.MembershipLeave}
	}
	return content, err
}

func (c *ClientStateStore) TryGetMember(ctx context.Context, roomID id.RoomID, userID id.UserID) (content *event.MemberEventContent, err error) {
	err = c.QueryRow(ctx, getStateEventContentQuery, roomID, event.StateMember.Type, userID).Scan(&dbutil.JSON{Data: &content})
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (c *ClientStateStore) IsConfusableName(ctx context.Context, roomID id.RoomID, currentUser id.UserID, name string) ([]id.UserID, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ClientStateStore) GetPowerLevels(ctx context.Context, roomID id.RoomID) (content *event.PowerLevelsEventContent, err error) {
	err = c.QueryRow(ctx, getStateEventContentQuery, roomID, event.StatePowerLevels.Type, "").Scan(&dbutil.JSON{Data: &content})
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (c *ClientStateStore) GetRoomJoinedMembers(ctx context.Context, roomID id.RoomID) ([]id.UserID, error) {
	rows, err := c.Query(ctx, getRoomJoinedMembersQuery, roomID)
	return dbutil.NewRowIterWithError(rows, dbutil.ScanSingleColumn[id.UserID], err).AsList()
}

func (c *ClientStateStore) GetRoomJoinedOrInvitedMembers(ctx context.Context, roomID id.RoomID) ([]id.UserID, error) {
	rows, err := c.Query(ctx, getRoomJoinedOrInvitedMembersQuery, roomID)
	return dbutil.NewRowIterWithError(rows, dbutil.ScanSingleColumn[id.UserID], err).AsList()
}

func (c *ClientStateStore) HasFetchedMembers(ctx context.Context, roomID id.RoomID) (hasFetched bool, err error) {
	//err = c.QueryRow(ctx, getHasFetchedMembersQuery, roomID).Scan(&hasFetched)
	//if errors.Is(err, sql.ErrNoRows) {
	//	err = nil
	//}
	//return
	return false, fmt.Errorf("not implemented")
}

func (c *ClientStateStore) MarkMembersFetched(ctx context.Context, roomID id.RoomID) error {
	return fmt.Errorf("not implemented")
}

func (c *ClientStateStore) GetAllMembers(ctx context.Context, roomID id.RoomID) (map[id.UserID]*event.MemberEventContent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ClientStateStore) IsEncrypted(ctx context.Context, roomID id.RoomID) (isEncrypted bool, err error) {
	err = c.QueryRow(ctx, isRoomEncryptedQuery, roomID).Scan(&isEncrypted)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (c *ClientStateStore) GetEncryptionEvent(ctx context.Context, roomID id.RoomID) (content *event.EncryptionEventContent, err error) {
	err = c.QueryRow(ctx, getRoomEncryptionEventQuery, roomID).
		Scan(&dbutil.JSON{Data: &content})
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (c *ClientStateStore) FindSharedRooms(ctx context.Context, userID id.UserID) ([]id.RoomID, error) {
	// TODO for multiuser support, this might need to filter by the local user's membership
	rows, err := c.Query(ctx, findSharedRoomsQuery, userID)
	return dbutil.NewRowIterWithError(rows, dbutil.ScanSingleColumn[id.RoomID], err).AsList()
}

// Update methods are all intentionally no-ops as the state store wants to have the full event

func (c *ClientStateStore) SetMembership(ctx context.Context, roomID id.RoomID, userID id.UserID, membership event.Membership) error {
	return nil
}

func (c *ClientStateStore) SetMember(ctx context.Context, roomID id.RoomID, userID id.UserID, member *event.MemberEventContent) error {
	return nil
}

func (c *ClientStateStore) ClearCachedMembers(ctx context.Context, roomID id.RoomID, memberships ...event.Membership) error {
	return nil
}

func (c *ClientStateStore) SetPowerLevels(ctx context.Context, roomID id.RoomID, levels *event.PowerLevelsEventContent) error {
	return nil
}

func (c *ClientStateStore) SetEncryptionEvent(ctx context.Context, roomID id.RoomID, content *event.EncryptionEventContent) error {
	return nil
}

func (c *ClientStateStore) UpdateState(ctx context.Context, evt *event.Event) {}

func (c *ClientStateStore) ReplaceCachedMembers(ctx context.Context, roomID id.RoomID, evts []*event.Event, onlyMemberships ...event.Membership) error {
	return nil
}
