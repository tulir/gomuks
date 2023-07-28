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
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/kyokomi/emoji/v2"
	"github.com/mattn/go-runewidth"
	"github.com/zyedidia/clipboard"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util/variationselector"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/open"
	"maunium.net/go/gomuks/lib/util"
	"maunium.net/go/gomuks/matrix/muksevt"
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

	selecting     bool
	selectReason  SelectReason
	selectContent string

	replying *muksevt.Event

	editing      *muksevt.Event
	editMoveText string

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
		SetTextColor(tcell.ColorDefault).
		SetBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Send a message...").
		SetPlaceholderTextColor(tcell.ColorGray).
		SetTabCompleteFunc(view.InputTabComplete).
		SetPressKeyUpAtStartFunc(view.EditPrevious).
		SetPressKeyDownAtEndFunc(view.EditNext)

	if room.Encrypted {
		view.input.SetPlaceholder("Send an encrypted message...")
	}

	view.topic.
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkGreen)

	view.status.SetBackgroundColor(tcell.ColorDimGray)

	return view
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
	view.StopSelecting()
	view.input.Blur()
}

func (view *RoomView) StartSelecting(reason SelectReason, content string) {
	view.selecting = true
	view.selectReason = reason
	view.selectContent = content
	msgView := view.MessageView()
	if msgView.selected != nil {
		view.OnSelect(msgView.selected)
	} else {
		view.input.Blur()
		view.SelectPrevious()
	}
}

func (view *RoomView) StopSelecting() {
	view.selecting = false
	view.selectContent = ""
	view.MessageView().SetSelected(nil)
}

func (view *RoomView) OnSelect(message *messages.UIMessage) {
	if !view.selecting || message == nil {
		return
	}
	switch view.selectReason {
	case SelectReply:
		view.replying = message.Event
		if len(view.selectContent) > 0 {
			go view.SendMessage(event.MsgText, view.selectContent)
		}
	case SelectEdit:
		view.SetEditing(message.Event)
	case SelectReact:
		go view.SendReaction(message.EventID, view.selectContent)
	case SelectRedact:
		go view.Redact(message.EventID, view.selectContent)
	case SelectDownload, SelectOpen:
		msg, ok := message.Renderer.(*messages.FileMessage)
		if ok {
			path := ""
			if len(view.selectContent) > 0 {
				path = view.selectContent
			} else if view.selectReason == SelectDownload {
				path = msg.Body
			}
			go view.Download(msg.URL, msg.File, path, view.selectReason == SelectOpen)
		}
	case SelectCopy:
		go view.CopyToClipboard(message.Renderer.PlainText(), view.selectContent)
	}
	view.selecting = false
	view.selectContent = ""
	view.MessageView().SetSelected(nil)
	view.input.Focus()
}

func (view *RoomView) GetStatus() string {
	var buf strings.Builder

	if view.editing != nil {
		buf.WriteString("Editing message - ")
	} else if view.replying != nil {
		buf.WriteString("Replying to ")
		buf.WriteString(string(view.replying.Sender))
		buf.WriteString(" - ")
	} else if view.selecting {
		buf.WriteString("Selecting message to ")
		buf.WriteString(string(view.selectReason))
		buf.WriteString(" - ")
	}

	if len(view.completions.list) > 0 {
		if view.completions.textCache != view.input.GetText() || view.completions.time.Add(10*time.Second).Before(time.Now()) {
			view.completions.list = []string{}
		} else {
			buf.WriteString(strings.Join(view.completions.list, ", "))
			buf.WriteString(" - ")
		}
	}

	if len(view.typing) == 1 {
		buf.WriteString("Typing: " + string(view.typing[0]))
		buf.WriteString(" - ")
	} else if len(view.typing) > 1 {
		buf.WriteString("Typing: ")
		for i, userID := range view.typing {
			if i == len(view.typing)-1 {
				buf.WriteString(" and ")
			} else if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(string(userID))
		}
		buf.WriteString(" - ")
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

func (view *RoomView) ClearAllContext() {
	view.SetEditing(nil)
	view.StopSelecting()
	view.replying = nil
	view.input.Focus()
}

func (view *RoomView) OnKeyEvent(event mauview.KeyEvent) bool {
	msgView := view.MessageView()
	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}

	if view.selecting {
		switch view.config.Keybindings.Visual[kb] {
		case "clear":
			view.ClearAllContext()
		case "select_prev":
			view.SelectPrevious()
		case "select_next":
			view.SelectNext()
		case "confirm":
			view.OnSelect(msgView.selected)
		default:
			return false
		}
		return true
	}

	switch view.config.Keybindings.Room[kb] {
	case "clear":
		view.ClearAllContext()
		return true
	case "scroll_up":
		if msgView.IsAtTop() {
			go view.parent.LoadHistory(view.Room.ID)
		}
		msgView.AddScrollOffset(+msgView.Height() / 2)
		return true
	case "scroll_down":
		msgView.AddScrollOffset(-msgView.Height() / 2)
		return true
	case "send":
		view.InputSubmit(view.input.GetText())
		return true
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
		member := view.Room.GetMember(id.UserID(user))
		if member != nil {
			view.typing[index] = member.Displayname
		}
	}
}

func (view *RoomView) SetTyping(users []id.UserID) {
	view.typing = make([]string, len(users))
	for i, user := range users {
		view.typing[i] = string(user)
	}
	if view.Room.Loaded() {
		view.loadTyping()
	}
}

var editHTMLParser = &format.HTMLParser{
	PillConverter: func(displayname, mxid, eventID string, ctx format.Context) string {
		if len(eventID) > 0 {
			return fmt.Sprintf(`[%s](https://matrix.to/#/%s/%s)`, displayname, mxid, eventID)
		} else {
			return fmt.Sprintf(`[%s](https://matrix.to/#/%s)`, displayname, mxid)
		}
	},
	Newline:        "\n",
	HorizontalLine: "\n---\n",
}

func (view *RoomView) SetEditing(evt *muksevt.Event) {
	if evt == nil {
		view.editing = nil
		view.SetInputText(view.editMoveText)
		view.editMoveText = ""
	} else {
		if view.editing == nil {
			view.editMoveText = view.GetInputText()
		}
		view.editing = evt
		// replying should never be non-nil when SetEditing, but do this just to be safe
		view.replying = nil
		msgContent := view.editing.Content.AsMessage()
		if len(view.editing.Gomuks.Edits) > 0 {
			// This feels kind of dangerous, but I think it works
			msgContent = view.editing.Gomuks.Edits[len(view.editing.Gomuks.Edits)-1].Content.AsMessage().NewContent
		}
		text := msgContent.Body
		if len(msgContent.FormattedBody) > 0 && (!view.config.Preferences.DisableMarkdown || !view.config.Preferences.DisableHTML) {
			if view.config.Preferences.DisableMarkdown {
				text = msgContent.FormattedBody
			} else {
				text = editHTMLParser.Parse(msgContent.FormattedBody, make(format.Context))
			}
		}
		if msgContent.MsgType == event.MsgEmote {
			text = "/me " + text
		}
		view.input.SetText(text)
	}
	view.status.SetText(view.GetStatus())
	view.input.SetCursorOffset(-1)
}

type findFilter func(evt *muksevt.Event) bool

func (view *RoomView) filterOwnOnly(evt *muksevt.Event) bool {
	return evt.Sender == view.parent.matrix.Client().UserID && evt.Type == event.EventMessage
}

func (view *RoomView) filterMediaOnly(evt *muksevt.Event) bool {
	content, ok := evt.Content.Parsed.(*event.MessageEventContent)
	return ok && (content.MsgType == event.MsgFile ||
		content.MsgType == event.MsgImage ||
		content.MsgType == event.MsgAudio ||
		content.MsgType == event.MsgVideo)
}

func (view *RoomView) findMessage(current *muksevt.Event, forward bool, allow findFilter) *messages.UIMessage {
	currentFound := current == nil
	msgs := view.MessageView().messages
	for i := 0; i < len(msgs); i++ {
		index := i
		if !forward {
			index = len(msgs) - i - 1
		}
		evt := msgs[index]
		if evt.EventID == "" || string(evt.EventID) == evt.TxnID || evt.IsService {
			continue
		} else if currentFound {
			if allow == nil || allow(evt.Event) {
				return evt
			}
		} else if evt.EventID == current.ID {
			currentFound = true
		}
	}
	return nil
}

func (view *RoomView) EditNext() {
	if view.editing == nil {
		return
	}
	foundMsg := view.findMessage(view.editing, true, view.filterOwnOnly)
	view.SetEditing(foundMsg.GetEvent())
}

func (view *RoomView) EditPrevious() {
	if view.replying != nil {
		return
	}
	foundMsg := view.findMessage(view.editing, false, view.filterOwnOnly)
	if foundMsg != nil {
		view.SetEditing(foundMsg.GetEvent())
	}
}

func (view *RoomView) SelectNext() {
	msgView := view.MessageView()
	if msgView.selected == nil {
		return
	}
	var filter findFilter
	if view.selectReason == SelectDownload || view.selectReason == SelectOpen {
		filter = view.filterMediaOnly
	}
	foundMsg := view.findMessage(msgView.selected.GetEvent(), true, filter)
	if foundMsg != nil {
		msgView.SetSelected(foundMsg)
		// TODO scroll selected message into view
	}
}

func (view *RoomView) SelectPrevious() {
	msgView := view.MessageView()
	var filter findFilter
	if view.selectReason == SelectDownload || view.selectReason == SelectOpen {
		filter = view.filterMediaOnly
	}
	foundMsg := view.findMessage(msgView.selected.GetEvent(), false, filter)
	if foundMsg != nil {
		msgView.SetSelected(foundMsg)
		// TODO scroll selected message into view
	}
}

type completion struct {
	displayName string
	id          string
}

func (view *RoomView) AutocompleteUser(existingText string) (completions []completion) {
	textWithoutPrefix := strings.TrimPrefix(existingText, "@")
	for userID, user := range view.Room.GetMembers() {
		if user.Displayname == textWithoutPrefix || string(userID) == existingText {
			// Exact match, return that.
			return []completion{{user.Displayname, string(userID)}}
		}

		if strings.HasPrefix(user.Displayname, textWithoutPrefix) || strings.HasPrefix(string(userID), existingText) {
			completions = append(completions, completion{user.Displayname, string(userID)})
		}
	}
	return
}

func (view *RoomView) AutocompleteRoom(existingText string) (completions []completion) {
	for _, room := range view.parent.rooms {
		alias := string(room.Room.GetCanonicalAlias())
		if alias == existingText {
			// Exact match, return that.
			return []completion{{alias, string(room.Room.ID)}}
		}
		if strings.HasPrefix(alias, existingText) {
			completions = append(completions, completion{alias, string(room.Room.ID)})
			continue
		}
	}
	return
}

func (view *RoomView) AutocompleteEmoji(word string) (completions []string) {
	if word[0] != ':' {
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

var (
	mentionMarkdown  = "[%[1]s](https://matrix.to/#/%[2]s)"
	mentionHTML      = `<a href="https://matrix.to/#/%[2]s">%[1]s</a>`
	mentionPlaintext = "%[1]s"
)

func (view *RoomView) defaultAutocomplete(word string, startIndex int) (strCompletions []string, strCompletion string) {
	if len(word) == 0 {
		return []string{}, ""
	}

	completions := view.AutocompleteUser(word)
	completions = append(completions, view.AutocompleteRoom(word)...)

	if len(completions) == 1 {
		completion := completions[0]
		template := mentionMarkdown
		if view.config.Preferences.DisableMarkdown {
			if view.config.Preferences.DisableHTML {
				template = mentionPlaintext
			} else {
				template = mentionHTML
			}
		}
		strCompletion = fmt.Sprintf(template, completion.displayName, completion.id)
		if startIndex == 0 && completion.id[0] == '@' {
			strCompletion = strCompletion + ":"
		}
	} else if len(completions) > 1 {
		for _, completion := range completions {
			strCompletions = append(strCompletions, completion.displayName)
		}
	}

	strCompletions = append(strCompletions, view.parent.cmdProcessor.AutocompleteCommand(word)...)
	strCompletions = append(strCompletions, view.AutocompleteEmoji(word)...)

	return
}

func (view *RoomView) InputTabComplete(text string, cursorOffset int) {
	if len(text) == 0 {
		return
	}

	str := runewidth.Truncate(text, cursorOffset, "")
	word := findWordToTabComplete(str)
	startIndex := len(str) - len(word)

	var strCompletion string

	strCompletions, newText, ok := view.parent.cmdProcessor.Autocomplete(view, text, cursorOffset)
	if !ok {
		strCompletions, strCompletion = view.defaultAutocomplete(word, startIndex)
	}

	if len(strCompletions) > 0 {
		strCompletion = util.LongestCommonPrefix(strCompletions)
		sort.Sort(sort.StringSlice(strCompletions))
	}
	if len(strCompletion) > 0 && len(strCompletions) < 2 {
		strCompletion += " "
		strCompletions = []string{}
	}

	if len(strCompletion) > 0 && newText == text {
		newText = str[0:startIndex] + strCompletion + text[len(str):]
	}

	view.input.SetTextAndMoveCursor(newText)
	view.SetCompletions(strCompletions)
}

func (view *RoomView) InputSubmit(text string) {
	if len(text) == 0 {
		return
	} else if cmd := view.parent.cmdProcessor.ParseCommand(view, text); cmd != nil {
		go view.parent.cmdProcessor.HandleCommand(cmd)
	} else {
		go view.SendMessage(event.MsgText, text)
	}
	view.editMoveText = ""
	view.SetInputText("")
}

func (view *RoomView) CopyToClipboard(text string, register string) {
	if register == "clipboard" || register == "primary" {
		err := clipboard.WriteAll(text, register)
		if err != nil {
			view.AddServiceMessage(fmt.Sprintf("Clipboard unsupported: %v", err))
			view.parent.parent.Render()
		}
	} else {
		view.AddServiceMessage(fmt.Sprintf("Clipboard register %v unsupported", register))
		view.parent.parent.Render()
	}
}

func (view *RoomView) Download(url id.ContentURI, file *attachment.EncryptedFile, filename string, openFile bool) {
	path, err := view.parent.matrix.DownloadToDisk(url, file, filename)
	if err != nil {
		view.AddServiceMessage(fmt.Sprintf("Failed to download media: %v", err))
		view.parent.parent.Render()
		return
	}
	view.AddServiceMessage(fmt.Sprintf("File downloaded to %s", path))
	view.parent.parent.Render()
	if openFile {
		debug.Print("Opening file", path)
		open.Open(path)
	}
}

func (view *RoomView) Redact(eventID id.EventID, reason string) {
	defer debug.Recover()
	err := view.parent.matrix.Redact(view.Room.ID, eventID, reason)
	if err != nil {
		if httpErr, ok := err.(mautrix.HTTPError); ok {
			err = httpErr
			if respErr := httpErr.RespError; respErr != nil {
				err = respErr
			}
		}
		view.AddServiceMessage(fmt.Sprintf("Failed to redact message: %v", err))
		view.parent.parent.Render()
	}
}

func (view *RoomView) SendReaction(eventID id.EventID, reaction string) {
	defer debug.Recover()
	if !view.config.Preferences.DisableEmojis {
		reaction = emoji.Sprint(reaction)
	}
	reaction = variationselector.Add(strings.TrimSpace(reaction))
	debug.Print("Reacting to", eventID, "in", view.Room.ID, "with", reaction)
	eventID, err := view.parent.matrix.SendEvent(&muksevt.Event{
		Event: &event.Event{
			Type:   event.EventReaction,
			RoomID: view.Room.ID,
			Content: event.Content{Parsed: &event.ReactionEventContent{RelatesTo: event.RelatesTo{
				Type:    event.RelAnnotation,
				EventID: eventID,
				Key:     reaction,
			}}},
		},
	})
	if err != nil {
		if httpErr, ok := err.(mautrix.HTTPError); ok {
			err = httpErr
			if respErr := httpErr.RespError; respErr != nil {
				err = respErr
			}
		}
		view.AddServiceMessage(fmt.Sprintf("Failed to send reaction: %v", err))
		view.parent.parent.Render()
	}
}

func (view *RoomView) SendMessage(msgtype event.MessageType, text string) {
	view.SendMessageHTML(msgtype, text, "")
}

func (view *RoomView) getRelationForNewEvent() *ifc.Relation {
	if view.editing != nil {
		return &ifc.Relation{
			Type:  event.RelReplace,
			Event: view.editing,
		}
	} else if view.replying != nil {
		return &ifc.Relation{
			Type:  event.RelReply,
			Event: view.replying,
		}
	}
	return nil
}

func (view *RoomView) SendMessageHTML(msgtype event.MessageType, text, html string) {
	defer debug.Recover()
	debug.Print("Sending message", msgtype, text, "to", view.Room.ID)
	if !view.config.Preferences.DisableEmojis {
		text = emoji.Sprint(text)
	}
	rel := view.getRelationForNewEvent()
	evt := view.parent.matrix.PrepareMarkdownMessage(view.Room.ID, msgtype, text, html, rel)
	view.addLocalEcho(evt)
}

func (view *RoomView) SendMessageMedia(path string) {
	defer debug.Recover()
	debug.Print("Sending media at", path, "to", view.Room.ID)
	rel := view.getRelationForNewEvent()
	evt, err := view.parent.matrix.PrepareMediaMessage(view.Room, path, rel)
	if err != nil {
		view.AddServiceMessage(fmt.Sprintf("Failed to upload media: %v", err))
		view.parent.parent.Render()
		return
	}
	view.addLocalEcho(evt)
}

func (view *RoomView) addLocalEcho(evt *muksevt.Event) {
	msg := view.parseEvent(evt.SomewhatDangerousCopy())
	view.content.AddMessage(msg, AppendMessage)
	view.ClearAllContext()
	view.status.SetText(view.GetStatus())
	eventID, err := view.parent.matrix.SendEvent(evt)
	if err != nil {
		msg.State = muksevt.StateSendFail
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
		msg.State = muksevt.StateDefault
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
	topicStr := strings.TrimSpace(strings.ReplaceAll(view.Room.GetTopic(), "\n", " "))
	if view.config.Preferences.HideRoomList {
		if len(topicStr) > 0 {
			topicStr = fmt.Sprintf("%s - %s", view.Room.GetTitle(), topicStr)
		} else {
			topicStr = view.Room.GetTitle()
		}
		topicStr = strings.TrimSpace(topicStr)
	}
	view.topic.SetText(topicStr)
	if !view.userListLoaded {
		view.UpdateUserList()
	}
}

func (view *RoomView) UpdateUserList() {
	pls := &event.PowerLevelsEventContent{}
	if plEvent := view.Room.GetStateEvent(event.StatePowerLevels, ""); plEvent != nil {
		pls = plEvent.Content.AsPowerLevels()
	}
	view.userList.Update(view.Room.GetMembers(), pls)
	view.userListLoaded = true
}

func (view *RoomView) AddServiceMessage(text string) {
	view.content.AddMessage(messages.NewServiceMessage(text), AppendMessage)
}

func (view *RoomView) parseEvent(evt *muksevt.Event) *messages.UIMessage {
	return messages.ParseEvent(view.parent.matrix, view.parent, view.Room, evt)
}

func (view *RoomView) AddHistoryEvent(evt *muksevt.Event) {
	if msg := view.parseEvent(evt); msg != nil {
		view.content.AddMessage(msg, PrependMessage)
	}
}

func (view *RoomView) AddEvent(evt *muksevt.Event) ifc.Message {
	if msg := view.parseEvent(evt); msg != nil {
		view.content.AddMessage(msg, AppendMessage)
		return msg
	}
	return nil
}

func (view *RoomView) AddRedaction(redactedEvt *muksevt.Event) {
	view.AddEvent(redactedEvt)
}

func (view *RoomView) AddEdit(evt *muksevt.Event) {
	if msg := view.parseEvent(evt); msg != nil {
		view.content.AddMessage(msg, IgnoreMessage)
	}
}

func (view *RoomView) AddReaction(evt *muksevt.Event, key string) {
	msgView := view.MessageView()
	msg := msgView.getMessageByID(evt.ID)
	if msg == nil {
		// Message not in view, nothing to do
		return
	}
	heightChanged := len(msg.Reactions) == 0
	msg.AddReaction(key)
	if heightChanged {
		// Replace buffer to update height of message
		msgView.replaceBuffer(msg, msg)
	}
}

func (view *RoomView) GetEvent(eventID id.EventID) ifc.Message {
	message, ok := view.content.messageIDs[eventID]
	if !ok {
		return nil
	}
	return message
}
