// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"go.mau.fi/util/dbutil"

	"go.mau.fi/gomuks/pkg/hicli/database/upgrades"
)

type Database struct {
	*dbutil.Database

	Account          *AccountQuery
	AccountData      *AccountDataQuery
	Room             *RoomQuery
	InvitedRoom      *InvitedRoomQuery
	Event            *EventQuery
	CurrentState     *CurrentStateQuery
	Timeline         *TimelineQuery
	SessionRequest   *SessionRequestQuery
	Receipt          *ReceiptQuery
	Media            *MediaQuery
	SpaceEdge        *SpaceEdgeQuery
	PushRegistration *PushRegistrationQuery
}

func New(rawDB *dbutil.Database) *Database {
	rawDB.UpgradeTable = upgrades.Table
	eventQH := dbutil.MakeQueryHelper(rawDB, newEvent)
	return &Database{
		Database: rawDB,

		Account:          &AccountQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newAccount)},
		AccountData:      &AccountDataQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newAccountData)},
		Room:             &RoomQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newRoom)},
		InvitedRoom:      &InvitedRoomQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newInvitedRoom)},
		Event:            &EventQuery{QueryHelper: eventQH},
		CurrentState:     &CurrentStateQuery{QueryHelper: eventQH},
		Timeline:         &TimelineQuery{QueryHelper: eventQH},
		SessionRequest:   &SessionRequestQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newSessionRequest)},
		Receipt:          &ReceiptQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newReceipt)},
		Media:            &MediaQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newMedia)},
		SpaceEdge:        &SpaceEdgeQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newSpaceEdge)},
		PushRegistration: &PushRegistrationQuery{QueryHelper: dbutil.MakeQueryHelper(rawDB, newPushRegistration)},
	}
}

func newSessionRequest(_ *dbutil.QueryHelper[*SessionRequest]) *SessionRequest {
	return &SessionRequest{}
}

func newEvent(_ *dbutil.QueryHelper[*Event]) *Event {
	return &Event{}
}

func newRoom(_ *dbutil.QueryHelper[*Room]) *Room {
	return &Room{}
}

func newInvitedRoom(_ *dbutil.QueryHelper[*InvitedRoom]) *InvitedRoom {
	return &InvitedRoom{}
}

func newReceipt(_ *dbutil.QueryHelper[*Receipt]) *Receipt {
	return &Receipt{}
}

func newMedia(_ *dbutil.QueryHelper[*Media]) *Media {
	return &Media{}
}

func newAccountData(_ *dbutil.QueryHelper[*AccountData]) *AccountData {
	return &AccountData{}
}

func newAccount(_ *dbutil.QueryHelper[*Account]) *Account {
	return &Account{}
}

func newSpaceEdge(_ *dbutil.QueryHelper[*SpaceEdge]) *SpaceEdge {
	return &SpaceEdge{}
}

func newPushRegistration(_ *dbutil.QueryHelper[*PushRegistration]) *PushRegistration {
	return &PushRegistration{}
}
