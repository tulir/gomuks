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
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"maunium.net/go/tview"
)

func WriteLineSimple(screen tcell.Screen, line string, x, y int) {
	WriteLine(screen, tview.AlignLeft, line, x, y, 1<<30, tcell.StyleDefault)
}

func WriteLineSimpleColor(screen tcell.Screen, line string, x, y int, color tcell.Color) {
	WriteLine(screen, tview.AlignLeft, line, x, y, 1<<30, tcell.StyleDefault.Foreground(color))
}

func WriteLineColor(screen tcell.Screen, align int, line string, x, y, maxWidth int, color tcell.Color) {
	WriteLine(screen, align, line, x, y, maxWidth, tcell.StyleDefault.Foreground(color))
}

func WriteLine(screen tcell.Screen, align int, line string, x, y, maxWidth int, style tcell.Style) {
	offsetX := 0
	if align == tview.AlignRight {
		offsetX = maxWidth - runewidth.StringWidth(line)
	}
	if offsetX < 0 {
		offsetX = 0
	}
	for _, ch := range line {
		chWidth := runewidth.RuneWidth(ch)
		if chWidth == 0 {
			continue
		}

		for localOffset := 0; localOffset < chWidth; localOffset++ {
			screen.SetContent(x+offsetX+localOffset, y, ch, nil, style)
		}
		offsetX += chWidth
		if offsetX > maxWidth {
			break
		}
	}
}
