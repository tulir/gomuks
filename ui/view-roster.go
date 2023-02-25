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

package ui

import (
	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/matrix/rooms"
	"maunium.net/go/gomuks/ui/widget"
)

type RosterView struct {
	mauview.Component

	selected *rooms.Room

	height, width int

	// The item main text color.
	mainTextColor tcell.Color
	// The text color for selected items.
	selectedTextColor tcell.Color
	// The background color for selected items.
	selectedBackgroundColor tcell.Color

	parent *MainView
}

func NewRosterView(mainView *MainView) *RosterView {
	rstr := &RosterView{
		parent: mainView,
	}

	return rstr
}

func (rstr *RosterView) Draw(screen mauview.Screen) {
	rstr.width, rstr.height = screen.Size()
	y := 0

	for _, room := range rstr.parent.rooms {
		if room.Room.IsReplaced() {
			continue
		}

		renderHeight := 1
		if y+renderHeight >= rstr.height {
			renderHeight = rstr.height - y
		}

		isSelected := room.Room == rstr.selected

		style := tcell.StyleDefault.
			Foreground(rstr.mainTextColor).
			Bold(room.Room.HasNewMessages())
		if isSelected {
			style = style.
				Foreground(rstr.selectedTextColor).
				Background(rstr.selectedBackgroundColor)
		}

		widget.WriteLinePadded(
			screen,
			mauview.AlignLeft,
			room.Room.GetTitle(),
			0, y, rstr.width,
			style,
		)

		y += renderHeight
		if y >= rstr.height {
			break
		}
	}
}
