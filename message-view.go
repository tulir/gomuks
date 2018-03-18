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

package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"maunium.net/go/tview"
)

type Message struct {
	ID        string
	Sender    string
	Text      string
	Timestamp string
	Date      string

	buffer      []string
	senderColor tcell.Color
}

func NewMessage(id, sender, text, timestamp, date string, senderColor tcell.Color) *Message {
	return &Message{
		ID:          id,
		Sender:      sender,
		Text:        text,
		Timestamp:   timestamp,
		Date:        date,
		senderColor: senderColor,
	}
}

var (
	boundaryPattern = regexp.MustCompile("([[:punct:]]\\s*|\\s+)")
	spacePattern    = regexp.MustCompile(`\s+`)
)

func (message *Message) calculateBuffer(width int) {
	if width < 1 {
		return
	}
	message.buffer = []string{}
	forcedLinebreaks := strings.Split(message.Text, "\n")
	newlines := 0
	for _, str := range forcedLinebreaks {
		if len(str) == 0 && newlines < 1 {
			message.buffer = append(message.buffer, "")
			newlines++
		} else {
			newlines = 0
		}
		// From tview/textview.go#reindexBuffer()
		for len(str) > 0 {
			extract := runewidth.Truncate(str, width, "")
			if len(extract) < len(str) {
				if spaces := spacePattern.FindStringIndex(str[len(extract):]); spaces != nil && spaces[0] == 0 {
					extract = str[:len(extract)+spaces[1]]
				}

				matches := boundaryPattern.FindAllStringIndex(extract, -1)
				if len(matches) > 0 {
					extract = extract[:matches[len(matches)-1][1]]
				}
			}
			message.buffer = append(message.buffer, extract)
			str = str[len(extract):]
		}
	}
}

type MessageView struct {
	*tview.Box

	ScrollOffset    int
	MaxSenderWidth  int
	DateFormat      string
	TimestampFormat string
	TimestampWidth  int
	Separator       rune

	widestSender        int
	prevWidth           int
	prevHeight          int
	prevScrollOffset    int
	firstDisplayMessage int
	lastDisplayMessage  int
	totalHeight         int

	messageIDs map[string]bool
	messages   []*Message
}

func NewMessageView() *MessageView {
	return &MessageView{
		Box:             tview.NewBox(),
		MaxSenderWidth:  20,
		DateFormat:      "January _2, 2006",
		TimestampFormat: "15:04:05",
		TimestampWidth:  8,
		Separator:       '|',
		ScrollOffset:    0,

		messages:   make([]*Message, 0),
		messageIDs: make(map[string]bool),

		widestSender:        5,
		prevWidth:           -1,
		prevHeight:          -1,
		prevScrollOffset:    -1,
		firstDisplayMessage: -1,
		lastDisplayMessage:  -1,
		totalHeight:         -1,
	}
}

func (view *MessageView) NewMessage(id, sender, text string, timestamp time.Time) *Message {
	return NewMessage(id, sender, text,
		timestamp.Format(view.TimestampFormat),
		timestamp.Format(view.DateFormat),
		getColor(sender))
}

func (view *MessageView) recalculateBuffers() {
	_, _, width, _ := view.GetInnerRect()
	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	if width != view.prevWidth {
		for _, message := range view.messages {
			message.calculateBuffer(width)
		}
		view.prevWidth = width
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

const (
	AppendMessage  int = iota
	PrependMessage
)

func (view *MessageView) AddMessage(message *Message, direction int) {
	_, messageExists := view.messageIDs[message.ID]
	if messageExists {
		return
	}

	view.updateWidestSender(message.Sender)

	_, _, width, _ := view.GetInnerRect()
	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	message.calculateBuffer(width)

	if direction == AppendMessage {
		if view.ScrollOffset > 0 {
			view.ScrollOffset += len(message.buffer)
		}
		view.messages = append(view.messages, message)
	} else if direction == PrependMessage {
		view.messages = append([]*Message{message}, view.messages...)
	}

	view.messageIDs[message.ID] = true
	view.recalculateHeight()
}

func (view *MessageView) recalculateHeight() {
	_, _, width, height := view.GetInnerRect()
	if height != view.prevHeight || width != view.prevWidth || view.ScrollOffset != view.prevScrollOffset {
		view.firstDisplayMessage = -1
		view.lastDisplayMessage = -1
		view.totalHeight = 0
		prevDate := ""
		for i := len(view.messages) - 1; i >= 0; i-- {
			prevTotalHeight := view.totalHeight
			message := view.messages[i]
			view.totalHeight += len(message.buffer)
			if message.Date != prevDate {
				if len(prevDate) != 0 {
					view.totalHeight++
				}
				prevDate = message.Date
			}

			if view.totalHeight < view.ScrollOffset {
				continue
			} else if view.firstDisplayMessage == -1 {
				view.lastDisplayMessage = i
				view.firstDisplayMessage = i
			}

			if prevTotalHeight < height+view.ScrollOffset {
				view.lastDisplayMessage = i
			}
		}
		view.prevScrollOffset = view.ScrollOffset
	}
}

func (view *MessageView) PageUp() {
	_, _, _, height := view.GetInnerRect()
	view.ScrollOffset += height / 2
	if view.ScrollOffset > view.totalHeight-height {
		view.ScrollOffset = view.totalHeight - height + 5
	}
}

func (view *MessageView) PageDown() {
	_, _, _, height := view.GetInnerRect()
	view.ScrollOffset -= height / 2
	if view.ScrollOffset < 0 {
		view.ScrollOffset = 0
	}
}

func (view *MessageView) writeLine(screen tcell.Screen, line string, x, y int, color tcell.Color) {
	offsetX := 0
	for _, ch := range line {
		chWidth := runewidth.RuneWidth(ch)
		if chWidth == 0 {
			continue
		}

		for localOffset := 0; localOffset < chWidth; localOffset++ {
			screen.SetContent(x+offsetX+localOffset, y, ch, nil, tcell.StyleDefault.Foreground(color))
		}
		offsetX += chWidth
	}
}

func (view *MessageView) writeLineRight(screen tcell.Screen, line string, x, y, maxWidth int, color tcell.Color) {
	offsetX := maxWidth - runewidth.StringWidth(line)
	if offsetX < 0 {
		offsetX = 0
	}
	for _, ch := range line {
		chWidth := runewidth.RuneWidth(ch)
		if chWidth == 0 {
			continue
		}

		for localOffset := 0; localOffset < chWidth; localOffset++ {
			screen.SetContent(x+offsetX+localOffset, y, ch, nil, tcell.StyleDefault.Foreground(color))
		}
		offsetX += chWidth
	}
}

const (
	TimestampSenderGap = 1
	SenderSeparatorGap = 1
	SenderMessageGap   = 3
)

func (view *MessageView) Draw(screen tcell.Screen) {
	view.Box.Draw(screen)

	x, y, _, height := view.GetInnerRect()
	view.recalculateBuffers()
	view.recalculateHeight()

	if view.firstDisplayMessage == -1 || view.lastDisplayMessage == -1 {
		view.writeLine(screen, "It's quite empty in here.", x, y+height, tcell.ColorDefault)
		return
	}

	usernameOffsetX := view.TimestampWidth + TimestampSenderGap
	messageOffsetX := usernameOffsetX + view.widestSender + SenderMessageGap
	separatorX := x + usernameOffsetX + view.widestSender + SenderSeparatorGap
	for separatorY := y; separatorY < y+height; separatorY++ {
		screen.SetContent(separatorX, separatorY, view.Separator, nil, tcell.StyleDefault)
	}

	writeOffset := 0
	prevDate := ""
	prevSender := ""
	prevSenderLine := -1
	for i := view.firstDisplayMessage; i >= view.lastDisplayMessage; i-- {
		message := view.messages[i]
		messageHeight := len(message.buffer)

		// Show message when the date changes.
		if message.Date != prevDate {
			if len(prevDate) > 0 {
				writeOffset++
				view.writeLine(
					screen, fmt.Sprintf("Date changed to %s", prevDate),
					x+messageOffsetX, y+height-writeOffset, tcell.ColorGreen)
			}
			prevDate = message.Date
		}

		senderAtLine := y + height - writeOffset - messageHeight
		// The message may be only partially on screen, so we need to make sure the sender
		// is on screen even when the message is not shown completely.
		if senderAtLine < y {
			senderAtLine = y
		}

		view.writeLine(screen, message.Timestamp, x, senderAtLine, tcell.ColorDefault)
		view.writeLineRight(screen, message.Sender,
			x+usernameOffsetX, senderAtLine,
			view.widestSender, message.senderColor)

		if message.Sender == prevSender {
			// Sender is same as previous. We're looping from bottom to top, and we want the
			// sender name only on the topmost message, so clear out the duplicate sender name
			// below.
			view.writeLineRight(screen, strings.Repeat(" ", view.widestSender),
				x+usernameOffsetX, prevSenderLine,
				view.widestSender, message.senderColor)
		}
		prevSender = message.Sender
		prevSenderLine = senderAtLine

		for num, line := range message.buffer {
			offsetY := height - messageHeight - writeOffset + num
			// Only render message if it's within the message view.
			if offsetY >= 0 {
				view.writeLine(screen, line, x+messageOffsetX, y+offsetY, tcell.ColorDefault)
			}
		}
		writeOffset += messageHeight
	}
}
