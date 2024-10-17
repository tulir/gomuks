// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type pushRoom struct {
	ctx    context.Context
	roomID id.RoomID
	h      *HiClient
	ll     *mautrix.LazyLoadSummary
}

func (p *pushRoom) GetOwnDisplayname() string {
	// TODO implement
	return ""
}

func (p *pushRoom) GetMemberCount() int {
	if p.ll == nil {
		room, err := p.h.DB.Room.Get(p.ctx, p.roomID)
		if err != nil {
			zerolog.Ctx(p.ctx).Err(err).
				Stringer("room_id", p.roomID).
				Msg("Failed to get room by ID in push rule evaluator")
		} else if room != nil {
			p.ll = room.LazyLoadSummary
		}
	}
	if p.ll != nil && p.ll.JoinedMemberCount != nil {
		return *p.ll.JoinedMemberCount
	}
	// TODO query db?
	return 0
}

func (p *pushRoom) GetEvent(id id.EventID) *event.Event {
	evt, err := p.h.DB.Event.GetByID(p.ctx, id)
	if err != nil {
		zerolog.Ctx(p.ctx).Err(err).
			Stringer("event_id", id).
			Msg("Failed to get event by ID in push rule evaluator")
	}
	return evt.AsRawMautrix()
}

var _ pushrules.EventfulRoom = (*pushRoom)(nil)

func (h *HiClient) evaluatePushRules(ctx context.Context, llSummary *mautrix.LazyLoadSummary, baseType database.UnreadType, evt *event.Event) database.UnreadType {
	should := h.PushRules.Load().GetMatchingRule(&pushRoom{
		ctx:    ctx,
		roomID: evt.RoomID,
		h:      h,
		ll:     llSummary,
	}, evt).GetActions().Should()
	if should.Notify {
		baseType |= database.UnreadTypeNotify
	}
	if should.Highlight {
		baseType |= database.UnreadTypeHighlight
	}
	if should.PlaySound {
		baseType |= database.UnreadTypeSound
	}
	return baseType
}
