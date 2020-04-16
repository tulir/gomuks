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
	"maunium.net/go/gomuks/matrix/muksevt"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/ui/messages/html"
)

type HTMLMessage struct {
	Root      html.Entity
	FocusedBg tcell.Color
	TextColor tcell.Color
	focused   bool
}

func NewHTMLMessage(evt *muksevt.Event, displayname string, root html.Entity) *UIMessage {
	return newUIMessage(evt, displayname, &HTMLMessage{
		Root: root,
	})
}

func (hw *HTMLMessage) Clone() MessageRenderer {
	return &HTMLMessage{
		Root:      hw.Root.Clone(),
		FocusedBg: hw.FocusedBg,
	}
}

func (hw *HTMLMessage) Draw(screen mauview.Screen) {
	if hw.focused {
		screen.SetStyle(tcell.StyleDefault.Background(hw.FocusedBg).Foreground(hw.TextColor))
	}
	if hw.TextColor != tcell.ColorDefault {
		hw.Root.AdjustStyle(func(style tcell.Style) tcell.Style {
			fg, _, _ := style.Decompose()
			if fg == tcell.ColorDefault {
				return style.Foreground(hw.TextColor)
			}
			return style
		})
	}
	screen.Clear()
	hw.Root.Draw(screen)
}

func (hw *HTMLMessage) Focus() {
	hw.focused = true
}

func (hw *HTMLMessage) Blur() {
	hw.focused = false
}

func (hw *HTMLMessage) OnKeyEvent(event mauview.KeyEvent) bool {
	return false
}

func (hw *HTMLMessage) OnMouseEvent(event mauview.MouseEvent) bool {
	return false
}

func (hw *HTMLMessage) OnPasteEvent(event mauview.PasteEvent) bool {
	return false
}

func (hw *HTMLMessage) CalculateBuffer(preferences config.UserPreferences, width int, msg *UIMessage) {
	if width < 2 {
		return
	}
	// TODO account for bare messages in initial startX
	startX := 0
	hw.TextColor = msg.TextColor()
	hw.Root.CalculateBuffer(width, startX, preferences.BareMessageView)
}

func (hw *HTMLMessage) Height() int {
	return hw.Root.Height()
}

func (hw *HTMLMessage) PlainText() string {
	return hw.Root.PlainText()
}

func (hw *HTMLMessage) NotificationContent() string {
	return hw.Root.PlainText()
}

func (hw *HTMLMessage) String() string {
	return hw.Root.String()
}
