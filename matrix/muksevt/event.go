// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package muksevt

import (
	"maunium.net/go/mautrix/event"
)

type Event struct {
	*event.Event
	Gomuks GomuksContent `json:"-"`
}

func (evt *Event) SomewhatDangerousCopy() *Event {
	base := *evt.Event
	return &Event{
		Event: &base,
		Gomuks: evt.Gomuks,
	}
}

func Wrap(event *event.Event) *Event {
	return &Event{Event: event}
}

type OutgoingState int

const (
	StateDefault OutgoingState = iota
	StateLocalEcho
	StateSendFail
)

type GomuksContent struct {
	OutgoingState OutgoingState
	Edits         []*Event
}
