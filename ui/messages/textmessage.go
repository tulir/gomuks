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

package messages

import (
	"fmt"
	"regexp"
	"time"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

type TextMessage struct {
	cache       tstring.TString
	buffer      []tstring.TString
	isHighlight bool
	eventID     id.EventID
	Text        string
}

// NewTextMessage creates a new UITextMessage object with the provided values and the default state.
func NewTextMessage(evt *muksevt.Event, displayname string, text string) *UIMessage {
	return newUIMessage(evt, displayname, &TextMessage{
		eventID: evt.ID,
		Text:    text,
	})
}

func NewServiceMessage(text string) *UIMessage {
	return &UIMessage{
		SenderID:   "*",
		SenderName: "*",
		Timestamp:  time.Now(),
		IsService:  true,
		Renderer: &TextMessage{
			Text: text,
		},
	}
}

func (msg *TextMessage) Clone() MessageRenderer {
	return &TextMessage{
		Text: msg.Text,
	}
}

var linkRegex = regexp.MustCompile(`https?://\S+`)

func (msg *TextMessage) getCache(uiMsg *UIMessage) tstring.TString {
	if msg.cache == nil {
		var content = tstring.NewBlankTString()
		indices := linkRegex.FindAllStringIndex(msg.Text, -1)
		var lastEnd int
		for i, item := range indices {
			start, end := item[0], item[1]
			link := msg.Text[start:end]
			linkID := fmt.Sprintf("%s-%d", msg.eventID, i)
			content = content.
				Append(msg.Text[:start]).
				AppendTString(tstring.NewStyleTString(link, tcell.StyleDefault.Hyperlink(link, linkID)))
			lastEnd = end
		}
		if lastEnd < len(msg.Text) {
			content = content.Append(msg.Text[lastEnd:])
		}
		switch uiMsg.Type {
		case "m.emote":
			prefix := tstring.NewTString("* ")
			name := tstring.NewColorTString(uiMsg.SenderName, uiMsg.SenderColor())
			msg.cache = prefix.AppendTString(name, tstring.NewTString(" "), content)
		default:
			msg.cache = content
		}
	}
	return msg.cache
}

func (msg *TextMessage) NotificationContent() string {
	return msg.Text
}

func (msg *TextMessage) PlainText() string {
	return msg.Text
}

func (msg *TextMessage) String() string {
	return fmt.Sprintf(`&messages.TextMessage{Text="%s"}`, msg.Text)
}

func (msg *TextMessage) CalculateBuffer(prefs config.UserPreferences, width int, uiMsg *UIMessage) {
	if uiMsg.IsHighlight != msg.isHighlight {
		msg.cache = nil
	}
	msg.buffer = calculateBufferWithText(prefs, msg.getCache(uiMsg), width, uiMsg)
}

func (msg *TextMessage) Height() int {
	return len(msg.buffer)
}

func (msg *TextMessage) Draw(screen mauview.Screen, _ *UIMessage) {
	for y, line := range msg.buffer {
		line.Draw(screen, 0, y)
	}
}
