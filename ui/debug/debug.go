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

package debug

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"runtime/debug"

	"maunium.net/go/tview"
)

type Printer interface {
	Printf(text string, args ...interface{})
	Print(text ...interface{})
}

type Pane struct {
	*tview.TextView
	Height int
	Width  int
	num    int
}

var Default Printer
var RedirectAllExt bool

func NewPane() *Pane {
	pane := tview.NewTextView()
	pane.
		SetScrollable(true).
		SetWrap(true).
		SetBorder(true).
		SetTitle("Debug output")
	fmt.Fprintln(pane, "[0] Debug pane initialized")

	return &Pane{
		TextView: pane,
		Height:   35,
		Width:    80,
		num:      0,
	}
}

func (db *Pane) Printf(text string, args ...interface{}) {
	db.WriteString(fmt.Sprintf(text, args...) + "\n")
}

func (db *Pane) Print(text ...interface{}) {
	db.WriteString(fmt.Sprintln(text...))
}

func (db *Pane) WriteString(text string) {
	db.num++
	fmt.Fprintf(db, "[%d] %s", db.num, text)
}

type PaneSide int

const (
	Top PaneSide = iota
	Bottom
	Left
	Right
)

func (db *Pane) Wrap(main tview.Primitive, side PaneSide) tview.Primitive {
	rows, columns := []int{0}, []int{0}
	mainRow, mainColumn, paneRow, paneColumn := 0, 0, 0, 0
	switch side {
	case Top:
		rows = []int{db.Height, 0}
		mainRow = 1
	case Bottom:
		rows = []int{0, db.Height}
		paneRow = 1
	case Left:
		columns = []int{db.Width, 0}
		mainColumn = 1
	case Right:
		columns = []int{0, db.Width}
		paneColumn = 1
	}
	return tview.NewGrid().SetRows(rows...).SetColumns(columns...).
		AddItem(main, mainRow, mainColumn, 1, 1, 1, 1, true).
		AddItem(db, paneRow, paneColumn, 1, 1, 1, 1, false)
}

func Printf(text string, args ...interface{}) {
	if RedirectAllExt {
		ExtPrintf(text, args...)
	} else if Default != nil {
		Default.Printf(text, args...)
	}
}

func Print(text ...interface{}) {
	if RedirectAllExt {
		ExtPrint(text...)
	} else if Default != nil {
		Default.Print(text...)
	}
}

const Oops = ` __________
< Oh noes! >
 ‾‾‾\‾‾‾‾‾‾
     \   ^__^
      \  (XX)\_______
         (__)\       )\/\
          U  ||----W |
             ||     ||`

func PrettyPanic() {
	fmt.Println(Oops)
	fmt.Println("")
	fmt.Println("A fatal error has occurred.")
	fmt.Println("")
	traceFile := fmt.Sprintf("/tmp/gomuks-panic-%s.txt", time.Now().Format("2006-01-02--15-04-05"))
	data := debug.Stack()
	err := ioutil.WriteFile(traceFile, data, 0644)
	if err != nil {
		fmt.Println("Saving the stack trace to", traceFile, "failed:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println(err)
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("")
		fmt.Println("You can file an issue at https://github.com/tulir/gomuks/issues.")
		fmt.Println("Please provide the file save error (above) and the stack trace of the original error (below) when filing an issue.")
		fmt.Println("")
		fmt.Println("--------------------------------------------------------------------------------")
		debug.PrintStack()
		fmt.Println("--------------------------------------------------------------------------------")
	} else {
		fmt.Println("The stack trace has been saved to", traceFile)
		fmt.Println("")
		fmt.Println("You can file an issue at https://github.com/tulir/gomuks/issues.")
		fmt.Println("Please provide the contents of that file when filing an issue.")
	}
	os.Exit(1)
}
