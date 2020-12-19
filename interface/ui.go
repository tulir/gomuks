// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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

	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"
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

type SyncingModal interface {
	SetIndeterminate()
	SetMessage(string)
	SetSteps(int)
	Step()
	Close()
}

type MainView interface {
	GetRoom(roomID id.RoomID) RoomView
	AddRoom(room *rooms.Room)
	RemoveRoom(room *rooms.Room)
	SetRooms(rooms *rooms.RoomCache)
	Bump(room *rooms.Room)

	UpdateTags(room *rooms.Room)

	SetTyping(roomID id.RoomID, users []id.UserID)
	OpenSyncingModal() SyncingModal

	NotifyMessage(room *rooms.Room, message Message, should pushrules.PushActionArrayShould)
}

type RoomView interface {
	MxRoom() *rooms.Room

	SetCompletions(completions []string)
	SetTyping(users []id.UserID)
	SetEncrypted()
	UpdateUserList()

	AddEvent(evt *muksevt.Event) Message
	AddRedaction(evt *muksevt.Event)
	AddEdit(evt *muksevt.Event)
	AddReaction(evt *muksevt.Event, key string)
	GetEvent(eventID id.EventID) Message
	AddServiceMessage(message string)
}

type Message interface {
	ID() id.EventID
	Time() time.Time
	NotificationSenderName() string
	NotificationContent() string

	SetIsHighlight(highlight bool)
	SetID(id id.EventID)
}
