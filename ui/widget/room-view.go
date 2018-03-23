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

package widget

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/types"
	"maunium.net/go/tview"
)

type RoomView struct {
	*tview.Box

	topic    *tview.TextView
	content  *MessageView
	status   *tview.TextView
	userList *tview.TextView
	ulBorder *Border
	input    *AdvancedInputField
	Room     *rooms.Room
}

func NewRoomView(room *rooms.Room) *RoomView {
	view := &RoomView{
		Box:      tview.NewBox(),
		topic:    tview.NewTextView(),
		content:  NewMessageView(),
		status:   tview.NewTextView(),
		userList: tview.NewTextView(),
		ulBorder: NewBorder(),
		input:    NewAdvancedInputField(),
		Room:     room,
	}

	view.input.
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Send a message...").
		SetPlaceholderExtColor(tcell.ColorGray)

	view.topic.
		SetText(strings.Replace(room.GetTopic(), "\n", " ", -1)).
		SetBackgroundColor(tcell.ColorDarkGreen)

	view.status.SetBackgroundColor(tcell.ColorDimGray)

	view.userList.SetDynamicColors(true)

	return view
}

func (view *RoomView) logPath(dir string) string {
	return filepath.Join(dir, fmt.Sprintf("%s.gmxlog", view.Room.ID))
}

func (view *RoomView) SaveHistory(dir string) error {
	return view.MessageView().SaveHistory(view.logPath(dir))
}

func (view *RoomView) LoadHistory(dir string) (int, error) {
	return view.MessageView().LoadHistory(view.logPath(dir))
}

func (view *RoomView) SetTabCompleteFunc(fn func(room *RoomView, text string, cursorOffset int) string) *RoomView {
	view.input.SetTabCompleteFunc(func(text string, cursorOffset int) string {
		return fn(view, text, cursorOffset)
	})
	return view
}

func (view *RoomView) SetInputCapture(fn func(room *RoomView, event *tcell.EventKey) *tcell.EventKey) *RoomView {
	view.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return fn(view, event)
	})
	return view
}

func (view *RoomView) SetInputSubmitFunc(fn func(room *RoomView, text string)) *RoomView {
	view.input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			fn(view, view.input.GetText())
		}
	})
	return view
}

func (view *RoomView) SetInputChangedFunc(fn func(room *RoomView, text string)) *RoomView {
	view.input.SetChangedFunc(func(text string) {
		fn(view, text)
	})
	return view
}

func (view *RoomView) SetInputText(newText string) *RoomView {
	view.input.SetText(newText)
	return view
}

func (view *RoomView) GetInputText() string {
	return view.input.GetText()
}

func (view *RoomView) GetInputField() *AdvancedInputField {
	return view.input
}

func (view *RoomView) Focus(delegate func(p tview.Primitive)) {
	delegate(view.input)
}

// Constants defining the size of the room view grid.
const (
	UserListBorderWidth   = 1
	UserListWidth         = 20
	StaticHorizontalSpace = UserListBorderWidth + UserListWidth

	TopicBarHeight      = 1
	StatusBarHeight     = 1
	InputBarHeight      = 1
	StaticVerticalSpace = TopicBarHeight + StatusBarHeight + InputBarHeight
)

func (view *RoomView) Draw(screen tcell.Screen) {
	x, y, width, height := view.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	// Calculate actual grid based on view rectangle and constants defined above.
	var (
		contentHeight = height - StaticVerticalSpace
		contentWidth  = width - StaticHorizontalSpace

		userListBorderColumn = x + contentWidth
		userListColumn       = userListBorderColumn + UserListBorderWidth

		topicRow   = y
		contentRow = topicRow + TopicBarHeight
		statusRow  = contentRow + contentHeight
		inputRow   = statusRow + StatusBarHeight
	)

	// Update the rectangles of all the children.
	view.topic.SetRect(x, topicRow, width, TopicBarHeight)
	view.content.SetRect(x, contentRow, contentWidth, contentHeight)
	view.status.SetRect(x, statusRow, width, StatusBarHeight)
	if userListColumn > x {
		view.userList.SetRect(userListColumn, contentRow, UserListWidth, contentHeight)
		view.ulBorder.SetRect(userListBorderColumn, contentRow, UserListBorderWidth, contentHeight)
	}
	view.input.SetRect(x, inputRow, width, InputBarHeight)

	// Draw everything
	view.Box.Draw(screen)
	view.topic.Draw(screen)
	view.content.Draw(screen)
	view.status.Draw(screen)
	view.input.Draw(screen)
	view.ulBorder.Draw(screen)
	view.userList.Draw(screen)
}

func (view *RoomView) SetStatus(status string) {
	view.status.SetText(status)
}

func (view *RoomView) SetTyping(users []string) {
	for index, user := range users {
		member := view.Room.GetMember(user)
		if member != nil {
			users[index] = member.DisplayName
		}
	}
	if len(users) == 0 {
		view.status.SetText("")
	} else if len(users) < 2 {
		view.status.SetText("Typing: " + strings.Join(users, " and "))
	} else {
		view.status.SetText(fmt.Sprintf(
			"Typing: %s and %s",
			strings.Join(users[:len(users)-1], ", "), users[len(users)-1]))
	}
}

func (view *RoomView) AutocompleteUser(existingText string) (completions []string) {
	textWithoutPrefix := existingText
	if strings.HasPrefix(existingText, "@") {
		textWithoutPrefix = existingText[1:]
	}
	for _, user := range view.Room.GetMembers() {
		if strings.HasPrefix(user.DisplayName, textWithoutPrefix) {
			completions = append(completions, user.DisplayName)
		} else if strings.HasPrefix(user.UserID, existingText) {
			completions = append(completions, user.UserID)
		}
	}
	return
}

func (view *RoomView) MessageView() *MessageView {
	return view.content
}

func (view *RoomView) UpdateUserList() {
	var joined strings.Builder
	var invited strings.Builder
	for _, user := range view.Room.GetMembers() {
		if user.Membership == "join" {
			joined.WriteString(AddHashColor(user.DisplayName))
			joined.WriteRune('\n')
		} else if user.Membership == "invite" {
			invited.WriteString(AddHashColor(user.DisplayName))
			invited.WriteRune('\n')
		}
	}
	view.userList.Clear()
	fmt.Fprintf(view.userList, "%s\n", joined.String())
	if invited.Len() > 0 {
		fmt.Fprintf(view.userList, "\nInvited:\n%s", invited.String())
	}
}

func (view *RoomView) NewMessage(id, sender, msgtype, text string, timestamp time.Time) *types.Message {
	member := view.Room.GetMember(sender)
	if member != nil {
		sender = member.DisplayName
	}
	return view.content.NewMessage(id, sender, msgtype, text, timestamp)
}

func (view *RoomView) NewTempMessage(msgtype, text string) *types.Message {
	now := time.Now()
	id := strconv.FormatInt(now.UnixNano(), 10)
	sender := ""
	if ownerMember := view.Room.GetSessionOwner(); ownerMember != nil {
		sender = ownerMember.DisplayName
	}
	message := view.NewMessage(id, sender, msgtype, text, now)
	message.State = types.MessageStateSending
	view.AddMessage(message, AppendMessage)
	return message
}

func (view *RoomView) AddMessage(message *types.Message, direction MessageDirection) {
	view.content.AddMessage(message, direction)
}
