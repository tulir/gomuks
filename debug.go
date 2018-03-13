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
	"fmt"

	"github.com/rivo/tview"
)

type DebugPane struct {
	text string
	pane *tview.TextView
	num  int
}

func (db *DebugPane) Printf(text string, args ...interface{}) {
	db.num++
	db.Write(fmt.Sprintf("[%d] %s\n", db.num, fmt.Sprintf(text, args...)))
}

func (db *DebugPane) Print(text ...interface{}) {
	db.num++
	db.Write(fmt.Sprintf("[%d] %s", db.num, fmt.Sprintln(text...)))
}

func (db *DebugPane) Write(text string) {
	if db.pane != nil {
		db.text += text
		db.pane.SetText(db.text)
	}
}

func (db *DebugPane) Wrap(main *tview.Pages) tview.Primitive {
	db.pane = tview.NewTextView()
	db.pane.SetBorder(true).SetTitle("Debug output")
	db.text += "[0] Debug pane initialized\n"
	db.pane.SetText(db.text)
	return tview.NewGrid().SetRows(0, 20).SetColumns(0).
		AddItem(main, 0, 0, 1, 1, 1, 1, true).
		AddItem(db.pane, 1, 0, 1, 1, 1, 1, false)
}
