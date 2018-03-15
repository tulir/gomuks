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

package main

import (
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"maunium.net/go/gomatrix"
	"maunium.net/go/tview"
)

type MainView struct {
	*tview.Grid

	roomList         *tview.List
	roomView         *tview.Pages
	rooms            map[string]*RoomView
	input            *tview.InputField
	currentRoomIndex int
	roomIDs          []string

	matrix *MatrixContainer
	debug  DebugPrinter
	gmx    Gomuks
	config *Config
	parent *GomuksUI
}

func (view *MainView) addItem(p tview.Primitive, x, y, w, h int) {
	view.Grid.AddItem(p, x, y, w, h, 0, 0, false)
}

func (ui *GomuksUI) NewMainView() tview.Primitive {
	mainView := &MainView{
		Grid:     tview.NewGrid(),
		roomList: tview.NewList(),
		roomView: tview.NewPages(),
		rooms:    make(map[string]*RoomView),
		input:    tview.NewInputField(),

		matrix: ui.matrix,
		debug:  ui.debug,
		gmx:    ui.gmx,
		config: ui.config,
		parent: ui,
	}

	mainView.SetColumns(30, 1, 0).SetRows(0, 1)

	mainView.roomList.
		ShowSecondaryText(false).
		SetBorderPadding(0, 0, 1, 0)

	mainView.input.
		SetDoneFunc(mainView.InputDone).
		SetChangedFunc(mainView.InputChanged).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetInputCapture(mainView.InputCapture)

	mainView.addItem(mainView.roomList, 0, 0, 2, 1)
	mainView.addItem(NewBorder(), 0, 1, 2, 1)
	mainView.addItem(mainView.roomView, 0, 2, 1, 1)
	mainView.AddItem(mainView.input, 1, 2, 1, 1, 0, 0, true)

	ui.mainView = mainView

	return mainView
}

func (view *MainView) InputChanged(text string) {
	if len(text) == 0 {
		view.matrix.SendTyping(view.CurrentRoomID(), false)
	} else if text[0] != '/' {
		view.matrix.SendTyping(view.CurrentRoomID(), true)
	}
}

func (view *MainView) InputDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		room, text := view.CurrentRoomID(), view.input.GetText()
		if len(text) == 0 {
			return
		} else if text[0] == '/' {
			args := strings.SplitN(text, " ", 2)
			command := strings.ToLower(args[0])
			args = args[1:]
			view.HandleCommand(room, command, args)
		} else {
			view.matrix.SendMessage(room, text)
		}
		view.input.SetText("")
	}
}

func (view *MainView) HandleCommand(room, command string, args []string) {
	view.debug.Print("Handling command", command, args)
	switch command {
	case "/quit":
		view.gmx.Stop()
	case "/clearcache":
		view.config.Session.Rooms = make(map[string]*gomatrix.Room)
		view.config.Session.NextBatch = ""
		view.config.Session.FilterID = ""
		view.config.Session.Save()
		view.gmx.Stop()
	case "/part":
	case "/leave":
		view.matrix.client.LeaveRoom(room)
	case "/join":
		if len(args) == 0 {
			view.AddMessage(room, "*", "Usage: /join <room>", time.Now())
			break
		}
		mxid := args[0]
		server := mxid[strings.Index(mxid, ":")+1:]
		view.matrix.client.JoinRoom(mxid, server, nil)
	}
}

func (view *MainView) InputCapture(key *tcell.EventKey) *tcell.EventKey {
	if key.Modifiers() == tcell.ModCtrl {
		if key.Key() == tcell.KeyDown {
			view.SwitchRoom(view.currentRoomIndex + 1)
			view.roomList.SetCurrentItem(view.currentRoomIndex)
		} else if key.Key() == tcell.KeyUp {
			view.SwitchRoom(view.currentRoomIndex - 1)
			view.roomList.SetCurrentItem(view.currentRoomIndex)
		} else {
			return key
		}
	} else if key.Key() == tcell.KeyPgUp || key.Key() == tcell.KeyPgDn {
		view.rooms[view.CurrentRoomID()].InputHandler()(key, nil)
	} else {
		return key
	}
	return nil
}

func (view *MainView) CurrentRoomID() string {
	if len(view.roomIDs) == 0 {
		return ""
	}
	return view.roomIDs[view.currentRoomIndex]
}

func (view *MainView) SwitchRoom(roomIndex int) {
	if roomIndex < 0 {
		roomIndex = len(view.roomIDs) - 1
	}
	view.currentRoomIndex = roomIndex % len(view.roomIDs)
	view.roomView.SwitchToPage(view.CurrentRoomID())
	view.parent.Render()
}

func (view *MainView) SetRoomList(rooms []string) {
	view.roomIDs = rooms
	view.roomList.Clear()
	for index, room := range rooms {
		localRoomIndex := index

		view.matrix.UpdateState(room)
		roomStore := view.matrix.config.Session.LoadRoom(room)

		name := room
		topic := ""
		var users []string
		if roomStore != nil {
			name = roomStore.GetTitle()
			topic = roomStore.GetTopic()
			users = roomStore.GetMembers()
		}

		view.roomList.AddItem(name, "", 0, func() {
			view.SwitchRoom(localRoomIndex)
		})
		if !view.roomView.HasPage(room) {
			roomView := NewRoomView(topic)
			roomView.SetUsers(users)
			view.rooms[room] = roomView
			view.roomView.AddPage(room, roomView, true, false)
		}
	}
	view.SwitchRoom(0)
}

func (view *MainView) SetTyping(room string, users []string) {
	roomView, ok := view.rooms[room]
	if ok {
		roomView.SetTyping(users)
		view.parent.Render()
	}
}

func (view *MainView) AddMessage(room, sender, message string, timestamp time.Time) {
	roomView, ok := view.rooms[room]
	if ok {
		roomView.AddMessage(sender, message, timestamp)
		view.parent.Render()
	}
}
