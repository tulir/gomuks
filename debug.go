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

	"maunium.net/go/tview"
)

const DebugPaneHeight = 35

type DebugPrinter interface {
	Printf(text string, args ...interface{})
	Print(text ...interface{})
}

type DebugPane struct {
	pane *tview.TextView
	num  int
	gmx  Gomuks
}

func NewDebugPane(gmx Gomuks) *DebugPane {
	pane := tview.NewTextView()
	pane.
		SetScrollable(true).
		SetWrap(true)
	pane.SetChangedFunc(func() {
		gmx.App().Draw()
	})
	pane.SetBorder(true).SetTitle("Debug output")
	fmt.Fprintln(pane, "[0] Debug pane initialized")

	return &DebugPane{
		pane: pane,
		num:  0,
		gmx:  gmx,
	}
}

func (db *DebugPane) Printf(text string, args ...interface{}) {
	db.Write(fmt.Sprintf(text, args...) + "\n")
}

func (db *DebugPane) Print(text ...interface{}) {
	db.Write(fmt.Sprintln(text...))
}

func (db *DebugPane) Write(text string) {
	if db.pane != nil {
		db.num++
		fmt.Fprintf(db.pane, "[%d] %s", db.num, text)
	}
}

func (db *DebugPane) Wrap(main tview.Primitive) tview.Primitive {
	return tview.NewGrid().SetRows(0, DebugPaneHeight).SetColumns(0).
		AddItem(main, 0, 0, 1, 1, 1, 1, true).
		AddItem(db.pane, 1, 0, 1, 1, 1, 1, false)
}
