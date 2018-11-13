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
	"time"
	"unicode"

	"github.com/kyokomi/emoji"

	"bufio"
	"os"

	"maunium.net/go/mautrix"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/notification"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages/parser"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type MainView struct {
	*tview.Flex

	roomList     *RoomList
	roomView     *tview.Pages
	rooms        map[string]*RoomView
	cmdProcessor *CommandProcessor

	lastFocusTime time.Time

	matrix ifc.MatrixContainer
	gmx    ifc.Gomuks
	config *config.Config
	parent *GomuksUI
}

func (ui *GomuksUI) NewMainView() tview.Primitive {
	mainView := &MainView{
		Flex:     tview.NewFlex(),
		roomList: NewRoomList(),
		roomView: tview.NewPages(),
		rooms:    make(map[string]*RoomView),

		matrix: ui.gmx.Matrix(),
		gmx:    ui.gmx,
		config: ui.gmx.Config(),
		parent: ui,
	}
	mainView.cmdProcessor = NewCommandProcessor(mainView)

	mainView.
		SetDirection(tview.FlexColumn).
		AddItem(mainView.roomList, 25, 0, false).
		AddItem(widget.NewBorder(), 1, 0, false).
		AddItem(mainView.roomView, 0, 1, true)
	mainView.BumpFocus(nil)

	ui.mainView = mainView

	return mainView
}

func (view *MainView) Draw(screen tcell.Screen) {
	if view.config.Preferences.HideRoomList {
		view.roomView.SetRect(view.GetRect())
		view.roomView.Draw(screen)
	} else {
		view.Flex.Draw(screen)
	}
}

func (view *MainView) BumpFocus(roomView *RoomView) {
	view.lastFocusTime = time.Now()
	view.MarkRead(roomView)
}

func (view *MainView) MarkRead(roomView *RoomView) {
	if roomView != nil && roomView.Room.HasNewMessages() && roomView.MessageView().ScrollOffset == 0 {
		msgList := roomView.MessageView().messages
		msg := msgList[len(msgList)-1]
		roomView.Room.MarkRead(msg.ID())
		view.matrix.MarkRead(roomView.Room.ID, msg.ID())
	}
}

func (view *MainView) InputChanged(roomView *RoomView, text string) {
	if !roomView.config.Preferences.DisableTypingNotifs {
		if len(text) == 0 {
			go view.matrix.SendTyping(roomView.Room.ID, false)
		} else if text[0] != '/' {
			go view.matrix.SendTyping(roomView.Room.ID, true)
		}
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

func (view *MainView) InputSubmit(roomView *RoomView, text string) {
	if len(text) == 0 {
		return
	} else if text[0] == '/' {
		cmd := view.cmdProcessor.ParseCommand(roomView, text)
		go view.cmdProcessor.HandleCommand(cmd)
	} else {
		view.SendMessage(roomView, text)
	}
	roomView.SetInputText("")
}

func (view *MainView) SendMessage(roomView *RoomView, text string) {
	tempMessage := roomView.NewTempMessage("m.text", text)
	go view.sendTempMessage(roomView, tempMessage, text)
}

func (view *MainView) sendTempMessage(roomView *RoomView, tempMessage ifc.Message, text string) {
	defer debug.Recover()
	debug.Print("Sending message", tempMessage.Type(), text)
	if !roomView.config.Preferences.DisableEmojis {
		text = emoji.Sprint(text)
	}
	eventID, err := view.matrix.SendMarkdownMessage(roomView.Room.ID, tempMessage.Type(), text)
	if err != nil {
		tempMessage.SetState(ifc.MessageStateFailed)
		if httpErr, ok := err.(mautrix.HTTPError); ok {
			if respErr, ok := httpErr.WrappedError.(mautrix.RespError); ok {
				// Show shorter version if available
				err = respErr
			}
		}
		roomView.AddServiceMessage(fmt.Sprintf("Failed to send message: %v", err))
		view.parent.Render()
	} else {
		debug.Print("Event ID received:", eventID)
		roomView.MessageView().UpdateMessageID(tempMessage, eventID)
	}
}

func (view *MainView) ShowBare(roomView *RoomView) {
	_, height := view.parent.app.GetScreen().Size()
	view.parent.app.Suspend(func() {
		print("\033[2J\033[0;0H")
		// We don't know how much space there exactly is. Too few messages looks weird,
		// and too many messages shouldn't cause any problems, so we just show too many.
		height *= 2
		fmt.Println(roomView.MessageView().CapturePlaintext(height))
		fmt.Println("Press enter to return to normal mode.")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadRune()
		print("\033[2J\033[0;0H")
	})
}

func (view *MainView) KeyEventHandler(roomView *RoomView, key *tcell.EventKey) *tcell.EventKey {
	view.BumpFocus(roomView)

	k := key.Key()
	c := key.Rune()
	if key.Modifiers() == tcell.ModCtrl || key.Modifiers() == tcell.ModAlt {
		switch {
		case k == tcell.KeyDown:
			view.SwitchRoom(view.roomList.Next())
		case k == tcell.KeyUp:
			view.SwitchRoom(view.roomList.Previous())
		case k == tcell.KeyEnter:
			searchModal := NewFuzzySearchModal(view, 42, 12)
			view.parent.views.AddPage("fuzzy-search-modal", searchModal, true, true)
			view.parent.app.SetFocus(searchModal)
		case c == 'l':
			view.ShowBare(roomView)
		default:
			return key
		}
	} else if k == tcell.KeyAltDown || k == tcell.KeyCtrlDown {
		view.SwitchRoom(view.roomList.Next())
	} else if k == tcell.KeyAltUp || k == tcell.KeyCtrlUp {
		view.SwitchRoom(view.roomList.Previous())
	} else if k == tcell.KeyPgUp || k == tcell.KeyPgDn || k == tcell.KeyUp || k == tcell.KeyDown || k == tcell.KeyEnd || k == tcell.KeyHome {
		msgView := roomView.MessageView()

		if msgView.IsAtTop() && (k == tcell.KeyPgUp || k == tcell.KeyUp) {
			go view.LoadHistory(roomView.Room.ID)
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

func isInArea(x, y int, p tview.Primitive) bool {
	rx, ry, rw, rh := p.GetRect()
	return x >= rx && y >= ry && x < rx+rw && y < ry+rh
}

func (view *MainView) MouseEventHandler(roomView *RoomView, event *tcell.EventMouse) *tcell.EventMouse {
	if event.Buttons() == tcell.ButtonNone || event.HasMotion() {
		return event
	}
	view.BumpFocus(roomView)

	msgView := roomView.MessageView()
	x, y := event.Position()

	switch {
	case isInArea(x, y, msgView):
		mx, my, _, _ := msgView.GetRect()
		switch event.Buttons() {
		case tcell.WheelUp:
			if msgView.IsAtTop() {
				go view.LoadHistory(roomView.Room.ID)
			} else {
				msgView.AddScrollOffset(WheelScrollOffsetDiff)

				view.parent.Render()
			}
		case tcell.WheelDown:
			msgView.AddScrollOffset(-WheelScrollOffsetDiff)
			view.parent.Render()
			view.MarkRead(roomView)
		default:
			if msgView.HandleClick(x-mx, y-my, event.Buttons()) {
				view.parent.Render()
			}
		}
	case isInArea(x, y, view.roomList):
		switch event.Buttons() {
		case tcell.WheelUp:
			view.roomList.AddScrollOffset(-WheelScrollOffsetDiff)
			view.parent.Render()
		case tcell.WheelDown:
			view.roomList.AddScrollOffset(WheelScrollOffsetDiff)
			view.parent.Render()
		case tcell.Button1:
			_, rly, _, _ := msgView.GetRect()
			line := y - rly + 1
			switchToTag, switchToRoom := view.roomList.HandleClick(x, line, event.Modifiers() == tcell.ModCtrl)
			if switchToRoom != nil {
				view.SwitchRoom(switchToTag, switchToRoom)
			} else {
				view.parent.Render()
			}
		}
	default:
		debug.Print("Unhandled mouse event:", event.Buttons(), event.Modifiers(), x, y)
	}
	return event
}

func (view *MainView) SwitchRoom(tag string, room *rooms.Room) {
	if room == nil {
		return
	}

	view.roomView.SwitchToPage(room.ID)
	roomView := view.rooms[room.ID]
	if roomView == nil {
		debug.Print("Tried to switch to non-nil room with nil roomView!")
		debug.Print(tag, room)
		return
	}
	view.MarkRead(roomView)
	view.roomList.SetSelected(tag, room)
	view.parent.app.SetFocus(view)
	view.parent.Render()
}

func (view *MainView) Focus(delegate func(p tview.Primitive)) {
	room := view.roomList.SelectedRoom()
	if room != nil {
		roomView, ok := view.rooms[room.ID]
		if ok {
			delegate(roomView)
		}
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

func (view *MainView) addRoomPage(room *rooms.Room) {
	if !view.roomView.HasPage(room.ID) {
		roomView := NewRoomView(view, room).
			SetInputSubmitFunc(view.InputSubmit).
			SetInputChangedFunc(view.InputChanged).
			SetInputCapture(view.KeyEventHandler).
			SetMouseCapture(view.MouseEventHandler)
		view.rooms[room.ID] = roomView
		view.roomView.AddPage(room.ID, roomView, true, false)
		roomView.UpdateUserList()

		_, err := roomView.LoadHistory(view.matrix, view.config.HistoryDir)
		if err != nil {
			debug.Printf("Failed to load history of %s: %v", roomView.Room.GetTitle(), err)
		}
	}
}

func (view *MainView) GetRoom(roomID string) ifc.RoomView {
	room, ok := view.rooms[roomID]
	if !ok {
		view.AddRoom(view.matrix.GetRoom(roomID))
		room, ok := view.rooms[roomID]
		if !ok {
			return nil
		}
		return room
	}
	return room
}

func (view *MainView) AddRoom(room *rooms.Room) {
	if view.roomList.Contains(room.ID) {
		debug.Print("Add aborted (room exists)", room.ID, room.GetTitle())
		return
	}
	debug.Print("Adding", room.ID, room.GetTitle())
	view.roomList.Add(room)
	view.addRoomPage(room)
	if !view.roomList.HasSelected() {
		view.SwitchRoom(view.roomList.First())
	}
}

func (view *MainView) RemoveRoom(room *rooms.Room) {
	roomView := view.GetRoom(room.ID)
	if roomView == nil {
		debug.Print("Remove aborted (not found)", room.ID, room.GetTitle())
		return
	}
	debug.Print("Removing", room.ID, room.GetTitle())

	view.roomList.Remove(room)
	view.SwitchRoom(view.roomList.Selected())

	view.roomView.RemovePage(room.ID)
	delete(view.rooms, room.ID)

	view.parent.Render()
}

func (view *MainView) SetRooms(rooms map[string]*rooms.Room) {
	view.roomList.Clear()
	view.roomView.Clear()
	view.rooms = make(map[string]*RoomView)
	for _, room := range rooms {
		if room.HasLeft {
			continue
		}
		view.roomList.Add(room)
		view.addRoomPage(room)
	}
	view.SwitchRoom(view.roomList.First())
}

func (view *MainView) UpdateTags(room *rooms.Room) {
	if !view.roomList.Contains(room.ID) {
		return
	}
	view.roomList.Remove(room)
	view.roomList.Add(room)
}

func (view *MainView) SetTyping(room string, users []string) {
	roomView, ok := view.rooms[room]
	if ok {
		roomView.SetTyping(users)
		view.parent.Render()
	}
}

func sendNotification(room *rooms.Room, sender, text string, critical, sound bool) {
	if room.GetTitle() != sender {
		sender = fmt.Sprintf("%s (%s)", sender, room.GetTitle())
	}
	debug.Printf("Sending notification with body \"%s\" from %s in room ID %s (critical=%v, sound=%v)", text, sender, room.ID, critical, sound)
	notification.Send(sender, text, critical, sound)
}

func (view *MainView) NotifyMessage(room *rooms.Room, message ifc.Message, should pushrules.PushActionArrayShould) {
	view.roomList.Bump(room)
	if message.SenderID() == view.config.UserID {
		return
	}
	// Whether or not the room where the message came is the currently shown room.
	isCurrent := room == view.roomList.SelectedRoom()
	// Whether or not the terminal window is focused.
	recentlyFocused := time.Now().Add(-30 * time.Second).Before(view.lastFocusTime)
	isFocused := time.Now().Add(-5 * time.Second).Before(view.lastFocusTime)

	// Whether or not the push rules say this message should be notified about.
	shouldNotify := should.Notify || !should.NotifySpecified

	if !isCurrent || !isFocused {
		// The message is not in the current room, show new message status in room list.
		room.AddUnread(message.ID(), shouldNotify, should.Highlight)
	} else {
		view.matrix.MarkRead(room.ID, message.ID())
	}

	if shouldNotify && !recentlyFocused {
		// Push rules say notify and the terminal is not focused, send desktop notification.
		shouldPlaySound := should.PlaySound && should.SoundName == "default"
		sendNotification(room, message.Sender(), message.NotificationContent(), should.Highlight, shouldPlaySound)
	}

	message.SetIsHighlight(should.Highlight)
}

func (view *MainView) InitialSyncDone() {
	view.roomList.Clear()
	for _, room := range view.rooms {
		view.roomList.Add(room.Room)
		room.UpdateUserList()
	}
}

func (view *MainView) LoadHistory(room string) {
	defer debug.Recover()
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

	debug.Print("Fetching history for", room, "starting from", batch)
	history, prevBatch, err := view.matrix.GetHistory(roomView.Room.ID, batch, 50)
	if err != nil {
		roomView.AddServiceMessage("Failed to fetch history")
		debug.Print("Failed to fetch history for", roomView.Room.ID, err)
		return
	}
	roomView.Room.PrevBatch = prevBatch
	for _, evt := range history {
		message := view.ParseEvent(roomView, evt)
		if message != nil {
			roomView.AddMessage(message, ifc.PrependMessage)
		}
	}
	err = roomView.SaveHistory(view.config.HistoryDir)
	if err != nil {
		debug.Printf("Failed to save history of %s: %v", roomView.Room.GetTitle(), err)
	}
	view.config.PutRoom(roomView.Room)
	view.parent.Render()
}

func (view *MainView) ParseEvent(roomView ifc.RoomView, evt *mautrix.Event) ifc.Message {
	return parser.ParseEvent(view.matrix, roomView.MxRoom(), evt)
}
