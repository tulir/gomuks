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

	roomList         *widget.RoomList
	roomView         *tview.Pages
	rooms            map[string]*widget.RoomView
	currentRoomIndex int
	roomIDs          []string

	lastFocusTime time.Time

	matrix ifc.MatrixContainer
	gmx    ifc.Gomuks
	config *config.Config
	parent *GomuksUI
}

func (ui *GomuksUI) NewMainView() tview.Primitive {
	mainView := &MainView{
		Flex:     tview.NewFlex(),
		roomList: widget.NewRoomList(),
		roomView: tview.NewPages(),
		rooms:    make(map[string]*widget.RoomView),

		matrix: ui.gmx.Matrix(),
		gmx:    ui.gmx,
		config: ui.gmx.Config(),
		parent: ui,
	}

	mainView.SetDirection(tview.FlexColumn)
	mainView.AddItem(mainView.roomList, 25, 0, false)
	mainView.AddItem(widget.NewBorder(), 1, 0, false)
	mainView.AddItem(mainView.roomView, 0, 1, true)

	ui.mainView = mainView

	return mainView
}

func (view *MainView) BumpFocus() {
	view.lastFocusTime = time.Now()
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
		go view.HandleCommand(roomView, command, args)
	} else {
		view.SendMessage(roomView, text)
	}
	roomView.SetInputText("")
}

func (view *MainView) SendMessage(roomView *widget.RoomView, text string) {
	tempMessage := roomView.NewTempMessage("m.text", text)
	go view.sendTempMessage(roomView, tempMessage)
}

func (view *MainView) sendTempMessage(roomView *widget.RoomView, tempMessage *types.Message) {
	defer view.gmx.Recover()
	eventID, err := view.matrix.SendMessage(roomView.Room.ID, tempMessage.Type, tempMessage.Text)
	if err != nil {
		tempMessage.State = types.MessageStateFailed
		roomView.SetStatus(fmt.Sprintf("Failed to send message: %s", err))
	} else {
		roomView.MessageView().UpdateMessageID(tempMessage, eventID)
	}
}

func (view *MainView) HandleCommand(roomView *widget.RoomView, command string, args []string) {
	defer view.gmx.Recover()
	debug.Print("Handling command", command, args)
	switch command {
	case "/me":
		tempMessage := roomView.NewTempMessage("m.emote", strings.Join(args, " "))
		go view.sendTempMessage(roomView, tempMessage)
		view.parent.Render()
	case "/quit":
		view.gmx.Stop()
	case "/clearcache":
		view.config.Clear()
		view.gmx.Stop()
	case "/panic":
		panic("This is a test panic.")
	case "/part":
		fallthrough
	case "/leave":
		debug.Print("Leave room result:", view.matrix.LeaveRoom(roomView.Room.ID))
	case "/join":
		if len(args) == 0 {
			view.AddServiceMessage(roomView, "Usage: /join <room>")
			break
		}
		debug.Print("Join room result:", view.matrix.JoinRoom(args[0]))
	default:
		view.AddServiceMessage(roomView, "Unknown command.")
	}
}

func (view *MainView) KeyEventHandler(roomView *widget.RoomView, key *tcell.EventKey) *tcell.EventKey {
	view.BumpFocus()

	k := key.Key()
	if key.Modifiers() == tcell.ModCtrl || key.Modifiers() == tcell.ModAlt {
		switch k {
		case tcell.KeyDown:
			view.SwitchRoom(view.currentRoomIndex + 1)
		case tcell.KeyUp:
			view.SwitchRoom(view.currentRoomIndex - 1)
		default:
			return key
		}
	} else if k == tcell.KeyPgUp || k == tcell.KeyPgDn || k == tcell.KeyUp || k == tcell.KeyDown || k == tcell.KeyEnd || k == tcell.KeyHome {
		msgView := roomView.MessageView()

		if msgView.IsAtTop() && (k == tcell.KeyPgUp || k == tcell.KeyUp) {
			go view.LoadHistory(roomView.Room.ID, false)
		}

		switch k {
		case tcell.KeyPgUp:
			msgView.AddScrollOffset(msgView.Height() / 2)
		case tcell.KeyPgDn:
			msgView.AddScrollOffset(-msgView.Height() / 2)
		case tcell.KeyUp:
			msgView.AddScrollOffset(1)
		case tcell.KeyDown:
			msgView.AddScrollOffset(-1)
		case tcell.KeyHome:
			msgView.AddScrollOffset(msgView.TotalHeight())
		case tcell.KeyEnd:
			msgView.AddScrollOffset(-msgView.TotalHeight())
		}
	} else {
		return key
	}
	return nil
}

const WheelScrollOffsetDiff = 3

func (view *MainView) MouseEventHandler(roomView *widget.RoomView, event *tcell.EventMouse) *tcell.EventMouse {
	if event.Buttons() == tcell.ButtonNone {
		return event
	}
	view.BumpFocus()

	msgView := roomView.MessageView()
	x, y := event.Position()

	switch event.Buttons() {
	case tcell.WheelUp:
		if msgView.IsAtTop() {
			go view.LoadHistory(roomView.Room.ID, false)
		} else {
			msgView.AddScrollOffset(WheelScrollOffsetDiff)

			view.parent.Render()
		}
	case tcell.WheelDown:
		msgView.AddScrollOffset(-WheelScrollOffsetDiff)

		view.parent.Render()
	default:
		debug.Print("Mouse event received:", event.Buttons(), event.Modifiers(), x, y)
		return event
	}

	return event
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
	view.roomList.SetSelected(view.rooms[view.CurrentRoomID()].Room)
	view.gmx.App().SetFocus(view)
	view.parent.Render()
}

func (view *MainView) Focus(delegate func(p tview.Primitive)) {
	roomView, ok := view.rooms[view.CurrentRoomID()]
	if ok {
		delegate(roomView)
	}
}

func (view *MainView) SaveAllHistory() {
	for _, room := range view.rooms {
		err := room.SaveHistory(view.config.HistoryDir)
		if err != nil {
			debug.Printf("Failed to save history of %s: %v", room.Room.GetTitle(), err)
		}
	}
}

func (view *MainView) addRoom(index int, room string) {
	roomStore := view.matrix.GetRoom(room)

	view.roomList.Add(roomStore)
	if !view.roomView.HasPage(room) {
		roomView := widget.NewRoomView(roomStore).
			SetInputSubmitFunc(view.InputSubmit).
			SetInputChangedFunc(view.InputChanged).
			SetTabCompleteFunc(view.InputTabComplete).
			SetInputCapture(view.KeyEventHandler).
			SetMouseCapture(view.MouseEventHandler)
		view.rooms[room] = roomView
		view.roomView.AddPage(room, roomView, true, false)
		roomView.UpdateUserList()

		count, err := roomView.LoadHistory(view.config.HistoryDir)
		if err != nil {
			debug.Printf("Failed to load history of %s: %v", roomView.Room.GetTitle(), err)
		} else if count <= 0 {
			go view.LoadHistory(room, true)
		}
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
	roomView := view.GetRoom(room)
	if roomView == nil {
		return
	}
	removeIndex := 0
	if view.CurrentRoomID() == room {
		removeIndex = view.currentRoomIndex
		view.SwitchRoom(view.currentRoomIndex - 1)
	} else {
		removeIndex = sort.StringSlice(view.roomIDs).Search(room)
	}
	view.roomList.Remove(roomView.Room)
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

func (view *MainView) AddServiceMessage(roomView *widget.RoomView, text string) {
	message := roomView.NewMessage("", "*", "gomuks.service", text, time.Now())
	message.TextColor = tcell.ColorGray
	message.SenderColor = tcell.ColorGray
	roomView.AddMessage(message, widget.AppendMessage)
	view.parent.Render()
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
		debug.Print("Loading initial history for", room)
	} else {
		debug.Print("Loading more history for", room, "starting from", batch)
	}
	history, prevBatch, err := view.matrix.GetHistory(roomView.Room.ID, batch, 50)
	if err != nil {
		view.AddServiceMessage(roomView, "Failed to fetch history")
		debug.Print("Failed to fetch history for", roomView.Room.ID, err)
		return
	}
	roomView.Room.PrevBatch = prevBatch
	for _, evt := range history {
		var message *types.Message
		if evt.Type == "m.room.message" {
			message = view.ProcessMessageEvent(roomView, &evt)
		} else if evt.Type == "m.room.member" {
			message = view.ProcessMembershipEvent(roomView, &evt)
		}
		if message != nil {
			roomView.AddMessage(message, widget.PrependMessage)
		}
	}
	err = roomView.SaveHistory(view.config.HistoryDir)
	if err != nil {
		debug.Printf("Failed to save history of %s: %v", roomView.Room.GetTitle(), err)
	}
	view.config.Session.Save()
	view.parent.Render()
}

func (view *MainView) ProcessMessageEvent(room *widget.RoomView, evt *gomatrix.Event) (message *types.Message) {
	text, _ := evt.Content["body"].(string)
	msgtype, _ := evt.Content["msgtype"].(string)
	return room.NewMessage(evt.ID, evt.Sender, msgtype, text, unixToTime(evt.Timestamp))
}

func (view *MainView) getMembershipEventContent(evt *gomatrix.Event) (sender, text string) {
	membership, _ := evt.Content["membership"].(string)
	displayname, _ := evt.Content["displayname"].(string)
	if len(displayname) == 0 {
		displayname = *evt.StateKey
	}
	prevMembership := "leave"
	prevDisplayname := ""
	if evt.Unsigned.PrevContent != nil {
		prevMembership, _ = evt.Unsigned.PrevContent["membership"].(string)
		prevDisplayname, _ = evt.Unsigned.PrevContent["displayname"].(string)
	}

	if membership != prevMembership {
		switch membership {
		case "invite":
			sender = "---"
			text = fmt.Sprintf("%s invited %s.", evt.Sender, displayname)
		case "join":
			sender = "-->"
			text = fmt.Sprintf("%s joined the room.", displayname)
		case "leave":
			sender = "<--"
			if evt.Sender != *evt.StateKey {
				reason, _ := evt.Content["reason"].(string)
				text = fmt.Sprintf("%s kicked %s: %s", evt.Sender, displayname, reason)
			} else {
				text = fmt.Sprintf("%s left the room.", displayname)
			}
		}
	} else if displayname != prevDisplayname {
		sender = "---"
		text = fmt.Sprintf("%s changed their display name to %s.", prevDisplayname, displayname)
	}
	return
}

func (view *MainView) ProcessMembershipEvent(room *widget.RoomView, evt *gomatrix.Event) (message *types.Message) {
	sender, text := view.getMembershipEventContent(evt)
	if len(text) == 0 {
		return
	}
	message = room.NewMessage(evt.ID, sender, "m.room.member", text, unixToTime(evt.Timestamp))
	message.TextColor = tcell.ColorGreen
	return
}

func unixToTime(unix int64) time.Time {
	timestamp := time.Now()
	if unix != 0 {
		timestamp = time.Unix(unix/1000, unix%1000*1000)
	}
	return timestamp
}
