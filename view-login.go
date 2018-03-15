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
	"maunium.net/go/tview"
)

func (ui *GomuksUI) NewLoginView() tview.Primitive {
	hs := ui.config.HS
	if len(hs) == 0 {
		hs = "https://matrix.org"
	}

	ui.loginView = tview.NewForm()
	ui.loginView.
		AddInputField("Homeserver", hs, 30, nil, nil).
		AddInputField("Username", ui.config.MXID, 30, nil, nil).
		AddPasswordField("Password", "", 30, '*', nil).
		AddButton("Log in", ui.login).
		AddButton("Quit", ui.gmx.Stop).
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).SetTitle("Log in to Matrix")
	return Center(45, 11, ui.loginView)
}

func (ui *GomuksUI) login() {
	hs := ui.loginView.GetFormItem(0).(*tview.InputField).GetText()
	mxid := ui.loginView.GetFormItem(1).(*tview.InputField).GetText()
	password := ui.loginView.GetFormItem(2).(*tview.InputField).GetText()

	ui.debug.Printf("Logging into %s as %s...", hs, mxid)
	ui.config.HS = hs
	ui.debug.Print("Connect result:", ui.matrix.InitClient())
	ui.debug.Print("Login result:", ui.matrix.Login(mxid, password))
}
