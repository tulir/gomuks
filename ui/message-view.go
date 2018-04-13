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
	"encoding/gob"
	"fmt"
	"math"
	"os"

	"maunium.net/go/gomuks/lib/open"
	"maunium.net/go/gomuks/ui/messages/tstring"
	"maunium.net/go/tcell"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/messages"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tview"
)

type MessageView struct {
	*tview.Box

	ScrollOffset    int
	MaxSenderWidth  int
	DateFormat      string
	TimestampFormat string
	TimestampWidth  int
	LoadingMessages bool

	widestSender int
	prevWidth    int
	prevHeight   int
	prevMsgCount int

	messageIDs map[string]messages.UIMessage
	messages   []messages.UIMessage

	textBuffer []tstring.TString
	metaBuffer []ifc.MessageMeta
}

func NewMessageView() *MessageView {
	return &MessageView{
		Box:            tview.NewBox(),
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

func (view *MessageView) LoadHistory(gmx ifc.Gomuks, path string) (int, error) {
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
			message.RegisterGomuks(gmx)
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
		oldMsg.CopyFrom(message)
		message = oldMsg
		direction = ifc.IgnoreMessage
	}

	view.updateWidestSender(message.Sender())

	_, _, width, _ := view.GetInnerRect()
	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	message.CalculateBuffer(width)

	if direction == ifc.AppendMessage {
		if view.ScrollOffset > 0 {
			view.ScrollOffset += message.Height()
		}
		view.messages = append(view.messages, message)
		view.appendBuffer(message)
	} else if direction == ifc.PrependMessage {
		view.messages = append([]messages.UIMessage{message}, view.messages...)
	} else {
		view.replaceBuffer(message)
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

func (view *MessageView) replaceBuffer(message messages.UIMessage) {
	start := -1
	end := -1
	for index, meta := range view.metaBuffer {
		if meta == message {
			if start == -1 {
				start = index
			}
			end = index
		} else if start != -1 {
			break
		}
	}

	if len(view.textBuffer) > end {
		end++
	}

	view.textBuffer = append(append(view.textBuffer[0:start], message.Buffer()...), view.textBuffer[end:]...)
	if len(message.Buffer()) != end-start+1 {
		metaBuffer := view.metaBuffer[0:start]
		for range message.Buffer() {
			metaBuffer = append(metaBuffer, message)
		}
		view.metaBuffer = append(metaBuffer, view.metaBuffer[end:]...)
	}
}

func (view *MessageView) recalculateBuffers() {
	_, _, width, height := view.GetInnerRect()

	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	recalculateMessageBuffers := width != view.prevWidth
	if height != view.prevHeight || recalculateMessageBuffers || len(view.messages) != view.prevMsgCount {
		view.textBuffer = []tstring.TString{}
		view.metaBuffer = []ifc.MessageMeta{}
		view.prevMsgCount = 0
		for i, message := range view.messages {
			if message == nil {
				debug.Print("O.o found nil message at", i)
				break
			}
			if recalculateMessageBuffers {
				message.CalculateBuffer(width)
			}
			view.appendBuffer(message)
		}
		view.prevHeight = height
		view.prevWidth = width
	}
}

func (view *MessageView) HandleClick(x, y int, button tcell.ButtonMask) {
	if button != tcell.Button1 {
		return
	}

	_, _, _, height := view.GetRect()
	line := view.TotalHeight() - view.ScrollOffset - height + y
	if line < 0 || line >= view.TotalHeight() {
		return
	}

	message := view.metaBuffer[line]
	imageMessage, ok := message.(*messages.ImageMessage)
	if !ok {
		uiMessage, ok := message.(messages.UIMessage)
		if ok {
			debug.Print("Message clicked:", uiMessage.NotificationContent())
		}
		return
	}

	open.Open(imageMessage.Path())
}

const PaddingAtTop = 5

func (view *MessageView) AddScrollOffset(diff int) {
	_, _, _, height := view.GetInnerRect()

	totalHeight := view.TotalHeight()
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

func (view *MessageView) Height() int {
	_, _, _, height := view.GetInnerRect()
	return height
}

func (view *MessageView) TotalHeight() int {
	return len(view.textBuffer)
}

func (view *MessageView) IsAtTop() bool {
	_, _, _, height := view.GetInnerRect()
	totalHeight := len(view.textBuffer)
	return view.ScrollOffset >= totalHeight-height+PaddingAtTop
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

func (view *MessageView) Draw(screen tcell.Screen) {
	view.Box.Draw(screen)

	x, y, _, height := view.GetInnerRect()
	view.recalculateBuffers()

	if view.TotalHeight() == 0 {
		widget.WriteLineSimple(screen, "It's quite empty in here.", x, y+height)
		return
	}

	usernameX := x + view.TimestampWidth + TimestampSenderGap
	messageX := usernameX + view.widestSender + SenderMessageGap
	separatorX := usernameX + view.widestSender + SenderSeparatorGap

	indexOffset := view.TotalHeight() - view.ScrollOffset - height
	if indexOffset <= -PaddingAtTop {
		message := "Scroll up to load more messages."
		if view.LoadingMessages {
			message = "Loading more messages..."
		}
		widget.WriteLineSimpleColor(screen, message, messageX, y, tcell.ColorGreen)
	}

	if len(view.textBuffer) != len(view.metaBuffer) {
		debug.Printf("Unexpected text/meta buffer length mismatch: %d != %d.", len(view.textBuffer), len(view.metaBuffer))
		return
	}

	var scrollBarHeight, scrollBarPos int
	// Black magic (aka math) used to figure out where the scroll bar should be put.
	{
		viewportHeight := float64(height)
		contentHeight := float64(view.TotalHeight())

		scrollBarHeight = int(math.Ceil(viewportHeight / (contentHeight / viewportHeight)))

		scrollBarPos = height - int(math.Round(float64(view.ScrollOffset)/contentHeight*viewportHeight))
	}

	var prevMeta ifc.MessageMeta
	firstLine := true
	skippedLines := 0

	for line := 0; line < height; line++ {
		index := indexOffset + line
		if index < 0 {
			skippedLines++
			continue
		} else if index >= view.TotalHeight() {
			break
		}

		showScrollbar := line-skippedLines >= scrollBarPos-scrollBarHeight && line-skippedLines < scrollBarPos
		isTop := firstLine && view.ScrollOffset+height >= view.TotalHeight()
		isBottom := line == height-1 && view.ScrollOffset == 0

		borderChar, borderStyle := getScrollbarStyle(showScrollbar, isTop, isBottom)

		firstLine = false

		screen.SetContent(separatorX, y+line, borderChar, nil, borderStyle)

		text, meta := view.textBuffer[index], view.metaBuffer[index]
		if meta != prevMeta {
			if len(meta.FormatTime()) > 0 {
				widget.WriteLineSimpleColor(screen, meta.FormatTime(), x, y+line, meta.TimestampColor())
			}
			if prevMeta == nil || meta.Sender() != prevMeta.Sender() {
				widget.WriteLineColor(
					screen, tview.AlignRight, meta.Sender(),
					usernameX, y+line, view.widestSender,
					meta.SenderColor())
			}
			prevMeta = meta
		}

		text.Draw(screen, messageX, y+line)
	}
}
