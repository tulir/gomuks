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
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kyokomi/emoji"
	"github.com/mattn/go-runewidth"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/matrix/event"

	"maunium.net/go/mauview"

	"maunium.net/go/mautrix"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/util"
	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/widget"
)

type RoomView struct {
	topic    *mauview.TextView
	content  *MessageView
	status   *mauview.TextField
	userList *MemberList
	ulBorder *widget.Border
	input    *mauview.InputArea
	Room     *rooms.Room

	topicScreen    *mauview.ProxyScreen
	contentScreen  *mauview.ProxyScreen
	statusScreen   *mauview.ProxyScreen
	inputScreen    *mauview.ProxyScreen
	ulBorderScreen *mauview.ProxyScreen
	ulScreen       *mauview.ProxyScreen

	userListLoaded bool

	prevScreen mauview.Screen

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
		topic:    mauview.NewTextView(),
		status:   mauview.NewTextField(),
		userList: NewMemberList(),
		ulBorder: widget.NewBorder(),
		input:    mauview.NewInputArea(),
		Room:     room,

		topicScreen:    &mauview.ProxyScreen{OffsetX: 0, OffsetY: 0, Height: TopicBarHeight},
		contentScreen:  &mauview.ProxyScreen{OffsetX: 0, OffsetY: StatusBarHeight},
		statusScreen:   &mauview.ProxyScreen{OffsetX: 0, Height: StatusBarHeight},
		inputScreen:    &mauview.ProxyScreen{OffsetX: 0},
		ulBorderScreen: &mauview.ProxyScreen{OffsetY: StatusBarHeight, Width: UserListBorderWidth},
		ulScreen:       &mauview.ProxyScreen{OffsetY: StatusBarHeight, Width: UserListWidth},

		parent: parent,
		config: parent.config,
	}
	view.content = NewMessageView(view)
	view.Room.SetPreUnload(func() bool {
		if view.parent.currentRoom == view {
			return false
		}
		view.content.Unload()
		return true
	})
	view.Room.SetPostLoad(view.loadTyping)

	view.input.
		SetBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Send a message...").
		SetPlaceholderTextColor(tcell.ColorGray).
		SetTabCompleteFunc(view.InputTabComplete)

	view.topic.
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkGreen)

	view.status.SetBackgroundColor(tcell.ColorDimGray)

	return view
}

func (view *RoomView) logPath(dir string) string {
	return filepath.Join(dir, fmt.Sprintf("%s.gmxlog", view.Room.ID))
}

func (view *RoomView) SetInputChangedFunc(fn func(room *RoomView, text string)) *RoomView {
	view.input.SetChangedFunc(func(text string) {
		fn(view, text)
	})
	return view
}

func (view *RoomView) SetInputText(newText string) *RoomView {
	view.input.SetTextAndMoveCursor(newText)
	return view
}

func (view *RoomView) GetInputText() string {
	return view.input.GetText()
}

func (view *RoomView) Focus() {
	view.input.Focus()
}

func (view *RoomView) Blur() {
	view.input.Blur()
}

func (view *RoomView) GetStatus() string {
	var buf strings.Builder

	if len(view.completions.list) > 0 {
		if view.completions.textCache != view.input.GetText() || view.completions.time.Add(10 * time.Second).Before(time.Now()) {
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
		_, _ = fmt.Fprintf(&buf,
			"Typing: %s and %s - ",
			strings.Join(view.typing[:len(view.typing)-1], ", "), view.typing[len(view.typing)-1])
	}

	return strings.TrimSuffix(buf.String(), " - ")
}

// Constants defining the size of the room view grid.
const (
	UserListBorderWidth   = 1
	UserListWidth         = 20
	StaticHorizontalSpace = UserListBorderWidth + UserListWidth

	TopicBarHeight  = 1
	StatusBarHeight = 1

	MaxInputHeight = 5
)

func (view *RoomView) Draw(screen mauview.Screen) {
	width, height := screen.Size()
	if width <= 0 || height <= 0 {
		return
	}

	if view.prevScreen != screen {
		view.topicScreen.Parent = screen
		view.contentScreen.Parent = screen
		view.statusScreen.Parent = screen
		view.inputScreen.Parent = screen
		view.ulBorderScreen.Parent = screen
		view.ulScreen.Parent = screen
		view.prevScreen = screen
	}

	view.input.PrepareDraw(width)
	inputHeight := view.input.GetTextHeight()
	if inputHeight > MaxInputHeight {
		inputHeight = MaxInputHeight
	} else if inputHeight < 1 {
		inputHeight = 1
	}
	contentHeight := height - inputHeight - TopicBarHeight - StatusBarHeight
	contentWidth := width - StaticHorizontalSpace
	if view.config.Preferences.HideUserList {
		contentWidth = width
	}

	view.topicScreen.Width = width
	view.contentScreen.Width = contentWidth
	view.contentScreen.Height = contentHeight
	view.statusScreen.OffsetY = view.contentScreen.YEnd()
	view.statusScreen.Width = width
	view.inputScreen.Width = width
	view.inputScreen.OffsetY = view.statusScreen.YEnd()
	view.inputScreen.Height = inputHeight
	view.ulBorderScreen.OffsetX = view.contentScreen.XEnd()
	view.ulBorderScreen.Height = contentHeight
	view.ulScreen.OffsetX = view.ulBorderScreen.XEnd()
	view.ulScreen.Height = contentHeight

	// Draw everything
	view.topic.Draw(view.topicScreen)
	view.content.Draw(view.contentScreen)
	view.status.SetText(view.GetStatus())
	view.status.Draw(view.statusScreen)
	view.input.Draw(view.inputScreen)
	if !view.config.Preferences.HideUserList {
		view.ulBorder.Draw(view.ulBorderScreen)
		view.userList.Draw(view.ulScreen)
	}
}

func (view *RoomView) OnKeyEvent(event mauview.KeyEvent) bool {
	msgView := view.MessageView()
	switch event.Key() {
	case tcell.KeyPgUp:
		if msgView.IsAtTop() {
			go view.parent.LoadHistory(view.Room.ID)
		}
		msgView.AddScrollOffset(+msgView.Height() / 2)
		return true
	case tcell.KeyPgDn:
		msgView.AddScrollOffset(-msgView.Height() / 2)
		return true
	case tcell.KeyEnter:
		if event.Modifiers()&tcell.ModShift == 0 && event.Modifiers()&tcell.ModCtrl == 0 {
			view.InputSubmit(view.input.GetText())
			return true
		}
	}
	return view.input.OnKeyEvent(event)
}

func (view *RoomView) OnPasteEvent(event mauview.PasteEvent) bool {
	return view.input.OnPasteEvent(event)
}

func (view *RoomView) OnMouseEvent(event mauview.MouseEvent) bool {
	switch {
	case view.contentScreen.IsInArea(event.Position()):
		return view.content.OnMouseEvent(view.contentScreen.OffsetMouseEvent(event))
	case view.topicScreen.IsInArea(event.Position()):
		return view.topic.OnMouseEvent(view.topicScreen.OffsetMouseEvent(event))
	case view.inputScreen.IsInArea(event.Position()):
		return view.input.OnMouseEvent(view.inputScreen.OffsetMouseEvent(event))
	}
	return false
}

func (view *RoomView) SetCompletions(completions []string) {
	view.completions.list = completions
	view.completions.textCache = view.input.GetText()
	view.completions.time = time.Now()
}

func (view *RoomView) loadTyping() {
	for index, user := range view.typing {
		member := view.Room.GetMember(user)
		if member != nil {
			view.typing[index] = member.Displayname
		}
	}
}

func (view *RoomView) SetTyping(users []string) {
	view.typing = users
	if view.Room.Loaded() {
		view.loadTyping()
	}
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

func (view *RoomView) autocompleteEmoji(word string) (completions []string) {
	if len(word) == 0 || word[0] != ':' {
		return
	}
	var valueCompletion1 string
	var manyValues bool
	for name, value := range emoji.CodeMap() {
		if name == word {
			return []string{value}
		} else if strings.HasPrefix(name, word) {
			completions = append(completions, name)
			if valueCompletion1 == "" {
				valueCompletion1 = value
			} else if valueCompletion1 != value {
				manyValues = true
			}
		}
	}
	if !manyValues && len(completions) > 0 {
		return []string{emoji.CodeMap()[completions[0]]}
	}
	return
}

func (view *RoomView) InputTabComplete(text string, cursorOffset int) {
	debug.Print("Tab completing", cursorOffset, text)
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

	strCompletions = append(strCompletions, view.autocompleteEmoji(word)...)

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

func (view *RoomView) InputSubmit(text string) {
	if len(text) == 0 {
		return
	} else if cmd := view.parent.cmdProcessor.ParseCommand(view, text); cmd != nil {
		go view.parent.cmdProcessor.HandleCommand(cmd)
	} else {
		go view.SendMessage(mautrix.MsgText, text)
	}
	view.SetInputText("")
}

func (view *RoomView) SendMessage(msgtype mautrix.MessageType, text string) {
	defer debug.Recover()
	debug.Print("Sending message", msgtype, text, "to", view.Room.ID)
	if !view.config.Preferences.DisableEmojis {
		text = emoji.Sprint(text)
	}
	evt := view.parent.matrix.PrepareMarkdownMessage(view.Room.ID, msgtype, text)
	msg := view.parseEvent(evt)
	view.AddMessage(msg)
	eventID, err := view.parent.matrix.SendEvent(evt)
	if err != nil {
		msg.State = event.StateSendFail
		// Show shorter version if available
		if httpErr, ok := err.(mautrix.HTTPError); ok {
			err = httpErr
			if respErr := httpErr.RespError; respErr != nil {
				err = respErr
			}
		}
		view.AddServiceMessage(fmt.Sprintf("Failed to send message: %v", err))
		view.parent.parent.Render()
	} else {
		debug.Print("Event ID received:", eventID)
		msg.EventID = eventID
		msg.State = event.StateDefault
		view.MessageView().setMessageID(msg)
		view.parent.parent.Render()
	}
}

func (view *RoomView) MessageView() *MessageView {
	return view.content
}

func (view *RoomView) MxRoom() *rooms.Room {
	return view.Room
}

func (view *RoomView) Update() {
	view.topic.SetText(strings.Replace(view.Room.GetTopic(), "\n", " ", -1))
	if !view.userListLoaded {
		view.UpdateUserList()
	}
}

func (view *RoomView) UpdateUserList() {
	pls := &mautrix.PowerLevels{}
	if plEvent := view.Room.GetStateEvent(mautrix.StatePowerLevels, ""); plEvent != nil {
		pls = plEvent.Content.GetPowerLevels()
	}
	view.userList.Update(view.Room.GetMembers(), pls)
	view.userListLoaded = true
}

func (view *RoomView) AddServiceMessage(text string) {
	view.content.AddMessage(messages.NewServiceMessage(text), AppendMessage)
}

func (view *RoomView) AddMessage(message ifc.Message) {
	view.content.AddMessage(message, AppendMessage)
}

func (view *RoomView) parseEvent(evt *event.Event) *messages.UIMessage {
	return messages.ParseEvent(view.parent.matrix, view.parent, view.Room, evt)
}

func (view *RoomView) ParseEvent(evt *event.Event) ifc.Message {
	msg := view.parseEvent(evt)
	if msg == nil {
		return nil
	}
	return msg
}

func (view *RoomView) GetEvent(eventID string) ifc.Message {
	message, ok := view.content.messageIDs[eventID]
	if !ok {
		return nil
	}
	return message
}
