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
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/matrix/rooms"
)

type MatrixContainer interface {
	Client() *gomatrix.Client
	InitClient() error
	Initialized() bool

	Start()
	Stop()

	Login(user, password string) error
	Logout()

	SendPreferencesToMatrix()
	SendMessage(roomID, msgtype, message string) (string, error)
	SendMarkdownMessage(roomID, msgtype, message string) (string, error)
	SendTyping(roomID string, typing bool)
	MarkRead(roomID, eventID string)
	JoinRoom(roomID, server string) (*rooms.Room, error)
	LeaveRoom(roomID string) error

	GetHistory(roomID, prevBatch string, limit int) ([]gomatrix.Event, string, error)
	GetRoom(roomID string) *rooms.Room

	Download(mxcURL string) ([]byte, string, string, error)
	GetDownloadURL(homeserver, fileID string) string
	GetCachePath(homeserver, fileID string) string
}
