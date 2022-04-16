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
	"math"
	"strings"
	"sync/atomic"

	"github.com/mattn/go-runewidth"
	sync "github.com/sasha-s/go-deadlock"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/open"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/widget"
)

type MessageView struct {
	parent *RoomView
	config *config.Config

	ScrollOffset    int
	MaxSenderWidth  int
	DateFormat      string
	TimestampFormat string
	TimestampWidth  int

	// Used for locking
	loadingMessages int32
	historyLoadPtr  uint64

	_widestSender     uint32
	_prevWidestSender uint32

	_width      uint32
	_height     uint32
	_prevWidth  uint32
	_prevHeight uint32

	prevMsgCount int
	prevPrefs    config.UserPreferences

	messageIDLock sync.RWMutex
	messageIDs    map[id.EventID]*messages.UIMessage
	messagesLock  sync.RWMutex
	messages      []*messages.UIMessage
	msgBufferLock sync.RWMutex
	msgBuffer     []*messages.UIMessage
	selected      *messages.UIMessage

	initialHistoryLoaded bool
}

func NewMessageView(parent *RoomView) *MessageView {
	return &MessageView{
		parent: parent,
		config: parent.config,

		MaxSenderWidth: 15,
		TimestampWidth: len(messages.TimeFormat),
		ScrollOffset:   0,

		messages:   make([]*messages.UIMessage, 0),
		messageIDs: make(map[id.EventID]*messages.UIMessage),
		msgBuffer:  make([]*messages.UIMessage, 0),

		_widestSender:     5,
		_prevWidestSender: 0,

		_width:       80,
		_prevWidth:   0,
		_prevHeight:  0,
		prevMsgCount: -1,
	}
}

func (view *MessageView) Unload() {
	debug.Print("Unloading message view", view.parent.Room.ID)
	view.messagesLock.Lock()
	view.msgBufferLock.Lock()
	view.messageIDLock.Lock()
	view.messageIDs = make(map[id.EventID]*messages.UIMessage)
	view.msgBuffer = make([]*messages.UIMessage, 0)
	view.messages = make([]*messages.UIMessage, 0)
	view.initialHistoryLoaded = false
	view.ScrollOffset = 0
	view._widestSender = 5
	view.prevMsgCount = -1
	view.historyLoadPtr = 0
	view.messagesLock.Unlock()
	view.msgBufferLock.Unlock()
	view.messageIDLock.Unlock()
}

func (view *MessageView) updateWidestSender(sender string) {
	if len(sender) > int(view._widestSender) {
		if len(sender) > view.MaxSenderWidth {
			atomic.StoreUint32(&view._widestSender, uint32(view.MaxSenderWidth))
		} else {
			atomic.StoreUint32(&view._widestSender, uint32(len(sender)))
		}
	}
}

type MessageDirection int

const (
	AppendMessage MessageDirection = iota
	PrependMessage
	IgnoreMessage
)

func (view *MessageView) AddMessage(ifcMessage ifc.Message, direction MessageDirection) {
	if ifcMessage == nil {
		return
	}
	message, ok := ifcMessage.(*messages.UIMessage)
	if !ok || message == nil {
		debug.Print("[Warning] Passed non-UIMessage ifc.Message object to AddMessage().")
		debug.PrintStack()
		return
	}

	var oldMsg *messages.UIMessage
	if oldMsg = view.getMessageByID(message.EventID); oldMsg != nil {
		view.replaceMessage(oldMsg, message)
		direction = IgnoreMessage
	} else if oldMsg = view.getMessageByID(id.EventID(message.TxnID)); oldMsg != nil {
		view.replaceMessage(oldMsg, message)
		view.deleteMessageID(id.EventID(message.TxnID))
		direction = IgnoreMessage
	}

	view.updateWidestSender(message.Sender())

	width := view.width()
	bare := view.config.Preferences.BareMessageView
	if !bare {
		width -= view.widestSender() + SenderMessageGap
		if !view.config.Preferences.HideTimestamp {
			width -= view.TimestampWidth + TimestampSenderGap
		}
	}
	message.CalculateBuffer(view.config.Preferences, width)

	makeDateChange := func(msg *messages.UIMessage) *messages.UIMessage {
		dateChange := messages.NewDateChangeMessage(
			fmt.Sprintf("Date changed to %s", msg.FormatDate()))
		dateChange.CalculateBuffer(view.config.Preferences, width)
		view.appendBuffer(dateChange)
		return dateChange
	}

	if direction == AppendMessage {
		if view.ScrollOffset > 0 {
			view.ScrollOffset += message.Height()
		}
		view.messagesLock.Lock()
		if len(view.messages) > 0 && !view.messages[len(view.messages)-1].SameDate(message) {
			view.messages = append(view.messages, makeDateChange(message), message)
		} else {
			view.messages = append(view.messages, message)
		}
		view.messagesLock.Unlock()
		view.appendBuffer(message)
	} else if direction == PrependMessage {
		view.messagesLock.Lock()
		if len(view.messages) > 0 && !view.messages[0].SameDate(message) {
			view.messages = append([]*messages.UIMessage{message, makeDateChange(view.messages[0])}, view.messages...)
		} else {
			view.messages = append([]*messages.UIMessage{message}, view.messages...)
		}
		view.messagesLock.Unlock()
	} else if oldMsg != nil {
		view.replaceBuffer(oldMsg, message)
	} else {
		debug.Print("Unexpected AddMessage() call: Direction is not append or prepend, but message is new.")
		debug.PrintStack()
	}

	if len(message.ID()) > 0 {
		view.setMessageID(message)
	}
}

func (view *MessageView) replaceMessage(original *messages.UIMessage, new *messages.UIMessage) {
	if len(new.ID()) > 0 {
		view.setMessageID(new)
	}
	view.messagesLock.Lock()
	for index, msg := range view.messages {
		if msg == original {
			view.messages[index] = new
		}
	}
	view.messagesLock.Unlock()
}

func (view *MessageView) getMessageByID(id id.EventID) *messages.UIMessage {
	if id == "" {
		return nil
	}
	view.messageIDLock.RLock()
	defer view.messageIDLock.RUnlock()
	msg, ok := view.messageIDs[id]
	if !ok {
		return nil
	}
	return msg
}

func (view *MessageView) deleteMessageID(id id.EventID) {
	if id == "" {
		return
	}
	view.messageIDLock.Lock()
	delete(view.messageIDs, id)
	view.messageIDLock.Unlock()
}

func (view *MessageView) setMessageID(message *messages.UIMessage) {
	if message.ID() == "" {
		return
	}
	view.messageIDLock.Lock()
	view.messageIDs[message.ID()] = message
	view.messageIDLock.Unlock()
}

func (view *MessageView) appendBuffer(message *messages.UIMessage) {
	view.msgBufferLock.Lock()
	view.appendBufferUnlocked(message)
	view.msgBufferLock.Unlock()
}

func (view *MessageView) appendBufferUnlocked(message *messages.UIMessage) {
	for i := 0; i < message.Height(); i++ {
		view.msgBuffer = append(view.msgBuffer, message)
	}
	view.prevMsgCount++
}

func (view *MessageView) replaceBuffer(original *messages.UIMessage, new *messages.UIMessage) {
	start := -1
	end := -1
	view.msgBufferLock.RLock()
	for index, meta := range view.msgBuffer {
		if meta == original {
			if start == -1 {
				start = index
			}
			end = index
		} else if start != -1 {
			break
		}
	}
	view.msgBufferLock.RUnlock()

	if start == -1 {
		debug.Print("Called replaceBuffer() with message that was not in the buffer:", original)
		//debug.PrintStack()
		view.appendBuffer(new)
		return
	}

	if len(view.msgBuffer) > end {
		end++
	}

	if new.Height() == 0 {
		new.CalculateBuffer(view.prevPrefs, view.prevWidth())
	}

	view.msgBufferLock.Lock()
	if new.Height() != end-start {
		height := new.Height()

		newBuffer := make([]*messages.UIMessage, height+len(view.msgBuffer)-end)
		for i := 0; i < height; i++ {
			newBuffer[i] = new
		}
		for i := height; i < len(newBuffer); i++ {
			newBuffer[i] = view.msgBuffer[end+(i-height)]
		}
		view.msgBuffer = append(view.msgBuffer[0:start], newBuffer...)
	} else {
		for i := start; i < end; i++ {
			view.msgBuffer[i] = new
		}
	}
	view.msgBufferLock.Unlock()
}

func (view *MessageView) recalculateBuffers() {
	prefs := view.config.Preferences
	recalculateMessageBuffers := view.width() != view.prevWidth() ||
		view.widestSender() != view.prevWidestSender() ||
		view.prevPrefs.BareMessageView != prefs.BareMessageView ||
		view.prevPrefs.DisableImages != prefs.DisableImages
	view.messagesLock.RLock()
	view.msgBufferLock.Lock()
	if recalculateMessageBuffers || len(view.messages) != view.prevMsgCount {
		width := view.width()
		if !prefs.BareMessageView {
			width -= view.widestSender() + SenderMessageGap
			if !prefs.HideTimestamp {
				width -= view.TimestampWidth + TimestampSenderGap
			}
		}
		view.msgBuffer = []*messages.UIMessage{}
		view.prevMsgCount = 0
		for i, message := range view.messages {
			if message == nil {
				debug.Print("O.o found nil message at", i)
				break
			}
			if recalculateMessageBuffers {
				message.CalculateBuffer(prefs, width)
			}
			view.appendBufferUnlocked(message)
		}
	}
	view.msgBufferLock.Unlock()
	view.messagesLock.RUnlock()
	view.updatePrevSize()
	view.prevPrefs = prefs
}

func (view *MessageView) SetSelected(message *messages.UIMessage) {
	if view.selected != nil {
		view.selected.IsSelected = false
	}
	if message != nil && (view.selected == message || message.IsService) {
		view.selected = nil
	} else {
		view.selected = message
	}
	if view.selected != nil {
		view.selected.IsSelected = true
	}
}

func (view *MessageView) handleMessageClick(message *messages.UIMessage, mod tcell.ModMask) bool {
	if msg, ok := message.Renderer.(*messages.FileMessage); ok && mod > 0 && !msg.Thumbnail.IsEmpty() {
		debug.Print("Opening thumbnail", msg.ThumbnailPath())
		open.Open(msg.ThumbnailPath())
		// No need to re-render
		return false
	}
	view.SetSelected(message)
	view.parent.OnSelect(view.selected)
	return true
}

func (view *MessageView) handleUsernameClick(message *messages.UIMessage, prevMessage *messages.UIMessage) bool {
	// TODO this is needed if senders are hidden for messages from the same sender (see Draw method)
	//if prevMessage != nil && prevMessage.SenderName == message.SenderName {
	//	return false
	//}

	if message.SenderName == "---" || message.SenderName == "-->" || message.SenderName == "<--" || message.Type == event.MsgEmote {
		return false
	}

	sender := fmt.Sprintf("[%s](https://matrix.to/#/%s)", message.SenderName, message.SenderID)

	cursorPos := view.parent.input.GetCursorOffset()
	text := view.parent.input.GetText()
	var buf strings.Builder
	if cursorPos == 0 {
		buf.WriteString(sender)
		buf.WriteRune(':')
		buf.WriteRune(' ')
		buf.WriteString(text)
	} else {
		textBefore := runewidth.Truncate(text, cursorPos, "")
		textAfter := text[len(textBefore):]
		buf.WriteString(textBefore)
		buf.WriteString(sender)
		buf.WriteRune(' ')
		buf.WriteString(textAfter)
	}
	newText := buf.String()
	view.parent.input.SetText(string(newText))
	view.parent.input.SetCursorOffset(cursorPos + len(newText) - len(text))
	return true
}

func (view *MessageView) OnMouseEvent(event mauview.MouseEvent) bool {
	if event.HasMotion() {
		return false
	}
	switch event.Buttons() {
	case tcell.WheelUp:
		if view.IsAtTop() {
			go view.parent.parent.LoadHistory(view.parent.Room.ID)
		} else {
			view.AddScrollOffset(WheelScrollOffsetDiff)
			return true
		}
	case tcell.WheelDown:
		view.AddScrollOffset(-WheelScrollOffsetDiff)
		view.parent.parent.MarkRead(view.parent)
		return true
	case tcell.Button1:
		x, y := event.Position()
		line := view.TotalHeight() - view.ScrollOffset - view.Height() + y
		if line < 0 || line >= view.TotalHeight() {
			return false
		}

		view.msgBufferLock.RLock()
		message := view.msgBuffer[line]
		var prevMessage *messages.UIMessage
		if y != 0 && line > 0 {
			prevMessage = view.msgBuffer[line-1]
		}
		view.msgBufferLock.RUnlock()

		usernameX := 0
		if !view.config.Preferences.HideTimestamp {
			usernameX += view.TimestampWidth + TimestampSenderGap
		}
		messageX := usernameX + view.widestSender() + SenderMessageGap

		if x >= messageX {
			return view.handleMessageClick(message, event.Modifiers())
		} else if x >= usernameX {
			return view.handleUsernameClick(message, prevMessage)
		}
	}
	return false
}

const PaddingAtTop = 5

func (view *MessageView) AddScrollOffset(diff int) {
	totalHeight := view.TotalHeight()
	height := view.Height()
	if diff >= 0 && view.ScrollOffset+diff >= totalHeight-height+PaddingAtTop {
		view.ScrollOffset = totalHeight - height + PaddingAtTop
	} else {
		view.ScrollOffset += diff
	}

	if view.ScrollOffset > totalHeight-height+PaddingAtTop {
		view.ScrollOffset = totalHeight - height + PaddingAtTop
	}
	if view.ScrollOffset < 0 {
		view.ScrollOffset = 0
	}
}

func (view *MessageView) setSize(width, height int) {
	atomic.StoreUint32(&view._width, uint32(width))
	atomic.StoreUint32(&view._height, uint32(height))
}

func (view *MessageView) updatePrevSize() {
	atomic.StoreUint32(&view._prevWidth, atomic.LoadUint32(&view._width))
	atomic.StoreUint32(&view._prevHeight, atomic.LoadUint32(&view._height))
	atomic.StoreUint32(&view._prevWidestSender, atomic.LoadUint32(&view._widestSender))
}

func (view *MessageView) prevHeight() int {
	return int(atomic.LoadUint32(&view._prevHeight))
}

func (view *MessageView) prevWidth() int {
	return int(atomic.LoadUint32(&view._prevWidth))
}

func (view *MessageView) prevWidestSender() int {
	return int(atomic.LoadUint32(&view._prevWidestSender))
}

func (view *MessageView) widestSender() int {
	return int(atomic.LoadUint32(&view._widestSender))
}

func (view *MessageView) Height() int {
	return int(atomic.LoadUint32(&view._height))
}

func (view *MessageView) width() int {
	return int(atomic.LoadUint32(&view._width))
}

func (view *MessageView) TotalHeight() int {
	view.msgBufferLock.RLock()
	defer view.msgBufferLock.RUnlock()
	return len(view.msgBuffer)
}

func (view *MessageView) IsAtTop() bool {
	return view.ScrollOffset >= view.TotalHeight()-view.Height()+PaddingAtTop
}

const (
	TimestampSenderGap = 1
	SenderSeparatorGap = 1
	SenderMessageGap   = 3
)

func getScrollbarStyle(scrollbarHere, isTop, isBottom bool) (char rune, style tcell.Style) {
	char = '│'
	style = tcell.StyleDefault
	if scrollbarHere {
		style = style.Foreground(tcell.ColorGreen)
	}
	if isTop {
		if scrollbarHere {
			char = '╥'
		} else {
			char = '┬'
		}
	} else if isBottom {
		if scrollbarHere {
			char = '╨'
		} else {
			char = '┴'
		}
	} else if scrollbarHere {
		char = '║'
	}
	return
}

func (view *MessageView) calculateScrollBar(height int) (scrollBarHeight, scrollBarPos int) {
	viewportHeight := float64(height)
	contentHeight := float64(view.TotalHeight())

	scrollBarHeight = int(math.Ceil(viewportHeight / (contentHeight / viewportHeight)))

	scrollBarPos = height - int(math.Round(float64(view.ScrollOffset)/contentHeight*viewportHeight))

	return
}

func (view *MessageView) getIndexOffset(screen mauview.Screen, height, messageX int) (indexOffset int) {
	indexOffset = view.TotalHeight() - view.ScrollOffset - height
	if indexOffset <= -PaddingAtTop {
		message := "Scroll up to load more messages."
		if atomic.LoadInt32(&view.loadingMessages) == 1 {
			message = "Loading more messages..."
		}
		widget.WriteLineSimpleColor(screen, message, messageX, 0, tcell.ColorGreen)
	}
	return
}

func (view *MessageView) CapturePlaintext(height int) string {
	var buf strings.Builder
	indexOffset := view.TotalHeight() - view.ScrollOffset - height
	var prevMessage *messages.UIMessage
	view.msgBufferLock.RLock()
	for line := 0; line < height; line++ {
		index := indexOffset + line
		if index < 0 {
			continue
		}

		message := view.msgBuffer[index]
		if message != prevMessage {
			var sender string
			if len(message.Sender()) > 0 {
				sender = fmt.Sprintf(" <%s>", message.Sender())
			} else if message.Type == event.MsgEmote {
				sender = fmt.Sprintf(" * %s", message.SenderName)
			}
			fmt.Fprintf(&buf, "%s%s %s\n", message.FormatTime(), sender, message.PlainText())
			prevMessage = message
		}
	}
	view.msgBufferLock.RUnlock()
	return buf.String()
}

func (view *MessageView) Draw(screen mauview.Screen) {
	view.setSize(screen.Size())
	view.recalculateBuffers()

	height := view.Height()
	if view.TotalHeight() == 0 {
		widget.WriteLineSimple(screen, "It's quite empty in here.", 0, height)
		return
	}

	usernameX := 0
	if !view.config.Preferences.HideTimestamp {
		usernameX += view.TimestampWidth + TimestampSenderGap
	}
	messageX := usernameX + view.widestSender() + SenderMessageGap

	bareMode := view.config.Preferences.BareMessageView
	if bareMode {
		messageX = 0
	}

	indexOffset := view.getIndexOffset(screen, height, messageX)

	viewStart := 0
	if indexOffset < 0 {
		viewStart = -indexOffset
	}

	if !bareMode {
		separatorX := usernameX + view.widestSender() + SenderSeparatorGap
		scrollBarHeight, scrollBarPos := view.calculateScrollBar(height)

		for line := viewStart; line < height; line++ {
			showScrollbar := line-viewStart >= scrollBarPos-scrollBarHeight && line-viewStart < scrollBarPos
			isTop := line == viewStart && view.ScrollOffset+height >= view.TotalHeight()
			isBottom := line == height-1 && view.ScrollOffset == 0

			borderChar, borderStyle := getScrollbarStyle(showScrollbar, isTop, isBottom)

			screen.SetContent(separatorX, line, borderChar, nil, borderStyle)
		}
	}

	var prevMsg *messages.UIMessage
	view.msgBufferLock.RLock()
	for line := viewStart; line < height && indexOffset+line < len(view.msgBuffer); {
		index := indexOffset + line

		msg := view.msgBuffer[index]
		if msg == prevMsg {
			debug.Print("Unexpected re-encounter of", msg, msg.Height(), "at", line, index)
			line++
			continue
		}

		if len(msg.FormatTime()) > 0 && !view.config.Preferences.HideTimestamp {
			widget.WriteLineSimpleColor(screen, msg.FormatTime(), 0, line, msg.TimestampColor())
		}
		// TODO hiding senders might not be that nice after all, maybe an option? (disabled for now)
		//if !bareMode && (prevMsg == nil || meta.Sender() != prevMsg.Sender()) {
		widget.WriteLineColor(
			screen, mauview.AlignRight, msg.Sender(),
			usernameX, line, view.widestSender(),
			msg.SenderColor())
		//}
		if msg.Edited {
			// TODO add better indicator for edits
			screen.SetCell(usernameX+view.widestSender(), line, tcell.StyleDefault.Foreground(tcell.ColorDarkRed), '*')
		}

		for i := index - 1; i >= 0 && view.msgBuffer[i] == msg; i-- {
			line--
		}
		msg.Draw(mauview.NewProxyScreen(screen, messageX, line, view.width()-messageX, msg.Height()))
		line += msg.Height()

		prevMsg = msg
	}
	view.msgBufferLock.RUnlock()
}
