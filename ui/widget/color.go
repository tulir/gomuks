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
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/gdamore/tcell"
)

var colorNames []string

func init() {
	colorNames = make([]string, len(tcell.ColorNames))
	i := 0
	for name, _ := range tcell.ColorNames {
		colorNames[i] = name
		i++
	}
	sort.Sort(sort.StringSlice(colorNames))
}

func GetHashColorName(s string) string {
	switch s {
	case "-->":
		return "green"
	case "<--":
		return "red"
	case "---":
		return "yellow"
	default:
		h := fnv.New32a()
		h.Write([]byte(s))
		return colorNames[int(h.Sum32())%len(colorNames)]
	}
}

func GetHashColor(s string) tcell.Color {
	return tcell.ColorNames[GetHashColorName(s)]
}

func AddHashColor(s string) string {
	return fmt.Sprintf("[%s]%s[white]", GetHashColorName(s), s)
}
