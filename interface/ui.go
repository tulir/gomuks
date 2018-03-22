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
	"maunium.net/go/gomuks/ui/types"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tview"
)

type View string

// Allowed views in GomuksUI
const (
	ViewLogin View = "login"
	ViewMain  View = "main"
)

type GomuksUI interface {
	Render()
	SetView(name View)
	InitViews() tview.Primitive
	MainView() MainView
	LoginView() LoginView
}

type MainView interface {
	GetRoom(roomID string) *widget.RoomView
	HasRoom(roomID string) bool
	AddRoom(roomID string)
	RemoveRoom(roomID string)
	SetRooms(roomIDs []string)
	SaveAllHistory()

	SetTyping(roomID string, users []string)
	AddServiceMessage(roomID *widget.RoomView, message string)
	ProcessMessageEvent(roomView *widget.RoomView, evt *gomatrix.Event) *types.Message
	ProcessMembershipEvent(roomView *widget.RoomView, evt *gomatrix.Event) *types.Message
}

type LoginView interface {
}
