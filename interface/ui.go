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

package ifc

import (
	"time"

	"maunium.net/go/gomuks/matrix/event"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
)

type UIProvider func(gmx Gomuks) GomuksUI

type GomuksUI interface {
	Render()
	HandleNewPreferences()
	OnLogin()
	OnLogout()
	MainView() MainView

	Init()
	Start() error
	Stop()
	Finish()
}

type MainView interface {
	GetRoom(roomID string) RoomView
	AddRoom(room *rooms.Room)
	RemoveRoom(room *rooms.Room)
	SetRooms(rooms *rooms.RoomCache)
	Bump(room *rooms.Room)

	UpdateTags(room *rooms.Room)

	SetTyping(roomID string, users []string)

	NotifyMessage(room *rooms.Room, message Message, should pushrules.PushActionArrayShould)
}

type RoomView interface {
	MxRoom() *rooms.Room

	SetCompletions(completions []string)
	SetTyping(users []string)
	UpdateUserList()

	AddEvent(evt *event.Event) Message
	AddRedaction(evt *event.Event)
	AddEdit(evt *event.Event)
	AddReaction(evt *event.Event, key string)
	GetEvent(eventID string) Message
	AddServiceMessage(message string)
}

type Message interface {
	ID() string
	Time() time.Time
	NotificationSenderName() string
	NotificationContent() string

	SetIsHighlight(highlight bool)
	SetID(id string)
}
