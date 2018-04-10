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
	"strings"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"maunium.net/go/gomuks/interface"
)

// Message is a wrapper for the content and metadata of a Matrix message intended to be displayed.
type UIMessage interface {
	ifc.Message

	CalculateBuffer(width int)
	RecalculateBuffer()
	Buffer() []UIString
	Height() int

	RealSender() string
}

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

type UIString []Cell

func NewUIString(str string) UIString {
	newStr := make([]Cell, len(str))
	for i, char := range str {
		newStr[i] = NewCell(char)
	}
	return newStr
}

func NewColorUIString(str string, color tcell.Color) UIString {
	newStr := make([]Cell, len(str))
	for i, char := range str {
		newStr[i] = NewColorCell(char, color)
	}
	return newStr
}

func NewStyleUIString(str string, style tcell.Style) UIString {
	newStr := make([]Cell, len(str))
	for i, char := range str {
		newStr[i] = NewStyleCell(char, style)
	}
	return newStr
}

func (str UIString) Colorize(from, to int, color tcell.Color) {
	for i := from; i < to; i++ {
		str[i].Style = str[i].Style.Foreground(color)
	}
}

func (str UIString) Draw(screen tcell.Screen, x, y int) {
	offsetX := 0
	for _, cell := range str {
		offsetX += cell.Draw(screen, x+offsetX, y)
	}
}

func (str UIString) RuneWidth() (width int) {
	for _, cell := range str {
		width += runewidth.RuneWidth(cell.Char)
	}
	return width
}

func (str UIString) String() string {
	var buf strings.Builder
	for _, cell := range str {
		buf.WriteRune(cell.Char)
	}
	return buf.String()
}

// Truncate return string truncated with w cells
func (str UIString) Truncate(w int) UIString {
	if str.RuneWidth() <= w {
		return str[:]
	}
	width := 0
	i := 0
	for ; i < len(str); i++ {
		cw := runewidth.RuneWidth(str[i].Char)
		if width+cw > w {
			break
		}
		width += cw
	}
	return str[0:i]
}

func (str UIString) IndexFrom(r rune, from int) int {
	for i := from; i < len(str); i++ {
		if str[i].Char == r {
			return i
		}
	}
	return -1
}

func (str UIString) Index(r rune) int {
	return str.IndexFrom(r, 0)
}

func (str UIString) Count(r rune) (counter int) {
	index := 0
	for {
		index = str.IndexFrom(r, index)
		if index < 0 {
			break
		}
		index++
		counter++
	}
	return
}

func (str UIString) Split(sep rune) []UIString {
	a := make([]UIString, str.Count(sep)+1)
	i := 0
	orig := str
	for {
		m := orig.Index(sep)
		if m < 0 {
			break
		}
		a[i] = orig[:m]
		orig = orig[m+1:]
		i++
	}
	a[i] = orig
	return a[:i+1]
}

const DateFormat = "January _2, 2006"
const TimeFormat = "15:04:05"
