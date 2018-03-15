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
	"github.com/gdamore/tcell"
	"maunium.net/go/tview"
)

type Border struct {
	*tview.Box
}

func NewBorder() *Border {
	return &Border{tview.NewBox()}
}

func (border *Border) Draw(screen tcell.Screen) {
	background := tcell.StyleDefault.Background(border.GetBackgroundColor()).Foreground(border.GetBorderColor())
	x, y, width, height := border.GetRect()
	if width == 1 {
		for borderY := y; borderY < y+height; borderY++ {
			screen.SetContent(x, borderY, tview.GraphicsVertBar, nil, background)
		}
	} else if height == 1 {
		for borderX := x; borderX < x+width; borderX++ {
			screen.SetContent(borderX, y, tview.GraphicsHoriBar, nil, background)
		}
	}
}
