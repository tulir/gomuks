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
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/gomuks/matrix/rooms"
)

type Relation struct {
	Type  event.RelationType
	Event *muksevt.Event
}

type MatrixContainer interface {
	Client() *mautrix.Client
	Preferences() *config.UserPreferences
	InitClient() error
	Initialized() bool

	Start()
	Stop()

	Login(user, password string) error
	Logout()

	SendPreferencesToMatrix()
	PrepareMarkdownMessage(roomID id.RoomID, msgtype event.MessageType, text, html string, relation *Relation) *muksevt.Event
	SendEvent(evt *muksevt.Event) (id.EventID, error)
	Redact(roomID id.RoomID, eventID id.EventID, reason string) error
	SendTyping(roomID id.RoomID, typing bool)
	MarkRead(roomID id.RoomID, eventID id.EventID)
	JoinRoom(roomID id.RoomID, server string) (*rooms.Room, error)
	LeaveRoom(roomID id.RoomID) error
	CreateRoom(req *mautrix.ReqCreateRoom) (*rooms.Room, error)

	FetchMembers(room *rooms.Room) error
	GetHistory(room *rooms.Room, limit int) ([]*muksevt.Event, error)
	GetEvent(room *rooms.Room, eventID id.EventID) (*muksevt.Event, error)
	GetRoom(roomID id.RoomID) *rooms.Room
	GetOrCreateRoom(roomID id.RoomID) *rooms.Room

	Download(uri id.ContentURI) ([]byte, error)
	DownloadToDisk(uri id.ContentURI, target string) (string, error)
	GetDownloadURL(uri id.ContentURI) string
	GetCachePath(uri id.ContentURI) string
}
