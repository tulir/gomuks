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
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"

	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/lib/open"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/messages/tstring"
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

	textBuffer []tstring.TString
	metaBuffer []ifc.MessageMeta
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
		textBuffer: make([]tstring.TString, 0),
		metaBuffer: make([]ifc.MessageMeta, 0),

		widestSender: 5,
		prevWidth:    -1,
		prevHeight:   -1,
		prevMsgCount: -1,
	}
}

func (view *MessageView) SaveHistory(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	err = enc.Encode(view.messages)
	if err != nil {
		return err
	}

	return nil
}

func (view *MessageView) LoadHistory(matrix ifc.MatrixContainer, path string) (int, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return -1, err
	}
	defer file.Close()

	var msgs []messages.UIMessage

	dec := gob.NewDecoder(file)
	err = dec.Decode(&msgs)
	if err != nil {
		return -1, err
	}

	view.messages = make([]messages.UIMessage, len(msgs))
	indexOffset := 0
	for index, message := range msgs {
		if message != nil {
			view.messages[index-indexOffset] = message
			view.updateWidestSender(message.Sender())
			message.RegisterMatrix(matrix)
		} else {
			indexOffset++
		}
	}

	return len(view.messages), nil
}

func (view *MessageView) updateWidestSender(sender string) {
	if len(sender) > view.widestSender {
		view.widestSender = len(sender)
		if view.widestSender > view.MaxSenderWidth {
			view.widestSender = view.MaxSenderWidth
		}
	}
}

func (view *MessageView) UpdateMessageID(ifcMessage ifc.Message, newID string) {
	message, ok := ifcMessage.(messages.UIMessage)
	if !ok {
		debug.Print("[Warning] Passed non-UIMessage ifc.Message object to UpdateMessageID().")
		debug.PrintStack()
		return
	}
	delete(view.messageIDs, message.ID())
	message.SetID(newID)
	view.messageIDs[message.ID()] = message
}

func (view *MessageView) AddMessage(ifcMessage ifc.Message, direction ifc.MessageDirection) {
	if ifcMessage == nil {
		return
	}
	message, ok := ifcMessage.(messages.UIMessage)
	if !ok {
		debug.Print("[Warning] Passed non-UIMessage ifc.Message object to AddMessage().")
		debug.PrintStack()
		return
	}

	oldMsg, messageExists := view.messageIDs[message.ID()]
	if messageExists {
		view.replaceMessage(oldMsg, message)
		direction = ifc.IgnoreMessage
	}

	view.updateWidestSender(message.Sender())

	width := view.width
	bare := view.config.Preferences.BareMessageView
	if !bare {
		width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	}
	message.CalculateBuffer(view.config.Preferences, width)

	if direction == ifc.AppendMessage {
		if view.ScrollOffset > 0 {
			view.ScrollOffset += message.Height()
		}
		view.messages = append(view.messages, message)
		view.appendBuffer(message)
	} else if direction == ifc.PrependMessage {
		view.messages = append([]messages.UIMessage{message}, view.messages...)
	} else if oldMsg != nil {
		view.replaceBuffer(oldMsg, message)
	} else {
		debug.Print("Unexpected AddMessage() call: Direction is not append or prepend, but message is new.")
		debug.PrintStack()
	}

	view.messageIDs[message.ID()] = message
}

func (view *MessageView) appendBuffer(message messages.UIMessage) {
	if len(view.metaBuffer) > 0 {
		prevMeta := view.metaBuffer[len(view.metaBuffer)-1]
		if prevMeta != nil && prevMeta.FormatDate() != message.FormatDate() {
			view.textBuffer = append(view.textBuffer, tstring.NewColorTString(
				fmt.Sprintf("Date changed to %s", message.FormatDate()),
				tcell.ColorGreen))
			view.metaBuffer = append(view.metaBuffer, &messages.BasicMeta{
				BTimestampColor: tcell.ColorDefault, BTextColor: tcell.ColorGreen})
		}
	}

	view.textBuffer = append(view.textBuffer, message.Buffer()...)
	for range message.Buffer() {
		view.metaBuffer = append(view.metaBuffer, message)
	}
	view.prevMsgCount++
}

func (view *MessageView) replaceMessage(original messages.UIMessage, new messages.UIMessage) {
	view.messageIDs[new.ID()] = new
	for index, msg := range view.messages {
		if msg == original {
			view.messages[index] = new
		}
	}
}

func (view *MessageView) replaceBuffer(original messages.UIMessage, new messages.UIMessage) {
	start := -1
	end := -1
	for index, meta := range view.metaBuffer {
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

	if len(view.textBuffer) > end {
		end++
	}

	view.textBuffer = append(append(view.textBuffer[0:start], new.Buffer()...), view.textBuffer[end:]...)
	if len(new.Buffer()) != end-start {
		metaBuffer := view.metaBuffer[0:start]
		for range new.Buffer() {
			metaBuffer = append(metaBuffer, new)
		}
		view.metaBuffer = append(metaBuffer, view.metaBuffer[end:]...)
	} else {
		for i := start; i < end; i++ {
			view.metaBuffer[i] = new
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
		view.textBuffer = []tstring.TString{}
		view.metaBuffer = []ifc.MessageMeta{}
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

func (view *MessageView) handleMessageClick(message ifc.MessageMeta) bool {
	switch message := message.(type) {
	case *messages.ImageMessage:
		open.Open(message.Path())
	case messages.UIMessage:
		debug.Print("Message clicked:", message.NotificationContent())
	}
	return false
}

func (view *MessageView) handleUsernameClick(message ifc.MessageMeta, prevMessage ifc.MessageMeta) bool {
	uiMessage, ok := message.(messages.UIMessage)
	if !ok {
		return false
	}

	prevUIMessage, _ := prevMessage.(messages.UIMessage)
	if prevUIMessage != nil && prevUIMessage.Sender() == uiMessage.Sender() {
		return false
	}

	if len(uiMessage.Sender()) == 0 {
		return false
	}
	sender := fmt.Sprintf("[%s](https://matrix.to/#/%s)", uiMessage.Sender(), uiMessage.SenderID())

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

		message := view.metaBuffer[line]
		var prevMessage ifc.MessageMeta
		if y != 0 && line > 0 {
			prevMessage = view.metaBuffer[line-1]
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
	return len(view.textBuffer)
}

func (view *MessageView) IsAtTop() bool {
	totalHeight := len(view.textBuffer)
	return view.ScrollOffset >= totalHeight-view.height+PaddingAtTop
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

		meta := view.metaBuffer[index]
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
	separatorX := usernameX + view.widestSender + SenderSeparatorGap

	bareMode := view.config.Preferences.BareMessageView
	if bareMode {
		messageX = 0
	}

	indexOffset := view.getIndexOffset(screen, view.height, messageX)

	if len(view.textBuffer) != len(view.metaBuffer) {
		debug.Printf("Unexpected text/meta buffer length mismatch: %d != %d.", len(view.textBuffer), len(view.metaBuffer))
		view.prevMsgCount = 0
		return
	}

	scrollBarHeight, scrollBarPos := view.calculateScrollBar(view.height)

	var prevMeta ifc.MessageMeta
	firstLine := true
	skippedLines := 0

	for line := 0; line < view.height; line++ {
		index := indexOffset + line
		if index < 0 {
			skippedLines++
			continue
		} else if index >= view.TotalHeight() {
			break
		}

		showScrollbar := line-skippedLines >= scrollBarPos-scrollBarHeight && line-skippedLines < scrollBarPos
		isTop := firstLine && view.ScrollOffset+view.height >= view.TotalHeight()
		isBottom := line == view.height-1 && view.ScrollOffset == 0

		borderChar, borderStyle := getScrollbarStyle(showScrollbar, isTop, isBottom)

		firstLine = false

		if !bareMode {
			screen.SetContent(separatorX, line, borderChar, nil, borderStyle)
		}

		text, meta := view.textBuffer[index], view.metaBuffer[index]
		if meta != prevMeta {
			if len(meta.FormatTime()) > 0 {
				widget.WriteLineSimpleColor(screen, meta.FormatTime(), 0, line, meta.TimestampColor())
			}
			if !bareMode && (prevMeta == nil || meta.Sender() != prevMeta.Sender()) {
				widget.WriteLineColor(
					screen, mauview.AlignRight, meta.Sender(),
					usernameX, line, view.widestSender,
					meta.SenderColor())
			}
			prevMeta = meta
		}

		text.Draw(screen, messageX, line)
	}
	debug.Print(screen)
}
