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
	"os"
	"path/filepath"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

var matrix = new(MatrixContainer)
var config = new(Config)
var debug = new(DebugPane)

func main() {
	configDir := filepath.Join(os.Getenv("HOME"), ".config/gomuks")
	os.MkdirAll(configDir, 0700)
	config.Load(configDir)

	views := tview.NewPages()
	InitUI(views)

	main := debug.Wrap(views)

	if len(config.MXID) > 0 {
		config.LoadSession(config.MXID)
	}
	matrix.Init(config)

	if err := tview.NewApplication().SetRoot(main, true).Run(); err != nil {
		panic(err)
	}
}

func InitUI(views *tview.Pages) {
	views.AddPage("login", InitLoginUI(), true, true)
}

func Center(width, height int, p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(p, height, 1, true).
		AddItem(tview.NewBox(), 0, 1, false), width, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)
}

type FormTextView struct {
	*tview.TextView
}

func (ftv *FormTextView) GetLabel() string {
	return ""
}

func (ftv *FormTextView) SetFormAttributes(label string, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	return ftv
}

func (ftv *FormTextView) GetFieldWidth() int {
	_, _, w, _ := ftv.TextView.GetRect()
	return w
}

func (ftv *FormTextView) SetFinishedFunc(handler func(key tcell.Key)) tview.FormItem {
	ftv.SetDoneFunc(handler)
	return ftv
}

func login(form *tview.Form) func() {
	return func() {
		hs := form.GetFormItem(0).(*tview.InputField).GetText()
		mxid := form.GetFormItem(1).(*tview.InputField).GetText()
		password := form.GetFormItem(2).(*tview.InputField).GetText()
		debug.Printf("%s %s %s", hs, mxid, password)
		config.HS = hs
		debug.Print(matrix.Init(config))
		debug.Print(matrix.Login(mxid, password))
	}
}

func InitLoginUI() tview.Primitive {
	form := tview.NewForm().SetButtonsAlign(tview.AlignCenter)
	hs := config.HS
	if len(hs) == 0 {
		hs = "https://matrix.org"
	}
	form.
		AddInputField("Homeserver", hs, 30, nil, nil).
		AddInputField("Username", config.MXID, 30, nil, nil).
		AddPasswordField("Password", "", 30, '*', nil).
		AddButton("Log in", login(form))
	form.SetBorder(true).SetTitle("Log in to Matrix")
	return Center(45, 13, form)
}
