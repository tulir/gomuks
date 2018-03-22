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
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/interface"
	"maunium.net/go/gomuks/ui/debug"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/tview"
)

type LoginView struct {
	*tview.Form

	homeserver *widget.AdvancedInputField
	username   *widget.AdvancedInputField
	password   *widget.AdvancedInputField

	matrix ifc.MatrixContainer
	config *config.Config
	parent *GomuksUI
}

func (ui *GomuksUI) NewLoginView() tview.Primitive {
	view := &LoginView{
		Form: tview.NewForm(),

		homeserver: widget.NewAdvancedInputField(),
		username:   widget.NewAdvancedInputField(),
		password:   widget.NewAdvancedInputField(),

		matrix: ui.gmx.MatrixContainer(),
		config: ui.gmx.Config(),
		parent: ui,
	}
	hs := ui.gmx.Config().HS
	if len(hs) == 0 {
		hs = "https://matrix.org"
	}
	view.homeserver.SetLabel("Homeserver").SetText(hs).SetFieldWidth(30)
	view.username.SetLabel("Username").SetText(ui.gmx.Config().UserID).SetFieldWidth(30)
	view.password.SetLabel("Password").SetMaskCharacter('*').SetFieldWidth(30)

	view.
		AddFormItem(view.homeserver).AddFormItem(view.username).AddFormItem(view.password).
		AddButton("Log in", view.Login).
		AddButton("Quit", ui.gmx.Stop).
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).SetTitle("Log in to Matrix")

	ui.loginView = view

	return widget.Center(45, 11, ui.loginView)
}

func (view *LoginView) Login() {
	hs := view.homeserver.GetText()
	mxid := view.username.GetText()
	password := view.password.GetText()

	debug.Printf("Logging into %s as %s...", hs, mxid)
	view.config.HS = hs
	debug.Print("Connect result:", view.matrix.InitClient())
	debug.Print("Login result:", view.matrix.Login(mxid, password))
}
