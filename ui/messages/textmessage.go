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

package messages

import (
	"fmt"
	"time"

	"maunium.net/go/gomuks/matrix/event"
	"maunium.net/go/mauview"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

type TextMessage struct {
	cache       tstring.TString
	buffer      []tstring.TString
	isHighlight bool
	Text        string
}

// NewTextMessage creates a new UITextMessage object with the provided values and the default state.
func NewTextMessage(evt *event.Event, displayname string, text string) *UIMessage {
	return newUIMessage(evt, displayname, &TextMessage{
		Text: text,
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

func (msg *TextMessage) getCache(uiMsg *UIMessage) tstring.TString {
	if msg.cache == nil {
		switch uiMsg.Type {
		case "m.emote":
			msg.cache = tstring.NewColorTString(fmt.Sprintf("* %s %s", uiMsg.SenderName, msg.Text), uiMsg.TextColor())
			msg.cache.Colorize(0, len(uiMsg.SenderName)+2, uiMsg.SenderColor())
		default:
			msg.cache = tstring.NewColorTString(msg.Text, uiMsg.TextColor())
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

func (msg *TextMessage) Draw(screen mauview.Screen) {
	for y, line := range msg.buffer {
		line.Draw(screen, 0, y)
	}
}
