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
	"github.com/rivo/tview"
)

func (ui *GomuksUI) MakeLoginUI() tview.Primitive {
	form := tview.NewForm().SetButtonsAlign(tview.AlignCenter)
	hs := ui.config.HS
	if len(hs) == 0 {
		hs = "https://matrix.org"
	}
	form.
		AddInputField("Homeserver", hs, 30, nil, nil).
		AddInputField("Username", ui.config.MXID, 30, nil, nil).
		AddPasswordField("Password", "", 30, '*', nil).
		AddButton("Log in", ui.login(form))
	form.SetBorder(true).SetTitle("Log in to Matrix")
	return Center(45, 13, form)
}

func (ui *GomuksUI) login(form *tview.Form) func() {
	return func() {
		hs := form.GetFormItem(0).(*tview.InputField).GetText()
		mxid := form.GetFormItem(1).(*tview.InputField).GetText()
		password := form.GetFormItem(2).(*tview.InputField).GetText()

		ui.debug.Printf("Logging into %s as %s...", hs, mxid)
		ui.config.HS = hs
		ui.debug.Print(ui.matrix.InitClient())
		ui.debug.Print(ui.matrix.Login(mxid, password))
	}
}
