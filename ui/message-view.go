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
	"math"
	"strings"

	"github.com/mattn/go-runewidth"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
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
	LoadingMessages bool

	widestSender int
	width        int
	height       int
	prevWidth    int
	prevHeight   int
	prevMsgCount int
	prevPrefs    config.UserPreferences

	messageIDs map[string]messages.UIMessage
	messages   []messages.UIMessage

	msgBuffer []messages.UIMessage
}

func NewMessageView(parent *RoomView) *MessageView {
	return &MessageView{
		parent: parent,
		config: parent.config,

		MaxSenderWidth: 15,
		TimestampWidth: len(messages.TimeFormat),
		ScrollOffset:   0,

		messages:   make([]messages.UIMessage, 0),
		messageIDs: make(map[string]messages.UIMessage),
		msgBuffer:  make([]messages.UIMessage, 0),

		width:        80,
		widestSender: 5,
		prevWidth:    -1,
		prevHeight:   -1,
		prevMsgCount: -1,
	}
}

func (view *MessageView) updateWidestSender(sender string) {
	if len(sender) > view.widestSender {
		view.widestSender = len(sender)
		if view.widestSender > view.MaxSenderWidth {
			view.widestSender = view.MaxSenderWidth
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
	message, ok := ifcMessage.(messages.UIMessage)
	if !ok {
		debug.Print("[Warning] Passed non-UIMessage ifc.Message object to AddMessage().")
		debug.PrintStack()
		return
	}

	var oldMsg messages.UIMessage
	var messageExists bool
	if oldMsg, messageExists = view.messageIDs[message.ID()]; messageExists {
		view.replaceMessage(oldMsg, message)
		direction = IgnoreMessage
	} else if oldMsg, messageExists = view.messageIDs[message.TxnID()]; messageExists {
		view.replaceMessage(oldMsg, message)
		delete(view.messageIDs, message.TxnID())
		direction = IgnoreMessage
	}

	view.updateWidestSender(message.Sender())

	width := view.width
	bare := view.config.Preferences.BareMessageView
	if !bare {
		width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	}
	message.CalculateBuffer(view.config.Preferences, width)

	makeDateChange := func() messages.UIMessage {
		dateChange := messages.NewDateChangeMessage(
			fmt.Sprintf("Date changed to %s", message.FormatDate()))
		dateChange.CalculateBuffer(view.config.Preferences, width)
		view.appendBuffer(dateChange)
		return dateChange
	}

	if direction == AppendMessage {
		if view.ScrollOffset > 0 {
			view.ScrollOffset += message.Height()
		}
		if len(view.messages) > 0 && !view.messages[len(view.messages)-1].SameDate(message) {
			view.messages = append(view.messages, makeDateChange(), message)
		} else {
			view.messages = append(view.messages, message)
		}
		view.appendBuffer(message)
	} else if direction == PrependMessage {
		if len(view.messages) > 0 && !view.messages[0].SameDate(message) {
			view.messages = append([]messages.UIMessage{message, makeDateChange()}, view.messages...)
		} else {
			view.messages = append([]messages.UIMessage{message}, view.messages...)
		}
	} else if oldMsg != nil {
		view.replaceBuffer(oldMsg, message)
	} else {
		debug.Print("Unexpected AddMessage() call: Direction is not append or prepend, but message is new.")
		debug.PrintStack()
	}

	if len(message.ID()) > 0 {
		view.messageIDs[message.ID()] = message
	}
}

func (view *MessageView) appendBuffer(message messages.UIMessage) {
	for i := 0; i < message.Height(); i++ {
		view.msgBuffer = append(view.msgBuffer, message)
	}
	view.prevMsgCount++
}

func (view *MessageView) replaceMessage(original messages.UIMessage, new messages.UIMessage) {
	if len(new.ID()) > 0 {
		view.messageIDs[new.ID()] = new
	}
	for index, msg := range view.messages {
		if msg == original {
			view.messages[index] = new
		}
	}
}

func (view *MessageView) replaceBuffer(original messages.UIMessage, new messages.UIMessage) {
	start := -1
	end := -1
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

	if start == -1 {
		debug.Print("Called replaceBuffer() with message that was not in the buffer:", original.ID())
		debug.PrintStack()
		view.appendBuffer(new)
		return
	}

	if len(view.msgBuffer) > end {
		end++
	}

	if new.Height() == 0 {
		new.CalculateBuffer(view.prevPrefs, view.prevWidth)
	}

	if new.Height() != end-start {
		metaBuffer := view.msgBuffer[0:start]
		for i := 0; i < new.Height(); i++ {
			metaBuffer = append(metaBuffer, new)
		}
		view.msgBuffer = append(metaBuffer, view.msgBuffer[end:]...)
	} else {
		for i := start; i < end; i++ {
			view.msgBuffer[i] = new
		}
	}
}

func (view *MessageView) recalculateBuffers() {
	prefs := view.config.Preferences
	recalculateMessageBuffers := view.width != view.prevWidth ||
		view.prevPrefs.BareMessageView != prefs.BareMessageView ||
		view.prevPrefs.DisableImages != prefs.DisableImages
	if recalculateMessageBuffers || len(view.messages) != view.prevMsgCount {
		width := view.width
		if !prefs.BareMessageView {
			width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
		}
		view.msgBuffer = []messages.UIMessage{}
		view.prevMsgCount = 0
		for i, message := range view.messages {
			if message == nil {
				debug.Print("O.o found nil message at", i)
				break
			}
			if recalculateMessageBuffers {
				message.CalculateBuffer(prefs, width)
			}
			view.appendBuffer(message)
		}
	}
	view.prevHeight = view.height
	view.prevWidth = view.width
	view.prevPrefs = prefs
}

func (view *MessageView) handleMessageClick(message messages.UIMessage) bool {
	switch message := message.(type) {
	case *messages.ImageMessage:
		open.Open(message.Path())
	case messages.UIMessage:
		debug.Print("Message clicked:", message.NotificationContent())
	}
	return false
}

func (view *MessageView) handleUsernameClick(message messages.UIMessage, prevMessage messages.UIMessage) bool {
	if prevMessage != nil && prevMessage.Sender() == message.Sender() {
		return false
	}

	if len(message.Sender()) == 0 {
		return false
	}
	sender := fmt.Sprintf("[%s](https://matrix.to/#/%s)", message.Sender(), message.SenderID())

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
		line := view.TotalHeight() - view.ScrollOffset - view.height + y
		if line < 0 || line >= view.TotalHeight() {
			return false
		}

		message := view.msgBuffer[line]
		var prevMessage messages.UIMessage
		if y != 0 && line > 0 {
			prevMessage = view.msgBuffer[line-1]
		}

		usernameX := view.TimestampWidth + TimestampSenderGap
		messageX := usernameX + view.widestSender + SenderMessageGap

		if x >= messageX {
			return view.handleMessageClick(message)
		} else if x >= usernameX {
			return view.handleUsernameClick(message, prevMessage)
		}
	}
	return false
}

const PaddingAtTop = 5

func (view *MessageView) AddScrollOffset(diff int) {
	totalHeight := view.TotalHeight()
	if diff >= 0 && view.ScrollOffset+diff >= totalHeight-view.height+PaddingAtTop {
		view.ScrollOffset = totalHeight - view.height + PaddingAtTop
	} else {
		view.ScrollOffset += diff
	}

	if view.ScrollOffset > totalHeight-view.height+PaddingAtTop {
		view.ScrollOffset = totalHeight - view.height + PaddingAtTop
	}
	if view.ScrollOffset < 0 {
		view.ScrollOffset = 0
	}
}

func (view *MessageView) Height() int {
	return view.height
}

func (view *MessageView) TotalHeight() int {
	return len(view.msgBuffer)
}

func (view *MessageView) IsAtTop() bool {
	return view.ScrollOffset >= len(view.msgBuffer)-view.height+PaddingAtTop
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
		if view.LoadingMessages {
			message = "Loading more messages..."
		}
		widget.WriteLineSimpleColor(screen, message, messageX, 0, tcell.ColorGreen)
	}
	return
}

func (view *MessageView) CapturePlaintext(height int) string {
	var buf strings.Builder
	indexOffset := view.TotalHeight() - view.ScrollOffset - height
	var prevMessage messages.UIMessage
	for line := 0; line < height; line++ {
		index := indexOffset + line
		if index < 0 {
			continue
		}

		meta := view.msgBuffer[index]
		message, ok := meta.(messages.UIMessage)
		if ok && message != prevMessage {
			var sender string
			if len(message.Sender()) > 0 {
				sender = fmt.Sprintf(" <%s>", message.Sender())
			} else if message.Type() == "m.emote" {
				sender = fmt.Sprintf(" * %s", message.RealSender())
			}
			fmt.Fprintf(&buf, "%s%s %s\n", message.FormatTime(), sender, message.PlainText())
			prevMessage = message
		}
	}
	return buf.String()
}

func (view *MessageView) Draw(screen mauview.Screen) {
	view.width, view.height = screen.Size()
	view.recalculateBuffers()

	if view.TotalHeight() == 0 {
		widget.WriteLineSimple(screen, "It's quite empty in here.", 0, view.height)
		return
	}

	usernameX := view.TimestampWidth + TimestampSenderGap
	messageX := usernameX + view.widestSender + SenderMessageGap

	bareMode := view.config.Preferences.BareMessageView
	if bareMode {
		messageX = 0
	}

	indexOffset := view.getIndexOffset(screen, view.height, messageX)

	viewStart := 0
	if indexOffset < 0 {
		viewStart = -indexOffset
	}

	if !bareMode {
		separatorX := usernameX + view.widestSender + SenderSeparatorGap
		scrollBarHeight, scrollBarPos := view.calculateScrollBar(view.height)

		for line := viewStart; line < view.height; line++ {
			showScrollbar := line-viewStart >= scrollBarPos-scrollBarHeight && line-viewStart < scrollBarPos
			isTop := line == viewStart && view.ScrollOffset+view.height >= view.TotalHeight()
			isBottom := line == view.height-1 && view.ScrollOffset == 0

			borderChar, borderStyle := getScrollbarStyle(showScrollbar, isTop, isBottom)

			screen.SetContent(separatorX, line, borderChar, nil, borderStyle)
		}
	}

	var prevMsg messages.UIMessage
	for line := viewStart; line < view.height && indexOffset+line < view.TotalHeight(); line++ {
		index := indexOffset + line

		msg := view.msgBuffer[index]
		if msg != prevMsg {
			if len(msg.FormatTime()) > 0 {
				widget.WriteLineSimpleColor(screen, msg.FormatTime(), 0, line, msg.TimestampColor())
			}
			// TODO hiding senders might not be that nice after all, maybe an option? (disabled for now)
			//if !bareMode && (prevMsg == nil || meta.Sender() != prevMsg.Sender()) {
			widget.WriteLineColor(
				screen, mauview.AlignRight, msg.Sender(),
				usernameX, line, view.widestSender,
				msg.SenderColor())
			//}
			prevMsg = msg
		}

		for i := index - 1; i >= 0 && view.msgBuffer[i] == msg; i-- {
			line--
		}
		msg.Draw(mauview.NewProxyScreen(screen, messageX, line, view.width-messageX, msg.Height()))
		line += msg.Height() - 1
	}
}
