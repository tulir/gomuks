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
	"regexp"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"maunium.net/go/tview"
)

type Message struct {
	ID           string
	Sender       string
	Text         string
	Timestamp    string
	RenderSender bool

	buffer      []string
	senderColor tcell.Color
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

	messages []*Message

	debug DebugPrinter
}

func NewMessageView(debug DebugPrinter) *MessageView {
	return &MessageView{
		Box:             tview.NewBox(),
		MaxSenderWidth:  20,
		TimestampFormat: "15:04:05",
		TimestampWidth:  8,
		Separator:       '|',
		ScrollOffset:    0,

		widestSender:        5,
		prevWidth:           -1,
		prevHeight:          -1,
		prevScrollOffset:    -1,
		firstDisplayMessage: -1,
		lastDisplayMessage:  -1,
		totalHeight:         -1,

		debug: debug,
	}
}

func (view *MessageView) recalculateBuffers(width int) {
	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	for _, message := range view.messages {
		message.calculateBuffer(width)
	}
	view.prevWidth = width
}

func (view *MessageView) AddMessage(id, sender, text string, timestamp time.Time) {
	if len(sender) > view.widestSender {
		view.widestSender = len(sender)
		if view.widestSender > view.MaxSenderWidth {
			view.widestSender = view.MaxSenderWidth
		}
	}
	message := &Message{
		ID:           id,
		Sender:       sender,
		RenderSender: true,
		Text:         text,
		Timestamp:    timestamp.Format(view.TimestampFormat),
		senderColor:  getColor(sender),
	}
	_, _, width, height := view.GetInnerRect()
	width -= view.TimestampWidth + TimestampSenderGap + view.widestSender + SenderMessageGap
	message.calculateBuffer(width)
	if view.ScrollOffset > 0 {
		view.ScrollOffset += len(message.buffer)
	}
	if len(view.messages) > 0 && view.messages[len(view.messages)-1].Sender == message.Sender {
		message.RenderSender = false
	}
	view.messages = append(view.messages, message)
	view.recalculateHeight(height)
}

func (view *MessageView) recalculateHeight(height int) {
	view.firstDisplayMessage = -1
	view.lastDisplayMessage = -1
	view.totalHeight = 0
	for i := len(view.messages) - 1; i >= 0; i-- {
		prevTotalHeight := view.totalHeight
		view.totalHeight += len(view.messages[i].buffer)

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

const (
	TimestampSenderGap = 1
	SenderSeparatorGap = 1
	SenderMessageGap   = 3
)

func (view *MessageView) Draw(screen tcell.Screen) {
	view.Box.Draw(screen)

	x, y, width, height := view.GetInnerRect()
	if width != view.prevWidth {
		view.recalculateBuffers(width)
	}
	if height != view.prevHeight || width != view.prevWidth || view.ScrollOffset != view.prevScrollOffset {
		view.recalculateHeight(height)
	}
	usernameOffsetX := view.TimestampWidth + TimestampSenderGap
	messageOffsetX := usernameOffsetX + view.widestSender + SenderMessageGap

	separatorX := x + usernameOffsetX + view.widestSender + SenderSeparatorGap
	for separatorY := y; separatorY < y+height; separatorY++ {
		screen.SetContent(separatorX, separatorY, view.Separator, nil, tcell.StyleDefault)
	}

	if view.firstDisplayMessage == -1 || view.lastDisplayMessage == -1 {
		return
	}

	writeOffset := 0
	for i := view.firstDisplayMessage; i >= view.lastDisplayMessage; i-- {
		message := view.messages[i]
		messageHeight := len(message.buffer)

		senderAtLine := y + height - writeOffset - messageHeight
		if senderAtLine < y {
			senderAtLine = y
		}
		view.writeLine(screen, message.Timestamp, x, senderAtLine, tcell.ColorDefault)
		if message.RenderSender || i == view.lastDisplayMessage {
			view.writeLine(screen, message.Sender, x+usernameOffsetX, senderAtLine, message.senderColor)
		}

		for num, line := range message.buffer {
			offsetY := height - messageHeight - writeOffset + num
			if offsetY >= 0 {
				view.writeLine(screen, line, x+messageOffsetX, y+offsetY, tcell.ColorDefault)
			}
		}
		writeOffset += messageHeight
	}
}
