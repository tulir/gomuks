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

package tstring

import (
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
)

type TString []Cell

func NewBlankTString() TString {
	return make(TString, 0)
}

func NewTString(str string) TString {
	newStr := make(TString, len(str))
	for i, char := range str {
		newStr[i] = NewCell(char)
	}
	return newStr
}

func NewColorTString(str string, color tcell.Color) TString {
	newStr := make(TString, len(str))
	for i, char := range str {
		newStr[i] = NewColorCell(char, color)
	}
	return newStr
}

func NewStyleTString(str string, style tcell.Style) TString {
	newStr := make(TString, len(str))
	for i, char := range str {
		newStr[i] = NewStyleCell(char, style)
	}
	return newStr
}

func Join(strings []TString, separator string) TString {
	if len(strings) == 0 {
		return NewBlankTString()
	}

	out := strings[0]
	strings = strings[1:]

	if len(separator) == 0 {
		return out.AppendTString(strings...)
	}

	for _, str := range strings {
		out = append(out, str.Prepend(separator)...)
	}
	return out
}

func (str TString) Clone() TString {
	newStr := make(TString, len(str))
	copy(newStr, str)
	return newStr
}

func (str TString) AppendTString(dataList ...TString) TString {
	newStr := str
	for _, data := range dataList {
		newStr = append(newStr, data...)
	}
	return newStr
}

func (str TString) PrependTString(data TString) TString {
	return append(data, str...)
}

func (str TString) Append(data string) TString {
	return str.AppendCustom(data, func(r rune) Cell {
		return NewCell(r)
	})
}

func (str TString) TrimSpace() TString {
	return str.Trim(unicode.IsSpace)
}

func (str TString) Trim(fn func(rune) bool) TString {
	return str.TrimLeft(fn).TrimRight(fn)
}

func (str TString) TrimLeft(fn func(rune) bool) TString {
	for index, cell := range str {
		if !fn(cell.Char) {
			return append(NewBlankTString(), str[index:]...)
		}
	}
	return NewBlankTString()
}

func (str TString) TrimRight(fn func(rune) bool) TString {
	for i := len(str) - 1; i >= 0; i-- {
		if !fn(str[i].Char) {
			return append(NewBlankTString(), str[:i+1]...)
		}
	}
	return NewBlankTString()
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

func (str TString) Prepend(data string) TString {
	return str.PrependCustom(data, func(r rune) Cell {
		return NewCell(r)
	})
}

func (str TString) PrependColor(data string, color tcell.Color) TString {
	return str.PrependCustom(data, func(r rune) Cell {
		return NewColorCell(r, color)
	})
}

func (str TString) PrependStyle(data string, style tcell.Style) TString {
	return str.PrependCustom(data, func(r rune) Cell {
		return NewStyleCell(r, style)
	})
}

func (str TString) PrependCustom(data string, cellCreator func(rune) Cell) TString {
	newStr := make(TString, len(str)+len(data))
	copy(newStr[len(data):], str)
	for i, char := range data {
		newStr[i] = cellCreator(char)
	}
	return newStr
}

func (str TString) Colorize(from, length int, color tcell.Color) {
	str.AdjustStyle(from, length, func(style tcell.Style) tcell.Style {
		return style.Foreground(color)
	})
}

func (str TString) AdjustStyle(from, length int, fn func(tcell.Style) tcell.Style) {
	for i := from; i < from+length; i++ {
		str[i].Style = fn(str[i].Style)
	}
}

func (str TString) AdjustStyleFull(fn func(tcell.Style) tcell.Style) {
	str.AdjustStyle(0, len(str), fn)
}

func (str TString) Draw(screen mauview.Screen, x, y int) {
	for _, cell := range str {
		x += cell.Draw(screen, x, y)
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
