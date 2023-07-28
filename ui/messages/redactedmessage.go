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
	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/matrix/muksevt"

	"maunium.net/go/gomuks/config"
)

type RedactedMessage struct{}

func NewRedactedMessage(evt *muksevt.Event, displayname string) *UIMessage {
	return newUIMessage(evt, displayname, &RedactedMessage{})
}

func (msg *RedactedMessage) Clone() MessageRenderer {
	return &RedactedMessage{}
}

func (msg *RedactedMessage) NotificationContent() string {
	return ""
}

func (msg *RedactedMessage) PlainText() string {
	return "[redacted]"
}

func (msg *RedactedMessage) String() string {
	return "&messages.RedactedMessage{}"
}

func (msg *RedactedMessage) CalculateBuffer(prefs config.UserPreferences, width int, uiMsg *UIMessage) {
}

func (msg *RedactedMessage) Height() int {
	return 1
}

const RedactionChar = '█'
const RedactionMaxWidth = 40

var RedactionStyle = tcell.StyleDefault.Foreground(tcell.NewRGBColor(50, 0, 0))

func (msg *RedactedMessage) Draw(screen mauview.Screen, _ *UIMessage) {
	w, _ := screen.Size()
	for x := 0; x < w && x < RedactionMaxWidth; x++ {
		screen.SetContent(x, 0, RedactionChar, nil, RedactionStyle)
	}
}
