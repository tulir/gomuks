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

package ui

import (
	"bufio"
	"fmt"
	"os"
	"time"
	"unicode"

	"github.com/kyokomi/emoji"

	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/notification"
	"maunium.net/go/gomuks/matrix/pushrules"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/widget"
)

type MainView struct {
	flex *mauview.Flex

	roomList     *RoomList
	roomView     *mauview.Box
	currentRoom  *RoomView
	rooms        map[string]*RoomView
	cmdProcessor *CommandProcessor
	focused      mauview.Focusable

	modal mauview.Component

	lastFocusTime time.Time

	matrix ifc.MatrixContainer
	gmx    ifc.Gomuks
	config *config.Config
	parent *GomuksUI
}

func (ui *GomuksUI) NewMainView() mauview.Component {
	mainView := &MainView{
		flex:     mauview.NewFlex().SetDirection(mauview.FlexColumn),
		roomView: mauview.NewBox(nil).SetBorder(false),
		rooms:    make(map[string]*RoomView),

		matrix: ui.gmx.Matrix(),
		gmx:    ui.gmx,
		config: ui.gmx.Config(),
		parent: ui,
	}
	mainView.roomList = NewRoomList(mainView)
	mainView.cmdProcessor = NewCommandProcessor(mainView)

	mainView.flex.
		AddFixedComponent(mainView.roomList, 25).
		AddFixedComponent(widget.NewBorder(), 1).
		AddProportionalComponent(mainView.roomView, 1)
	mainView.BumpFocus(nil)

	ui.mainView = mainView

	return mainView
}

func (view *MainView) ShowModal(modal mauview.Component) {
	view.modal = modal
	var ok bool
	view.focused, ok = modal.(mauview.Focusable)
	if !ok {
		view.focused = nil
	} else {
		view.focused.Focus()
	}
}

func (view *MainView) HideModal() {
	view.modal = nil
	view.focused = view.roomView
}

func (view *MainView) Draw(screen mauview.Screen) {
	if view.config.Preferences.HideRoomList {
		view.roomView.Draw(screen)
	} else {
		view.flex.Draw(screen)
	}

	if view.modal != nil {
		view.modal.Draw(screen)
	}
}

func (view *MainView) BumpFocus(roomView *RoomView) {
	if roomView != nil {
		view.lastFocusTime = time.Now()
		view.MarkRead(roomView)
	}
}

func (view *MainView) MarkRead(roomView *RoomView) {
	if roomView != nil && roomView.Room.HasNewMessages() && roomView.MessageView().ScrollOffset == 0 {
		msgList := roomView.MessageView().messages
		if len(msgList) > 0 {
			msg := msgList[len(msgList)-1]
			if roomView.Room.MarkRead(msg.ID()) {
				view.matrix.MarkRead(roomView.Room.ID, msg.ID())
			}
		}
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
			if respErr := httpErr.RespError; respErr != nil {
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
	if roomView == nil {
		return
	}
	_, height := view.parent.app.Screen().Size()
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

func (view *MainView) OnKeyEvent(event mauview.KeyEvent) bool {
	view.BumpFocus(view.currentRoom)

	if view.modal != nil {
		return view.modal.OnKeyEvent(event)
	}

	k := event.Key()
	c := event.Rune()
	if event.Modifiers() == tcell.ModCtrl || event.Modifiers() == tcell.ModAlt {
		switch {
		case k == tcell.KeyDown:
			view.SwitchRoom(view.roomList.Next())
		case k == tcell.KeyUp:
			view.SwitchRoom(view.roomList.Previous())
		case k == tcell.KeyEnter:
			view.ShowModal(NewFuzzySearchModal(view, 42, 12))
		case k == tcell.KeyHome:
			msgView := view.currentRoom.MessageView()
			msgView.AddScrollOffset(msgView.TotalHeight())
		case k == tcell.KeyEnd:
			msgView := view.currentRoom.MessageView()
			msgView.AddScrollOffset(-msgView.TotalHeight())
		case c == 'n' || k == tcell.KeyCtrlN:
			return view.flex.OnKeyEvent(tcell.NewEventKey(tcell.KeyEnter, '\n', event.Modifiers()|tcell.ModShift))
		case c == 'a':
			view.SwitchRoom(view.roomList.NextWithActivity())
		case c == 'l' || k == tcell.KeyCtrlL:
			view.ShowBare(view.currentRoom)
		default:
			goto defaultHandler
		}
		return true
	}
defaultHandler:
	if view.config.Preferences.HideRoomList {
		return view.roomView.OnKeyEvent(event)
	}
	return view.flex.OnKeyEvent(event)
}

const WheelScrollOffsetDiff = 3

func (view *MainView) OnMouseEvent(event mauview.MouseEvent) bool {
	if event.HasMotion() {
		return false
	}
	if view.modal != nil {
		return view.modal.OnMouseEvent(event)
	}
	if view.config.Preferences.HideRoomList {
		return view.roomView.OnMouseEvent(event)
	}
	return view.flex.OnMouseEvent(event)
}

func (view *MainView) OnPasteEvent(event mauview.PasteEvent) bool {
	if view.modal != nil {
		return view.modal.OnPasteEvent(event)
	} else if view.config.Preferences.HideRoomList {
		return view.roomView.OnPasteEvent(event)
	}
	return view.flex.OnPasteEvent(event)
}

func (view *MainView) Focus() {
	if view.focused != nil {
		view.focused.Focus()
	}
}

func (view *MainView) Blur() {
	if view.focused != nil {
		view.focused.Blur()
	}
}

func (view *MainView) SwitchRoom(tag string, room *rooms.Room) {
	if room == nil {
		return
	}

	roomView := view.rooms[room.ID]
	if roomView == nil {
		debug.Print("Tried to switch to non-nil room with nil roomView!")
		debug.Print(tag, room)
		return
	}
	view.roomView.SetInnerComponent(roomView)
	view.currentRoom = roomView
	view.MarkRead(roomView)
	view.roomList.SetSelected(tag, room)
	view.parent.Render()
	if len(roomView.MessageView().messages) == 0 {
		go view.LoadHistory(room.ID)
	}
}

func (view *MainView) addRoomPage(room *rooms.Room) {
	if _, ok := view.rooms[room.ID]; !ok {
		roomView := NewRoomView(view, room).
			SetInputSubmitFunc(view.InputSubmit).
			SetInputChangedFunc(view.InputChanged)
		view.rooms[room.ID] = roomView
		roomView.UpdateUserList()

		// FIXME
		/*_, err := roomView.LoadHistory(view.matrix, view.config.HistoryDir)
		if err != nil {
			debug.Printf("Failed to load history of %s: %v", roomView.Room.GetTitle(), err)
		}*/
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

	delete(view.rooms, room.ID)

	view.parent.Render()
}

func (view *MainView) SetRooms(rooms map[string]*rooms.Room) {
	view.roomList.Clear()
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

	history, err := view.matrix.GetHistory(roomView.Room, 50)
	if err != nil {
		roomView.AddServiceMessage("Failed to fetch history")
		debug.Print("Failed to fetch history for", roomView.Room.ID, err)
		return
	}
	for _, evt := range history {
		message := view.ParseEvent(roomView, evt)
		if message != nil {
			roomView.AddMessage(message, ifc.PrependMessage)
		}
	}
	// TODO?
	/*err = roomView.SaveHistory(view.config.HistoryDir)
	if err != nil {
		debug.Printf("Failed to save history of %s: %v", roomView.Room.GetTitle(), err)
	}*/
	view.config.PutRoom(roomView.Room)
	view.parent.Render()
}

func (view *MainView) ParseEvent(roomView ifc.RoomView, evt *mautrix.Event) ifc.Message {
	return messages.ParseEvent(view.matrix, roomView.MxRoom(), evt)
}
