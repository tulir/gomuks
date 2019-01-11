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
	"maunium.net/go/mautrix"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/util"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

type RoomView struct {
	*tview.Box

	topic    *tview.TextView
	content  *MessageView
	status   *tview.TextView
	userList *tview.TextView
	ulBorder *widget.Border
	input    *widget.AdvancedInputField
	Room     *rooms.Room

	parent *MainView
	config *config.Config

	typing []string

	completions struct {
		list      []string
		textCache string
		time      time.Time
	}
}

func NewRoomView(parent *MainView, room *rooms.Room) *RoomView {
	view := &RoomView{
		Box:      tview.NewBox(),
		topic:    tview.NewTextView(),
		status:   tview.NewTextView(),
		userList: tview.NewTextView(),
		ulBorder: widget.NewBorder(),
		input:    widget.NewAdvancedInputField(),
		Room:     room,
		parent:   parent,
		config:   parent.config,
	}
	view.content = NewMessageView(view)

	view.input.
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Send a message...").
		SetPlaceholderExtColor(tcell.ColorGray).
		SetTabCompleteFunc(view.InputTabComplete)

	view.topic.
		SetText(strings.Replace(room.GetTopic(), "\n", " ", -1)).
		SetBackgroundColor(tcell.ColorDarkGreen)

	view.status.SetBackgroundColor(tcell.ColorDimGray)

	view.userList.
		SetDynamicColors(true).
		SetWrap(false)

	return view
}

func (view *RoomView) logPath(dir string) string {
	return filepath.Join(dir, fmt.Sprintf("%s.gmxlog", view.Room.ID))
}

func (view *RoomView) SaveHistory(dir string) error {
	return view.MessageView().SaveHistory(view.logPath(dir))
}

func (view *RoomView) LoadHistory(matrix ifc.MatrixContainer, dir string) (int, error) {
	return view.MessageView().LoadHistory(matrix, view.logPath(dir))
}

func (view *RoomView) SetInputCapture(fn func(room *RoomView, event *tcell.EventKey) *tcell.EventKey) *RoomView {
	view.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return fn(view, event)
	})
	return view
}

func (view *RoomView) SetMouseCapture(fn func(room *RoomView, event *tcell.EventMouse) *tcell.EventMouse) *RoomView {
	view.input.SetMouseCapture(func(event *tcell.EventMouse) *tcell.EventMouse {
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

func (view *RoomView) GetInputField() *widget.AdvancedInputField {
	return view.input
}

func (view *RoomView) Focus(delegate func(p tview.Primitive)) {
	delegate(view.input)
}

func (view *RoomView) GetStatus() string {
	var buf strings.Builder

	if len(view.completions.list) > 0 {
		if view.completions.textCache != view.input.GetText() || view.completions.time.Add(10*time.Second).Before(time.Now()) {
			view.completions.list = []string{}
		} else {
			buf.WriteString(strings.Join(view.completions.list, ", "))
			buf.WriteString(" - ")
		}
	}

	if len(view.typing) == 1 {
		buf.WriteString("Typing: " + view.typing[0])
		buf.WriteString(" - ")
	} else if len(view.typing) > 1 {
		fmt.Fprintf(&buf,
			"Typing: %s and %s - ",
			strings.Join(view.typing[:len(view.typing)-1], ", "), view.typing[len(view.typing)-1])
	}

	return strings.TrimSuffix(buf.String(), " - ")
}

func (view *RoomView) Draw(screen tcell.Screen) {
	x, y, width, height := view.GetRect()
	if width <= 0 || height <= 0 {
		return
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
	if view.config.Preferences.HideUserList {
		contentWidth = width
	}

	// Update the rectangles of all the children.
	view.topic.SetRect(x, topicRow, width, TopicBarHeight)
	view.content.SetRect(x, contentRow, contentWidth, contentHeight)
	view.status.SetRect(x, statusRow, width, StatusBarHeight)
	if !view.config.Preferences.HideUserList && userListColumn > x {
		view.userList.SetRect(userListColumn, contentRow, UserListWidth, contentHeight)
		view.ulBorder.SetRect(userListBorderColumn, contentRow, UserListBorderWidth, contentHeight)
	}
	view.input.SetRect(x, inputRow, width, InputBarHeight)

	// Draw everything
	view.Box.Draw(screen)
	view.topic.Draw(screen)
	view.content.Draw(screen)
	view.status.SetText(view.GetStatus())
	view.status.Draw(screen)
	view.input.Draw(screen)
	if !view.config.Preferences.HideUserList {
		view.ulBorder.Draw(screen)
		view.userList.Draw(screen)
	}
}

func (view *RoomView) SetCompletions(completions []string) {
	view.completions.list = completions
	view.completions.textCache = view.input.GetText()
	view.completions.time = time.Now()
}

func (view *RoomView) SetTyping(users []string) {
	for index, user := range users {
		member := view.Room.GetMember(user)
		if member != nil {
			users[index] = member.Displayname
		}
	}
	view.typing = users
}

type completion struct {
	displayName string
	id          string
}

func (view *RoomView) autocompleteUser(existingText string) (completions []completion) {
	textWithoutPrefix := strings.TrimPrefix(existingText, "@")
	for userID, user := range view.Room.GetMembers() {
		if user.Displayname == textWithoutPrefix || userID == existingText {
			// Exact match, return that.
			return []completion{{user.Displayname, userID}}
		}

		if strings.HasPrefix(user.Displayname, textWithoutPrefix) || strings.HasPrefix(userID, existingText) {
			completions = append(completions, completion{user.Displayname, userID})
		}
	}
	return
}

func (view *RoomView) autocompleteRoom(existingText string) (completions []completion) {
	for _, room := range view.parent.rooms {
		alias := room.Room.GetCanonicalAlias()
		if alias == existingText {
			// Exact match, return that.
			return []completion{{alias, room.Room.ID}}
		}
		if strings.HasPrefix(alias, existingText) {
			completions = append(completions, completion{alias, room.Room.ID})
			continue
		}
	}
	return
}

func (view *RoomView) InputTabComplete(text string, cursorOffset int) {
	str := runewidth.Truncate(text, cursorOffset, "")
	word := findWordToTabComplete(str)
	startIndex := len(str) - len(word)

	var strCompletions []string
	var strCompletion string

	completions := view.autocompleteUser(word)
	completions = append(completions, view.autocompleteRoom(word)...)

	if len(completions) == 1 {
		completion := completions[0]
		strCompletion = fmt.Sprintf("[%s](https://matrix.to/#/%s)", completion.displayName, completion.id)
		if startIndex == 0 {
			strCompletion = strCompletion + ": "
		}
	} else if len(completions) > 1 {
		for _, completion := range completions {
			strCompletions = append(strCompletions, completion.displayName)
		}
	}

	if len(strCompletions) > 0 {
		strCompletion = util.LongestCommonPrefix(strCompletions)
		sort.Sort(sort.StringSlice(strCompletions))
	}

	if len(strCompletion) > 0 {
		text = str[0:startIndex] + strCompletion + text[len(str):]
	}

	view.input.SetTextAndMoveCursor(text)
	view.SetCompletions(strCompletions)
}

func (view *RoomView) MessageView() *MessageView {
	return view.content
}

func (view *RoomView) MxRoom() *rooms.Room {
	return view.Room
}

func (view *RoomView) UpdateUserList() {
	var joined strings.Builder
	var invited strings.Builder
	for userID, user := range view.Room.GetMembers() {
		if user.Membership == "join" {
			joined.WriteString(widget.AddColor(user.Displayname, widget.GetHashColorName(userID)))
			joined.WriteRune('\n')
		} else if user.Membership == "invite" {
			invited.WriteString(widget.AddColor(user.Displayname, widget.GetHashColorName(userID)))
			invited.WriteRune('\n')
		}
	}
	view.userList.Clear()
	fmt.Fprintf(view.userList, "%s\n", joined.String())
	if invited.Len() > 0 {
		fmt.Fprintf(view.userList, "\nInvited:\n%s", invited.String())
	}
}

func (view *RoomView) newUIMessage(id, sender string, msgtype mautrix.MessageType, text string, timestamp time.Time) messages.UIMessage {
	member := view.Room.GetMember(sender)
	displayname := sender
	if member != nil {
		displayname = member.Displayname
	}
	msg := messages.NewTextMessage(id, sender, displayname, msgtype, text, timestamp)
	return msg
}

func (view *RoomView) NewTempMessage(msgtype mautrix.MessageType, text string) ifc.Message {
	now := time.Now()
	id := strconv.FormatInt(now.UnixNano(), 10)
	sender := ""
	if ownerMember := view.Room.GetMember(view.Room.GetSessionOwner()); ownerMember != nil {
		sender = ownerMember.Displayname
	}
	message := view.newUIMessage(id, sender, msgtype, text, now)
	message.SetState(ifc.MessageStateSending)
	view.AddMessage(message, ifc.AppendMessage)
	return message
}

func (view *RoomView) AddServiceMessage(text string) {
	message := view.newUIMessage(view.parent.matrix.Client().TxnID(), "*", "gomuks.service", text, time.Now())
	message.SetIsService(true)
	view.AddMessage(message, ifc.AppendMessage)
}

func (view *RoomView) AddMessage(message ifc.Message, direction ifc.MessageDirection) {
	view.content.AddMessage(message, direction)
}
