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
	"maunium.net/go/tcell"
	"maunium.net/go/tview"
)

// Border is a simple tview widget that renders a horizontal or vertical bar.
//
// If the width of the box is 1, the bar will be vertical.
// If the height is 1, the bar will be horizontal.
// If the width nor the height are 1, nothing will be rendered.
type Border struct {
	*tview.Box
}

// NewBorder wraps a new tview Box into a new Border.
func NewBorder() *Border {
	return &Border{tview.NewBox()}
}

func (border *Border) Draw(screen tcell.Screen) {
	background := tcell.StyleDefault.Background(border.GetBackgroundColor()).Foreground(border.GetBorderColor())
	x, y, width, height := border.GetRect()
	if width == 1 {
		for borderY := y; borderY < y+height; borderY++ {
			screen.SetContent(x, borderY, tview.Borders.Vertical, nil, background)
		}
	} else if height == 1 {
		for borderX := x; borderX < x+width; borderX++ {
			screen.SetContent(borderX, y, tview.Borders.Horizontal, nil, background)
		}
	}
}
