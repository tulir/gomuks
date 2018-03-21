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

package ui

import (
	"maunium.net/go/gomuks/ui/debug"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tview"
)

func (ui *GomuksUI) NewLoginView() tview.Primitive {
	hs := ui.gmx.Config().HS
	if len(hs) == 0 {
		hs = "https://matrix.org"
	}

	homeserver := widget.NewAdvancedInputField().SetLabel("Homeserver").SetText(hs).SetFieldWidth(30)
	username := widget.NewAdvancedInputField().SetLabel("Username").SetText(ui.gmx.Config().UserID).SetFieldWidth(30)
	password := widget.NewAdvancedInputField().SetLabel("Password").SetMaskCharacter('*').SetFieldWidth(30)

	ui.loginView = tview.NewForm()
	ui.loginView.
		AddFormItem(homeserver).AddFormItem(username).AddFormItem(password).
		AddButton("Log in", ui.login).
		AddButton("Quit", ui.gmx.Stop).
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).SetTitle("Log in to Matrix")
	return widget.Center(45, 11, ui.loginView)
}

func (ui *GomuksUI) login() {
	hs := ui.loginView.GetFormItem(0).(*widget.AdvancedInputField).GetText()
	mxid := ui.loginView.GetFormItem(1).(*widget.AdvancedInputField).GetText()
	password := ui.loginView.GetFormItem(2).(*widget.AdvancedInputField).GetText()

	debug.Printf("Logging into %s as %s...", hs, mxid)
	ui.gmx.Config().HS = hs
	mx := ui.gmx.MatrixContainer()
	debug.Print("Connect result:", mx.InitClient())
	debug.Print("Login result:", mx.Login(mxid, password))
}
