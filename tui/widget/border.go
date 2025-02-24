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

package widget

import (
	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
)

// Border is a simple tview widget that renders a horizontal or vertical bar.
//
// If the width of the box is 1, the bar will be vertical.
// If the height is 1, the bar will be horizontal.
// If the width nor the height are 1, nothing will be rendered.
type Border struct {
	Style tcell.Style
}

// NewBorder wraps a new tview Box into a new Border.
func NewBorder() *Border {
	return &Border{
		Style: tcell.StyleDefault.Foreground(mauview.Styles.BorderColor),
	}
}

func (border *Border) Draw(screen mauview.Screen) {
	width, height := screen.Size()
	if width == 1 {
		for borderY := 0; borderY < height; borderY++ {
			screen.SetContent(0, borderY, mauview.Borders.Vertical, nil, border.Style)
		}
	} else if height == 1 {
		for borderX := 0; borderX < width; borderX++ {
			screen.SetContent(borderX, 0, mauview.Borders.Horizontal, nil, border.Style)
		}
	}
}

func (border *Border) OnKeyEvent(event mauview.KeyEvent) bool {
	return false
}

func (border *Border) OnPasteEvent(event mauview.PasteEvent) bool {
	return false
}

func (border *Border) OnMouseEvent(event mauview.MouseEvent) bool {
	return false
}
