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
	"time"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/matrix/muksevt"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

type ExpandedTextMessage struct {
	Text   tstring.TString
	buffer []tstring.TString
}

// NewExpandedTextMessage creates a new ExpandedTextMessage object with the provided values and the default state.
func NewExpandedTextMessage(evt *muksevt.Event, displayname string, text tstring.TString) *UIMessage {
	return newUIMessage(evt, displayname, &ExpandedTextMessage{
		Text: text,
	})
}

func NewServiceMessage(text string) *UIMessage {
	return &UIMessage{
		SenderID:   "*",
		SenderName: "*",
		Timestamp:  time.Now(),
		IsService:  true,
		Renderer: &ExpandedTextMessage{
			Text: tstring.NewTString(text),
		},
	}
}

func NewDateChangeMessage(text string) *UIMessage {
	midnight := time.Now()
	midnight = time.Date(midnight.Year(), midnight.Month(), midnight.Day(),
		0, 0, 0, 0,
		midnight.Location())
	return &UIMessage{
		SenderID:   "*",
		SenderName: "*",
		Timestamp:  midnight,
		IsService:  true,
		Renderer: &ExpandedTextMessage{
			Text: tstring.NewColorTString(text, tcell.ColorGreen),
		},
	}
}

func (msg *ExpandedTextMessage) Clone() MessageRenderer {
	return &ExpandedTextMessage{
		Text: msg.Text.Clone(),
	}
}

func (msg *ExpandedTextMessage) NotificationContent() string {
	return msg.Text.String()
}

func (msg *ExpandedTextMessage) PlainText() string {
	return msg.Text.String()
}

func (msg *ExpandedTextMessage) String() string {
	return fmt.Sprintf(`&messages.ExpandedTextMessage{Text="%s"}`, msg.Text.String())
}

func (msg *ExpandedTextMessage) CalculateBuffer(prefs config.UserPreferences, width int, uiMsg *UIMessage) {
	msg.buffer = calculateBufferWithText(prefs, msg.Text, width, uiMsg)
}

func (msg *ExpandedTextMessage) Height() int {
	return len(msg.buffer)
}

func (msg *ExpandedTextMessage) Draw(screen mauview.Screen, _ *UIMessage) {
	for y, line := range msg.buffer {
		line.Draw(screen, 0, y)
	}
}
