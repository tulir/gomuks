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

package tstring

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"maunium.net/go/tcell"
)

type TString []Cell

func NewBlankTString() TString {
	return make([]Cell, 0)
}

func NewTString(str string) TString {
	newStr := make([]Cell, len(str))
	for i, char := range str {
		newStr[i] = NewCell(char)
	}
	return newStr
}

func NewColorTString(str string, color tcell.Color) TString {
	newStr := make([]Cell, len(str))
	for i, char := range str {
		newStr[i] = NewColorCell(char, color)
	}
	return newStr
}

func NewStyleTString(str string, style tcell.Style) TString {
	newStr := make([]Cell, len(str))
	for i, char := range str {
		newStr[i] = NewStyleCell(char, style)
	}
	return newStr
}

func (str TString) AppendTString(data TString) TString {
	return append(str, data...)
}

func (str TString) Append(data string) TString {
	newStr := make(TString, len(str)+len(data))
	copy(newStr, str)
	for i, char := range data {
		newStr[i+len(str)] = NewCell(char)
	}
	return newStr
}

func (str TString) AppendColor(data string, color tcell.Color) TString {
	return str.AppendCustom(data, func(r rune) Cell {
		return NewColorCell(r, color)
	})
}

func (str TString) AppendStyle(data string, style tcell.Style) TString {
	return str.AppendCustom(data, func(r rune) Cell {
		return NewStyleCell(r, style)
	})
}

func (str TString) AppendCustom(data string, cellCreator func(rune) Cell) TString {
	newStr := make(TString, len(str)+len(data))
	copy(newStr, str)
	for i, char := range data {
		newStr[i+len(str)] = cellCreator(char)
	}
	return newStr
}

func (str TString) Colorize(from, length int, color tcell.Color) {
	for i := from; i < from+length; i++ {
		str[i].Style = str[i].Style.Foreground(color)
	}
}

func (str TString) Draw(screen tcell.Screen, x, y int) {
	offsetX := 0
	for _, cell := range str {
		offsetX += cell.Draw(screen, x+offsetX, y)
	}
}

func (str TString) RuneWidth() (width int) {
	for _, cell := range str {
		width += runewidth.RuneWidth(cell.Char)
	}
	return width
}

func (str TString) String() string {
	var buf strings.Builder
	for _, cell := range str {
		buf.WriteRune(cell.Char)
	}
	return buf.String()
}

// Truncate return string truncated with w cells
func (str TString) Truncate(w int) TString {
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

func (str TString) IndexFrom(r rune, from int) int {
	for i := from; i < len(str); i++ {
		if str[i].Char == r {
			return i
		}
	}
	return -1
}

func (str TString) Index(r rune) int {
	return str.IndexFrom(r, 0)
}

func (str TString) Count(r rune) (counter int) {
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

func (str TString) Split(sep rune) []TString {
	a := make([]TString, str.Count(sep)+1)
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
