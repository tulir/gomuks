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
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/ui/messages/tstring"
)

type ExpandedTextMessage struct {
	BaseMessage
	MsgText tstring.TString
}

// NewExpandedTextMessage creates a new ExpandedTextMessage object with the provided values and the default state.
func NewExpandedTextMessage(event *mautrix.Event, displayname string, text tstring.TString) UIMessage {
	return &ExpandedTextMessage{
		BaseMessage: newBaseMessage(event, displayname),
		MsgText:     text,
	}
}

func NewDateChangeMessage(text string) UIMessage {
	midnight := time.Now()
	midnight = time.Date(midnight.Year(), midnight.Month(), midnight.Day(),
		0, 0, 0, 0,
		midnight.Location())
	return &ExpandedTextMessage{
		BaseMessage: BaseMessage{
			MsgSenderID:  "*",
			MsgSender:    "*",
			MsgTimestamp: midnight,
			MsgIsService: true,
		},
		MsgText: tstring.NewColorTString(text, tcell.ColorGreen),
	}
}


func (msg *ExpandedTextMessage) Clone() UIMessage {
	return &ExpandedTextMessage{
		BaseMessage: msg.BaseMessage.clone(),
		MsgText:     msg.MsgText.Clone(),
	}
}

func (msg *ExpandedTextMessage) GenerateText() tstring.TString {
	return msg.MsgText
}

func (msg *ExpandedTextMessage) NotificationContent() string {
	return msg.MsgText.String()
}

func (msg *ExpandedTextMessage) PlainText() string {
	return msg.MsgText.String()
}

func (msg *ExpandedTextMessage) CalculateBuffer(prefs config.UserPreferences, width int) {
	msg.CalculateReplyBuffer(prefs, width)
	msg.calculateBufferWithText(prefs, msg.MsgText, width)
}
