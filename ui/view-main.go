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

package ui

import (
	"bufio"
	"fmt"
	"os"
	"sync/atomic"
	"time"
	"strings"

	sync "github.com/sasha-s/go-deadlock"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/notification"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mautrix/pushrules"

	"gitlab.com/tslocum/cbind"
	tcell_v2 "github.com/gdamore/tcell/v2"
)

type MainView struct {
	flex *mauview.Flex

	roomList     *RoomList
	roomView     *mauview.Box
	currentRoom  *RoomView
	rooms        map[id.RoomID]*RoomView
	roomsLock    sync.RWMutex
	cmdProcessor *CommandProcessor
	focused      mauview.Focusable

	modal mauview.Component

	lastFocusTime time.Time

	matrix 		  ifc.MatrixContainer
	gmx    		  ifc.Gomuks
	config 		  *config.Config
	parent  	  *GomuksUI
	callbacks	  map[string]func()
}

func (ui *GomuksUI) NewMainView() mauview.Component {
	mainView := &MainView{
		flex:     mauview.NewFlex().SetDirection(mauview.FlexColumn),
		roomView: mauview.NewBox(nil).SetBorder(false),
		rooms:    make(map[id.RoomID]*RoomView),

		matrix: ui.gmx.Matrix(),
		gmx:    ui.gmx,
		config: ui.gmx.Config(),
		parent: ui,
		callbacks: make(map[string]func()),
	}

	mainView.roomList = NewRoomList(mainView)
	mainView.cmdProcessor = NewCommandProcessor(mainView)

	mainView.flex.
		AddFixedComponent(mainView.roomList, 25).
		AddFixedComponent(widget.NewBorder(), 1).
		AddProportionalComponent(mainView.roomView, 1)
	mainView.BumpFocus(nil)

	mainView.callbacks = map[string]func() {
		"previous_room": mainView.PreviousRoomCallback,
		"next_room": mainView.NextRoomCallback,
		"goto_room": mainView.SearchRoomCallback,
		"active_room": mainView.NextWithActivityCallback,
		"word_left": mainView.MoveWordLeftCallback,
		"word_right": mainView.MoveWordRightCallback,
		"char_left": mainView.MoveCharLeftCallback,
		"char_right": mainView.MoveCharRightCallback,
		"scroll_down": mainView.ScrollDownCallback,
		"scroll_up": mainView.ScrollUpCallback,
		"remove_char": mainView.RemoveNextCharCallback,
		"remove_previous_word": mainView.RemovePreviousWordCallback,
		"remove_next_word": mainView.RemoveNextWordCallback,
		"move_to_beginning": mainView.MoveToBeginningCallback,
		"move_to_end": mainView.MoveToEndCallback,
		"kill_to_beginning": mainView.KillToBeginningCallback,
		"kill_to_end": mainView.KillToEndCallback,
	}

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
		view.matrix.SendTyping(roomView.Room.ID, len(text) > 0 && text[0] != '/')
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
		_, _, _ = reader.ReadRune()
		print("\033[2J\033[0;0H")
	})
}

func (view *MainView) OpenSyncingModal() ifc.SyncingModal {
	component, modal := NewSyncingModal(view)
	view.ShowModal(component)
	return modal
}

func (view *MainView) PreviousRoomCallback() {
	view.SwitchRoom(view.roomList.Previous())
}

func (view *MainView) NextRoomCallback() {
	view.SwitchRoom(view.roomList.Next())
}

func (view *MainView) SearchRoomCallback() {
	view.ShowModal(NewFuzzySearchModal(view, 42, 12))
}

func (view *MainView) ScrollUpCallback() {
	msgView := view.currentRoom.MessageView()
	if msgView.IsAtTop() {
		go view.LoadHistory(view.currentRoom.Room.ID)
	}
	msgView.AddScrollOffset(+msgView.Height() / 2)
}

func (view *MainView) ScrollDownCallback() {
	msgView := view.currentRoom.MessageView()
	msgView.AddScrollOffset(-msgView.Height() / 2)
}

func (view *MainView) NextWithActivityCallback() {
	view.SwitchRoom(view.roomList.NextWithActivity())
}

func (view *MainView) ShowBareCallback() {
	view.ShowBare(view.currentRoom)
}

func (view *MainView) MoveWordLeftCallback() {
	view.currentRoom.input.MoveCursorLeft(true, false)
}

func (view *MainView) MoveWordRightCallback() {
	view.currentRoom.input.MoveCursorRight(true, false)
}

func (view *MainView) MoveCharLeftCallback() {
	view.currentRoom.input.MoveCursorLeft(false, false)
}

func (view *MainView) MoveCharRightCallback() {
	view.currentRoom.input.MoveCursorLeft(false, false)
}

func (view *MainView) RemoveNextCharCallback() {
	view.currentRoom.input.RemoveNextCharacter()
}

func (view *MainView) RemovePreviousCharCallback() {
	view.currentRoom.input.RemovePreviousCharacter()
}

func (view *MainView) RemoveNextWordCallback() {
	field := view.currentRoom.input
	if field.GetSelectedText() != "" {
		field.RemoveSelection()
		return
	}
	left := field.GetText()[0:field.GetCursorOffset()]
	right_index := strings.Index(field.GetText()[field.GetCursorOffset(): ], " ")
	right := field.GetText()[field.GetCursorOffset() + 1 + right_index: ]
	if right_index == -1 {
		right = ""
	}
	replacement := left + right
	field.SetText(replacement)
}

func (view *MainView) RemovePreviousWordCallback() {
	view.currentRoom.input.RemovePreviousWord()
}

func (view *MainView) MoveToBeginningCallback() {
	view.currentRoom.input.SetCursorPos(0, 0)
}

func (view *MainView) MoveToEndCallback() {
	view.currentRoom.input.SetCursorPos(999, 999)
}

func (view *MainView) KillToEndCallback() {
	field := view.currentRoom.input
	left := field.GetText()[0:field.GetCursorOffset()]
	field.SetText(left)
}

func (view *MainView) KillToBeginningCallback() {
	field := view.currentRoom.input
	right := field.GetText()[field.GetCursorOffset():]
	field.SetText(right)
	field.SetCursorPos(0, 0)
}

func (view *MainView) OnKeyEvent(event mauview.KeyEvent) bool {
	view.BumpFocus(view.currentRoom)

	if view.modal != nil {
		return view.modal.OnKeyEvent(event)
	}

	key_string, _ := cbind.Encode(
		tcell_v2.ModMask(event.Modifiers()),
		tcell_v2.Key(event.Key()),
		event.Rune())

	callback_string, _ := view.config.Keybindings.Keybindings[key_string]

	if callback_string != "" {
		callback, _ := view.callbacks[callback_string]
		if callback != nil {
			callback()
			return true
		}
	}

	k := event.Key()
	c := event.Rune()
	if event.Modifiers() == tcell.ModCtrl || event.Modifiers() == tcell.ModAlt {
		switch {
		case k == tcell.KeyDown:
			view.SwitchRoom(view.roomList.Next())
		case k == tcell.KeyUp:
			view.SwitchRoom(view.roomList.Previous())
		case c == 'k' || k == tcell.KeyCtrlK:
			view.ShowModal(NewFuzzySearchModal(view, 42, 12))
		case k == tcell.KeyHome:
			msgView := view.currentRoom.MessageView()
			msgView.AddScrollOffset(msgView.TotalHeight())
		case k == tcell.KeyEnd:
			msgView := view.currentRoom.MessageView()
			msgView.AddScrollOffset(-msgView.TotalHeight())
		case k == tcell.KeyEnter:
			return view.flex.OnKeyEvent(tcell.NewEventKey(tcell.KeyEnter, '\n', event.Modifiers()|tcell.ModShift, ""))
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
	view.switchRoom(tag, room, true)
}

func (view *MainView) switchRoom(tag string, room *rooms.Room, lock bool) {
	if room == nil {
		return
	}
	room.Load()

	roomView, ok := view.getRoomView(room.ID, lock)
	if !ok {
		debug.Print("Tried to switch to room with nonexistent roomView!")
		debug.Print(tag, room)
		return
	}
	roomView.Update()
	view.roomView.SetInnerComponent(roomView)
	view.currentRoom = roomView
	view.MarkRead(roomView)
	view.roomList.SetSelected(tag, room)
	view.flex.SetFocused(view.roomView)
	view.focused = view.roomView
	view.roomView.Focus()
	view.parent.Render()

	if msgView := roomView.MessageView(); len(msgView.messages) < 20 && !msgView.initialHistoryLoaded {
		msgView.initialHistoryLoaded = true
		go view.LoadHistory(room.ID)
	}
	if !room.MembersFetched {
		go func() {
			err := view.matrix.FetchMembers(room)
			if err != nil {
				debug.Print("Error fetching members:", err)
				return
			}
			roomView.UpdateUserList()
			view.parent.Render()
		}()
	}
}

func (view *MainView) addRoomPage(room *rooms.Room) *RoomView {
	if _, ok := view.rooms[room.ID]; !ok {
		roomView := NewRoomView(view, room).
			SetInputChangedFunc(view.InputChanged)
		view.rooms[room.ID] = roomView
		return roomView
	}
	return nil
}

func (view *MainView) GetRoom(roomID id.RoomID) ifc.RoomView {
	room, ok := view.getRoomView(roomID, true)
	if !ok {
		return view.addRoom(view.matrix.GetOrCreateRoom(roomID))
	}
	return room
}

func (view *MainView) getRoomView(roomID id.RoomID, lock bool) (room *RoomView, ok bool) {
	if lock {
		view.roomsLock.RLock()
		room, ok = view.rooms[roomID]
		view.roomsLock.RUnlock()
	} else {
		room, ok = view.rooms[roomID]
	}
	return room, ok
}

func (view *MainView) AddRoom(room *rooms.Room) {
	view.addRoom(room)
}

func (view *MainView) RemoveRoom(room *rooms.Room) {
	view.roomsLock.Lock()
	_, ok := view.getRoomView(room.ID, false)
	if !ok {
		view.roomsLock.Unlock()
		debug.Print("Remove aborted (not found)", room.ID, room.GetTitle())
		return
	}
	debug.Print("Removing", room.ID, room.GetTitle())

	view.roomList.Remove(room)
	t, r := view.roomList.Selected()
	view.switchRoom(t, r, false)
	delete(view.rooms, room.ID)
	view.roomsLock.Unlock()

	view.parent.Render()
}

func (view *MainView) addRoom(room *rooms.Room) *RoomView {
	if view.roomList.Contains(room.ID) {
		debug.Print("Add aborted (room exists)", room.ID, room.GetTitle())
		return nil
	}
	debug.Print("Adding", room.ID, room.GetTitle())
	view.roomList.Add(room)
	view.roomsLock.Lock()
	roomView := view.addRoomPage(room)
	if !view.roomList.HasSelected() {
		t, r := view.roomList.First()
		view.switchRoom(t, r, false)
	}
	view.roomsLock.Unlock()
	return roomView
}

func (view *MainView) SetRooms(rooms *rooms.RoomCache) {
	view.roomList.Clear()
	view.roomsLock.Lock()
	view.rooms = make(map[id.RoomID]*RoomView)
	for _, room := range rooms.Map {
		if room.HasLeft {
			continue
		}
		view.roomList.Add(room)
		view.addRoomPage(room)
	}
	t, r := view.roomList.First()
	view.switchRoom(t, r, false)
	view.roomsLock.Unlock()
}

func (view *MainView) UpdateTags(room *rooms.Room) {
	if !view.roomList.Contains(room.ID) {
		return
	}
	reselect := view.roomList.selected == room
	view.roomList.Remove(room)
	view.roomList.Add(room)
	if reselect {
		view.roomList.SetSelected(room.Tags()[0].Tag, room)
	}
	view.parent.Render()
}

func (view *MainView) SetTyping(roomID id.RoomID, users []id.UserID) {
	roomView, ok := view.getRoomView(roomID, true)
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

func (view *MainView) Bump(room *rooms.Room) {
	view.roomList.Bump(room)
}

func (view *MainView) NotifyMessage(room *rooms.Room, message ifc.Message, should pushrules.PushActionArrayShould) {
	view.Bump(room)
	uiMsg, ok := message.(*messages.UIMessage)
	if ok && uiMsg.SenderID == view.config.UserID {
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

	if shouldNotify && !recentlyFocused && !view.config.Preferences.DisableNotifications {
		// Push rules say notify and the terminal is not focused, send desktop notification.
		shouldPlaySound := should.PlaySound &&
			should.SoundName == "default" &&
			view.config.NotifySound
		sendNotification(room, message.NotificationSenderName(), message.NotificationContent(), should.Highlight, shouldPlaySound)
	}

	// TODO this should probably happen somewhere else
	//      (actually it's probably completely broken now)
	message.SetIsHighlight(should.Highlight)
}

func (view *MainView) LoadHistory(roomID id.RoomID) {
	defer debug.Recover()
	roomView, ok := view.getRoomView(roomID, true)
	if !ok {
		return
	}
	msgView := roomView.MessageView()

	if !atomic.CompareAndSwapInt32(&msgView.loadingMessages, 0, 1) {
		// Locked
		return
	}
	defer atomic.StoreInt32(&msgView.loadingMessages, 0)
	// Update the "Loading more messages..." text
	view.parent.Render()

	history, newLoadPtr, err := view.matrix.GetHistory(roomView.Room, 50, msgView.historyLoadPtr)
	if err != nil {
		roomView.AddServiceMessage("Failed to fetch history")
		debug.Print("Failed to fetch history for", roomView.Room.ID, err)
		view.parent.Render()
		return
	}
	//debug.Printf("Load pointer %d -> %d", msgView.historyLoadPtr, newLoadPtr)
	msgView.historyLoadPtr = newLoadPtr
	for _, evt := range history {
		roomView.AddHistoryEvent(evt)
	}
	view.parent.Render()
}
