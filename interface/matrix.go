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
	"maunium.net/go/mautrix"

	"maunium.net/go/gomuks/matrix/rooms"
)

type MatrixContainer interface {
	Client() *mautrix.Client
	InitClient() error
	Initialized() bool

	Start()
	Stop()

	Login(user, password string) error
	Logout()

	SendPreferencesToMatrix()
	PrepareMarkdownMessage(roomID string, msgtype mautrix.MessageType, message string) *mautrix.Event
	SendEvent(event *mautrix.Event) (string, error)
	SendTyping(roomID string, typing bool)
	MarkRead(roomID, eventID string)
	JoinRoom(roomID, server string) (*rooms.Room, error)
	LeaveRoom(roomID string) error
	CreateRoom(req *mautrix.ReqCreateRoom) (*rooms.Room, error)

	GetHistory(room *rooms.Room, limit int) ([]*mautrix.Event, error)
	GetEvent(room *rooms.Room, eventID string) (*mautrix.Event, error)
	GetRoom(roomID string) *rooms.Room
	GetOrCreateRoom(roomID string) *rooms.Room

	Download(mxcURL string) ([]byte, string, string, error)
	GetDownloadURL(homeserver, fileID string) string
	GetCachePath(homeserver, fileID string) string
}
