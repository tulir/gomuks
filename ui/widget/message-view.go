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
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/gdamore/tcell"
	"maunium.net/go/gomuks/ui/debug"
	"maunium.net/go/gomuks/ui/types"
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

	messageIDs map[string]*types.Message
	messages   []*types.Message

	textBuffer []string
	metaBuffer []types.MessageMeta
}

func NewMessageView() *MessageView {
	return &MessageView{
		Box:             tview.NewBox(),
		MaxSenderWidth:  15,
		DateFormat:      "January _2, 2006",
		TimestampFormat: "15:04:05",
		TimestampWidth:  8,
		ScrollOffset:    0,

		messages:   make([]*types.Message, 0),
		messageIDs: make(map[string]*types.Message),
		textBuffer: make([]string, 0),
		metaBuffer: make([]types.MessageMeta, 0),

		widestSender: 5,
		prevWidth:    -1,
		prevHeight:   -1,
		prevMsgCount: -1,
	}
}

func (view *MessageView) NewMessage(id, sender, msgtype, text string, timestamp time.Time) *types.Message {
	return types.NewMessage(id, sender, msgtype, text,
		timestamp.Format(view.TimestampFormat),
		timestamp.Format(view.DateFormat),
		GetHashColor(sender))
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

func (view *MessageView) LoadHistory(path string) (int, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return -1, err
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	err = dec.Decode(&view.messages)
	if err != nil {
		return -1, err
	}

	for _, message := range view.messages {
		view.updateWidestSender(message.Sender)
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

type MessageDirection int

const (
	AppendMessage  MessageDirection = iota
	PrependMessage
	IgnoreMessage
)

func (view *MessageView) UpdateMessageID(message *types.Message, newID string) {
	delete(view.messageIDs, message.ID)
	message.ID = newID
	view.messageIDs[message.ID] = message
}

func (view *MessageView) AddMessage(message *types.Message, direction MessageDirection) {
	if message == nil {
		return
	}

	msg, messageExists := view.messageIDs[message.ID]
	if msg != nil && messageExists {
		message.CopyTo(msg)
		message = msg
		direction = IgnoreMessage
	}

	view.updateWidestSender(message.Sender)

	_, _, width, _ := view.GetInnerRect()
	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	message.CalculateBuffer(width)

	if direction == AppendMessage {
		if view.ScrollOffset > 0 {
			view.ScrollOffset += message.Height()
		}
		view.messages = append(view.messages, message)
		view.appendBuffer(message)
	} else if direction == PrependMessage {
		view.messages = append([]*types.Message{message}, view.messages...)
	}

	view.messageIDs[message.ID] = message
}

func (view *MessageView) appendBuffer(message *types.Message) {
	if len(view.metaBuffer) > 0 {
		prevMeta := view.metaBuffer[len(view.metaBuffer)-1]
		if prevMeta != nil && prevMeta.GetDate() != message.Date {
			view.textBuffer = append(view.textBuffer, fmt.Sprintf("Date changed to %s", message.Date))
			view.metaBuffer = append(view.metaBuffer, &types.BasicMeta{TextColor: tcell.ColorGreen})
		}
	}

	view.textBuffer = append(view.textBuffer, message.Buffer()...)
	for range message.Buffer() {
		view.metaBuffer = append(view.metaBuffer, message)
	}
	view.prevMsgCount++
}

func (view *MessageView) recalculateBuffers() {
	_, _, width, height := view.GetInnerRect()

	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	recalculateMessageBuffers := width != view.prevWidth
	if height != view.prevHeight || recalculateMessageBuffers || len(view.messages) != view.prevMsgCount {
		view.textBuffer = []string{}
		view.metaBuffer = []types.MessageMeta{}
		view.prevMsgCount = 0
		for _, message := range view.messages {
			if recalculateMessageBuffers {
				message.CalculateBuffer(width)
			}
			view.appendBuffer(message)
		}
		view.prevHeight = height
		view.prevWidth = width
	}
}

const PaddingAtTop = 5

func (view *MessageView) AddScrollOffset(diff int) {
	_, _, _, height := view.GetInnerRect()

	totalHeight := len(view.textBuffer)
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

	if len(view.textBuffer) == 0 {
		writeLineSimple(screen, "It's quite empty in here.", x, y+height)
		return
	}

	usernameX := x + view.TimestampWidth + TimestampSenderGap
	messageX := usernameX + view.widestSender + SenderMessageGap
	separatorX := usernameX + view.widestSender + SenderSeparatorGap

	indexOffset := len(view.textBuffer) - view.ScrollOffset - height
	if indexOffset <= -PaddingAtTop {
		message := "Scroll up to load more messages."
		if view.LoadingMessages {
			message = "Loading more messages..."
		}
		writeLineSimpleColor(screen, message, messageX, y, tcell.ColorGreen)
	}

	if len(view.textBuffer) != len(view.metaBuffer) {
		debug.ExtPrintf("Unexpected text/meta buffer length mismatch: %d != %d.", len(view.textBuffer), len(view.metaBuffer))
		return
	}

	var scrollBarHeight, scrollBarPos int
	// Black magic (aka math) used to figure out where the scroll bar should be put.
	{
		viewportHeight := float64(height)
		contentHeight := float64(len(view.textBuffer))

		scrollBarHeight = int(math.Ceil(viewportHeight / (contentHeight / viewportHeight)))

		scrollBarPos = height - int(math.Round(float64(view.ScrollOffset)/contentHeight*viewportHeight))
	}

	var prevMeta types.MessageMeta
	firstLine := true
	skippedLines := 0

	for line := 0; line < height; line++ {
		index := indexOffset + line
		if index < 0 {
			skippedLines++
			continue
		} else if index >= len(view.textBuffer) {
			break
		}

		showScrollbar := line-skippedLines >= scrollBarPos-scrollBarHeight && line-skippedLines < scrollBarPos
		isTop := firstLine && view.ScrollOffset+height >= len(view.textBuffer)
		isBottom := line == height-1 && view.ScrollOffset == 0

		borderChar, borderStyle := getScrollbarStyle(showScrollbar, isTop, isBottom)

		firstLine = false

		screen.SetContent(separatorX, y+line, borderChar, nil, borderStyle)

		text, meta := view.textBuffer[index], view.metaBuffer[index]
		if meta != prevMeta {
			if len(meta.GetTimestamp()) > 0 {
				writeLineSimpleColor(screen, meta.GetTimestamp(), x, y+line, meta.GetTimestampColor())
			}
			if prevMeta == nil || meta.GetSender() != prevMeta.GetSender() {
				writeLineColor(
					screen, tview.AlignRight, meta.GetSender(),
					usernameX, y+line, view.widestSender,
					meta.GetSenderColor())
			}
			prevMeta = meta
		}
		writeLineSimpleColor(screen, text, messageX, y+line, meta.GetTextColor())
	}
}
