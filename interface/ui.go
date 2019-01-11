// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ifc

import (
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/tcell"
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
	SetRooms(rooms map[string]*rooms.Room)
	SaveAllHistory()

	UpdateTags(room *rooms.Room)

	SetTyping(roomID string, users []string)
	ParseEvent(roomView RoomView, evt *mautrix.Event) Message

	NotifyMessage(room *rooms.Room, message Message, should pushrules.PushActionArrayShould)
	InitialSyncDone()
}

type MessageDirection int

const (
	AppendMessage MessageDirection = iota
	PrependMessage
	IgnoreMessage
)

type RoomView interface {
	MxRoom() *rooms.Room
	SaveHistory(dir string) error
	LoadHistory(matrix MatrixContainer, dir string) (int, error)

	SetCompletions(completions []string)
	SetTyping(users []string)
	UpdateUserList()

	NewTempMessage(msgtype mautrix.MessageType, text string) Message
	AddMessage(message Message, direction MessageDirection)
	AddServiceMessage(message string)
}

type MessageMeta interface {
	Sender() string
	SenderColor() tcell.Color
	TextColor() tcell.Color
	TimestampColor() tcell.Color
	Timestamp() time.Time
	FormatTime() string
	FormatDate() string
}

// MessageState is an enum to specify if a Message is being sent, failed to send or was successfully sent.
type MessageState int

// Allowed MessageStates.
const (
	MessageStateSending MessageState = iota
	MessageStateDefault
	MessageStateFailed
)

type Message interface {
	MessageMeta

	SetIsHighlight(isHighlight bool)
	IsHighlight() bool

	SetIsService(isService bool)
	IsService() bool

	SetID(id string)
	ID() string

	SetType(msgtype mautrix.MessageType)
	Type() mautrix.MessageType

	NotificationContent() string

	SetState(state MessageState)
	State() MessageState

	SenderID() string
}
