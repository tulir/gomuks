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
	"maunium.net/go/mautrix/crypto/attachment"
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

type UploadedMediaInfo struct {
	*mautrix.RespMediaUpload
	EncryptionInfo *attachment.EncryptedFile
	MsgType        event.MessageType
	Name           string
	Info           *event.FileInfo
}

type MatrixContainer interface {
	Client() *mautrix.Client
	Preferences() *config.UserPreferences
	InitClient(isStartup bool) error
	Initialized() bool

	Start()
	Stop()

	Login(user, password string) error
	Logout()
	UIAFallback(authType mautrix.AuthType, sessionID string) error

	SendPreferencesToMatrix()
	PrepareMarkdownMessage(roomID id.RoomID, msgtype event.MessageType, text, html string, relation *Relation) *muksevt.Event
	PrepareMediaMessage(room *rooms.Room, path string, relation *Relation) (*muksevt.Event, error)
	SendEvent(evt *muksevt.Event) (id.EventID, error)
	Redact(roomID id.RoomID, eventID id.EventID, reason string) error
	SendTyping(roomID id.RoomID, typing bool)
	MarkRead(roomID id.RoomID, eventID id.EventID)
	JoinRoom(roomID id.RoomID, server string) (*rooms.Room, error)
	LeaveRoom(roomID id.RoomID) error
	CreateRoom(req *mautrix.ReqCreateRoom) (*rooms.Room, error)

	FetchMembers(room *rooms.Room) error
	GetHistory(room *rooms.Room, limit int, dbPointer uint64) ([]*muksevt.Event, uint64, error)
	GetEvent(room *rooms.Room, eventID id.EventID) (*muksevt.Event, error)
	GetRoom(roomID id.RoomID) *rooms.Room
	GetOrCreateRoom(roomID id.RoomID) *rooms.Room

	UploadMedia(path string, encrypt bool) (*UploadedMediaInfo, error)
	Download(uri id.ContentURI, file *attachment.EncryptedFile) ([]byte, error)
	DownloadToDisk(uri id.ContentURI, file *attachment.EncryptedFile, target string) (string, error)
	GetDownloadURL(uri id.ContentURI) string
	GetCachePath(uri id.ContentURI) string

	Crypto() Crypto
}

type Crypto interface {
	Load() error
	FlushStore() error
	ProcessSyncResponse(resp *mautrix.RespSync, since string) bool
	ProcessInRoomVerification(evt *event.Event) error
	HandleMemberEvent(*event.Event)
	DecryptMegolmEvent(*event.Event) (*event.Event, error)
	EncryptMegolmEvent(id.RoomID, event.Type, interface{}) (*event.EncryptedEventContent, error)
	ShareGroupSession(id.RoomID, []id.UserID) error
	Fingerprint() string
}
