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

package messages

import (
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

type Cell struct {
	Char  rune
	Style tcell.Style
}

func NewStyleCell(char rune, style tcell.Style) Cell {
	return Cell{char, style}
}

func NewColorCell(char rune, color tcell.Color) Cell {
	return Cell{char, tcell.StyleDefault.Foreground(color)}
}

func NewCell(char rune) Cell {
	return Cell{char, tcell.StyleDefault}
}

func (cell Cell) RuneWidth() int {
	return runewidth.RuneWidth(cell.Char)
}

func (cell Cell) Draw(screen tcell.Screen, x, y int) (chWidth int) {
	chWidth = cell.RuneWidth()
	for runeWidthOffset := 0; runeWidthOffset < chWidth; runeWidthOffset++ {
		screen.SetContent(x+runeWidthOffset, y, cell.Char, nil, cell.Style)
	}
	return
}
