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

package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"maunium.net/go/gomatrix"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/debug"
	"maunium.net/go/gomuks/ui/types"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tview"
)

type MainView struct {
	*tview.Flex

	roomList         *tview.List
	roomView         *tview.Pages
	rooms            map[string]*widget.RoomView
	currentRoomIndex int
	roomIDs          []string

	matrix ifc.MatrixContainer
	gmx    ifc.Gomuks
	config *config.Config
	parent *GomuksUI
}

func (ui *GomuksUI) NewMainView() tview.Primitive {
	mainView := &MainView{
		Flex:     tview.NewFlex(),
		roomList: tview.NewList(),
		roomView: tview.NewPages(),
		rooms:    make(map[string]*widget.RoomView),

		matrix: ui.gmx.MatrixContainer(),
		gmx:    ui.gmx,
		config: ui.gmx.Config(),
		parent: ui,
	}

	mainView.roomList.
		ShowSecondaryText(false).
		SetSelectedBackgroundColor(tcell.ColorDarkGreen).
		SetSelectedTextColor(tcell.ColorWhite).
		SetBorderPadding(0, 0, 1, 0)

	mainView.SetDirection(tview.FlexColumn)
	mainView.AddItem(mainView.roomList, 25, 0, false)
	mainView.AddItem(widget.NewBorder(), 1, 0, false)
	mainView.AddItem(mainView.roomView, 0, 1, true)

	ui.mainView = mainView

	return mainView
}

func (view *MainView) InputChanged(roomView *widget.RoomView, text string) {
	if len(text) == 0 {
		go view.matrix.SendTyping(roomView.Room.ID, false)
	} else if text[0] != '/' {
		go view.matrix.SendTyping(roomView.Room.ID, true)
	}
}

func findWordToTabComplete(text string) string {
	output := ""
	runes := []rune(text)
	for i := len(runes) - 1; i >= 0; i-- {
		if unicode.IsSpace(runes[i]) {
			break
		}
		output = string(runes[i]) + output
	}
	return output
}

func (view *MainView) InputTabComplete(roomView *widget.RoomView, text string, cursorOffset int) string {
	str := runewidth.Truncate(text, cursorOffset, "")
	word := findWordToTabComplete(str)
	userCompletions := roomView.AutocompleteUser(word)
	if len(userCompletions) == 1 {
		startIndex := len(str) - len(word)
		completion := userCompletions[0]
		if startIndex == 0 {
			completion = completion + ": "
		}
		text = str[0:startIndex] + completion + text[len(str):]
	} else if len(userCompletions) > 1 && len(userCompletions) < 6 {
		roomView.SetStatus(fmt.Sprintf("Completions: %s", strings.Join(userCompletions, ", ")))
	}
	return text
}

func (view *MainView) InputSubmit(roomView *widget.RoomView, text string) {
	if len(text) == 0 {
		return
	} else if text[0] == '/' {
		args := strings.SplitN(text, " ", 2)
		command := strings.ToLower(args[0])
		args = args[1:]
		go view.HandleCommand(roomView.Room.ID, command, args)
	} else {
		view.SendMessage(roomView.Room.ID, text)
	}
	roomView.SetInputText("")
}

func (view *MainView) SendMessage(room, text string) {
	now := time.Now()
	roomView := view.GetRoom(room)
	tempMessage := roomView.NewMessage(
		strconv.FormatInt(now.UnixNano(), 10),
		"Sending...", text, now)
	tempMessage.TimestampColor = tcell.ColorGray
	tempMessage.TextColor = tcell.ColorGray
	tempMessage.SenderColor = tcell.ColorGray
	roomView.AddMessage(tempMessage, widget.AppendMessage)
	go func() {
		defer view.gmx.Recover()
		eventID, err := view.matrix.SendMessage(room, text)
		if err != nil {
			tempMessage.TextColor = tcell.ColorRed
			tempMessage.TimestampColor = tcell.ColorRed
			tempMessage.SenderColor = tcell.ColorRed
			tempMessage.Sender = "Error"
			roomView.SetStatus(fmt.Sprintf("Failed to send message: %s", err))
		} else {
			roomView.MessageView().UpdateMessageID(tempMessage, eventID)
		}
	}()
}

func (view *MainView) HandleCommand(room, command string, args []string) {
	defer view.gmx.Recover()
	debug.Print("Handling command", command, args)
	switch command {
	case "/quit":
		view.gmx.Stop()
	case "/clearcache":
		view.config.Session.Clear()
		view.gmx.Stop()
	case "/panic":
		panic("This is a test panic.")
	case "/part":
		fallthrough
	case "/leave":
		debug.Print(view.matrix.LeaveRoom(room))
	case "/join":
		if len(args) == 0 {
			view.AddServiceMessage(room, "Usage: /join <room>")
			break
		}
		debug.Print(view.matrix.JoinRoom(args[0]))
	default:
		view.AddServiceMessage(room, "Unknown command.")
	}
}

func (view *MainView) InputKeyHandler(roomView *widget.RoomView, key *tcell.EventKey) *tcell.EventKey {
	k := key.Key()
	if key.Modifiers() == tcell.ModCtrl || key.Modifiers() == tcell.ModAlt {
		if k == tcell.KeyDown {
			view.SwitchRoom(view.currentRoomIndex + 1)
			view.roomList.SetCurrentItem(view.currentRoomIndex)
		} else if k == tcell.KeyUp {
			view.SwitchRoom(view.currentRoomIndex - 1)
			view.roomList.SetCurrentItem(view.currentRoomIndex)
		} else {
			return key
		}
	} else if k == tcell.KeyPgUp || k == tcell.KeyPgDn || k == tcell.KeyUp || k == tcell.KeyDown {
		msgView := roomView.MessageView()
		if k == tcell.KeyPgUp || k == tcell.KeyUp {
			if msgView.IsAtTop() {
				go view.LoadMoreHistory(roomView.Room.ID)
			} else {
				msgView.MoveUp(k == tcell.KeyPgUp)
			}
		} else {
			msgView.MoveDown(k == tcell.KeyPgDn)
		}
		view.parent.Render()
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
	if len(view.roomIDs) == 0 {
		return
	}
	view.currentRoomIndex = roomIndex % len(view.roomIDs)
	view.roomView.SwitchToPage(view.CurrentRoomID())
	view.roomList.SetCurrentItem(roomIndex)
	view.gmx.App().SetFocus(view)
	view.parent.Render()
}

func (view *MainView) Focus(delegate func(p tview.Primitive)) {
	roomView, ok := view.rooms[view.CurrentRoomID()]
	if ok {
		delegate(roomView)
	}
}

func (view *MainView) addRoom(index int, room string) {
	roomStore := view.matrix.GetRoom(room)

	view.roomList.AddItem(roomStore.GetTitle(), "", 0, func() {
		view.SwitchRoom(index)
	})
	if !view.roomView.HasPage(room) {
		roomView := widget.NewRoomView(roomStore).
			SetInputSubmitFunc(view.InputSubmit).
			SetInputChangedFunc(view.InputChanged).
			SetTabCompleteFunc(view.InputTabComplete).
			SetInputCapture(view.InputKeyHandler)
		view.rooms[room] = roomView
		view.roomView.AddPage(room, roomView, true, false)
		roomView.UpdateUserList()
		go view.LoadInitialHistory(room)
	}
}

func (view *MainView) GetRoom(id string) *widget.RoomView {
	return view.rooms[id]
}

func (view *MainView) HasRoom(room string) bool {
	for _, existingRoom := range view.roomIDs {
		if existingRoom == room {
			return true
		}
	}
	return false
}

func (view *MainView) AddRoom(room string) {
	if view.HasRoom(room) {
		return
	}
	view.roomIDs = append(view.roomIDs, room)
	view.addRoom(len(view.roomIDs)-1, room)
}

func (view *MainView) RemoveRoom(room string) {
	if !view.HasRoom(room) {
		return
	}
	removeIndex := 0
	if view.CurrentRoomID() == room {
		removeIndex = view.currentRoomIndex
		view.SwitchRoom(view.currentRoomIndex - 1)
	} else {
		removeIndex = sort.StringSlice(view.roomIDs).Search(room)
	}
	view.roomList.RemoveItem(removeIndex)
	view.roomIDs = append(view.roomIDs[:removeIndex], view.roomIDs[removeIndex+1:]...)
	view.roomView.RemovePage(room)
	delete(view.rooms, room)
	view.parent.Render()
}

func (view *MainView) SetRooms(rooms []string) {
	view.roomIDs = rooms
	view.roomList.Clear()
	view.roomView.Clear()
	view.rooms = make(map[string]*widget.RoomView)
	for index, room := range rooms {
		view.addRoom(index, room)
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

func (view *MainView) AddServiceMessage(room, message string) {
	roomView, ok := view.rooms[room]
	if ok {
		message := roomView.NewMessage("", "*", message, time.Now())
		roomView.AddMessage(message, widget.AppendMessage)
		view.parent.Render()
	}
}

func (view *MainView) LoadMoreHistory(room string) {
	view.LoadHistory(room, false)
}

func (view *MainView) LoadInitialHistory(room string) {
	view.LoadHistory(room, true)
}

func (view *MainView) LoadHistory(room string, initial bool) {
	defer view.gmx.Recover()
	roomView := view.rooms[room]

	batch := roomView.Room.PrevBatch
	lockTime := time.Now().Unix() + 1

	roomView.Room.LockHistory()
	roomView.MessageView().LoadingMessages = true
	defer func() {
		roomView.Room.UnlockHistory()
		roomView.MessageView().LoadingMessages = false
	}()

	// There's no clean way to try to lock a mutex, so we just check if we still
	// want to continue after we get the lock. This function should always be ran
	// in a goroutine, so the blocking doesn't matter.
	if time.Now().Unix() >= lockTime || batch != roomView.Room.PrevBatch {
		return
	}

	if initial {
		batch = view.config.Session.NextBatch
	}
	debug.Print("Loading history for", room, "starting from", batch, "(initial:", initial, ")")
	history, prevBatch, err := view.matrix.GetHistory(roomView.Room.ID, batch, 50)
	if err != nil {
		view.AddServiceMessage(room, "Failed to fetch history")
		debug.Print("Failed to fetch history for", roomView.Room.ID, err)
		return
	}
	roomView.Room.PrevBatch = prevBatch
	for _, evt := range history {
		var room *widget.RoomView
		var message *types.Message
		if evt.Type == "m.room.message" {
			room, message = view.ProcessMessageEvent(&evt)
		} else if evt.Type == "m.room.member" {
			room, message = view.ProcessMembershipEvent(&evt, false)
		}
		if room != nil && message != nil {
			room.AddMessage(message, widget.PrependMessage)
		}
	}
	view.parent.Render()
}

func (view *MainView) ProcessMessageEvent(evt *gomatrix.Event) (room *widget.RoomView, message *types.Message) {
	room = view.GetRoom(evt.RoomID)
	if room != nil {
		text, _ := evt.Content["body"].(string)
		message = room.NewMessage(evt.ID, evt.Sender, text, unixToTime(evt.Timestamp))
	}
	return
}

func (view *MainView) processOwnMembershipChange(evt *gomatrix.Event) {
	membership, _ := evt.Content["membership"].(string)
	prevMembership := "leave"
	if evt.Unsigned.PrevContent != nil {
		prevMembership, _ = evt.Unsigned.PrevContent["membership"].(string)
	}
	if membership == prevMembership {
		return
	}
	if membership == "join" {
		view.AddRoom(evt.RoomID)
	} else if membership == "leave" {
		view.RemoveRoom(evt.RoomID)
	}
}

func (view *MainView) ProcessMembershipEvent(evt *gomatrix.Event, new bool) (room *widget.RoomView, message *types.Message) {
	if new && evt.StateKey != nil && *evt.StateKey == view.config.Session.UserID {
		view.processOwnMembershipChange(evt)
	}

	room = view.GetRoom(evt.RoomID)
	if room != nil {
		membership, _ := evt.Content["membership"].(string)
		var sender, text string
		if membership == "invite" {
			sender = "---"
			text = fmt.Sprintf("%s invited %s.", evt.Sender, *evt.StateKey)
		} else if membership == "join" {
			sender = "-->"
			text = fmt.Sprintf("%s joined the room.", *evt.StateKey)
		} else if membership == "leave" {
			sender = "<--"
			if evt.Sender != *evt.StateKey {
				reason, _ := evt.Content["reason"].(string)
				text = fmt.Sprintf("%s kicked %s: %s", evt.Sender, *evt.StateKey, reason)
			} else {
				text = fmt.Sprintf("%s left the room.", *evt.StateKey)
			}
		} else {
			room = nil
			return
		}
		message = room.NewMessage(evt.ID, sender, text, unixToTime(evt.Timestamp))
		message.TextColor = tcell.ColorGreen
	}
	return
}

func unixToTime(unix int64) time.Time {
	timestamp := time.Now()
	if unix != 0 {
		timestamp = time.Unix(unix/1000, unix%1000*1000)
	}
	return timestamp
}
