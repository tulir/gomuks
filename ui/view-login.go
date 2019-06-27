// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
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

package ui

import (
	"maunium.net/go/tcell"

	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
)

type LoginView struct {
	*mauview.Form

	container *mauview.Centerer

	homeserverLabel *mauview.TextField
	usernameLabel   *mauview.TextField
	passwordLabel   *mauview.TextField

	homeserver *mauview.InputField
	username   *mauview.InputField
	password   *mauview.InputField
	error      *mauview.TextField

	loginButton *mauview.Button
	quitButton  *mauview.Button

	matrix ifc.MatrixContainer
	config *config.Config
	parent *GomuksUI
}

func (ui *GomuksUI) NewLoginView() mauview.Component {
	view := &LoginView{
		Form: mauview.NewForm(),

		usernameLabel:   mauview.NewTextField().SetText("Username"),
		passwordLabel:   mauview.NewTextField().SetText("Password"),
		homeserverLabel: mauview.NewTextField().SetText("Homeserver"),

		username:   mauview.NewInputField(),
		password:   mauview.NewInputField(),
		homeserver: mauview.NewInputField(),

		loginButton: mauview.NewButton("Login"),
		quitButton:  mauview.NewButton("Quit"),

		matrix: ui.gmx.Matrix(),
		config: ui.gmx.Config(),
		parent: ui,
	}

	hs := ui.gmx.Config().HS
	view.homeserver.SetText(hs)
	view.username.SetText(ui.gmx.Config().UserID)
	view.password.SetMaskCharacter('*')

	view.quitButton.SetOnClick(func() { ui.gmx.Stop(true) }).SetBackgroundColor(tcell.ColorDarkCyan)
	view.loginButton.SetOnClick(view.Login).SetBackgroundColor(tcell.ColorDarkCyan)

	view.SetColumns([]int{1, 10, 1, 9, 1, 9, 1, 10, 1})
	view.SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	view.AddFormItem(view.username, 3, 1, 5, 1).
		AddFormItem(view.password, 3, 3, 5, 1).
		AddFormItem(view.homeserver, 3, 5, 5, 1).
		AddFormItem(view.loginButton, 5, 7, 3, 1).
		AddFormItem(view.quitButton, 1, 7, 3, 1).
		AddComponent(view.usernameLabel, 1, 1, 1, 1).
		AddComponent(view.passwordLabel, 1, 3, 1, 1).
		AddComponent(view.homeserverLabel, 1, 5, 1, 1)
	view.FocusNextItem()
	ui.loginView = view

	view.container = mauview.Center(mauview.NewBox(view).SetTitle("Log in to Matrix"), 45, 13)
	view.container.SetAlwaysFocusChild(true)
	return view.container
}

func (view *LoginView) Error(err string) {
	if len(err) == 0 {
		debug.Print("Hiding error")
		view.RemoveComponent(view.error)
		view.error = nil
		return
	}
	debug.Print("Showing error", err)
	if view.error == nil {
		view.error = mauview.NewTextField().SetTextColor(tcell.ColorRed)
		view.AddComponent(view.error, 1, 9, 7, 1)
	}
	view.error.SetText(err)

	view.parent.Render()
}

func (view *LoginView) Login() {
	hs := view.homeserver.GetText()
	mxid := view.username.GetText()
	password := view.password.GetText()

	debug.Printf("Logging into %s as %s...", hs, mxid)
	view.config.HS = hs
	err := view.matrix.InitClient()
	if err != nil {
		debug.Print("Init error:", err)
	}
	err = view.matrix.Login(mxid, password)
	if err != nil {
		if httpErr, ok := err.(mautrix.HTTPError); ok {
			if httpErr.RespError != nil {
				view.Error(httpErr.RespError.Err)
			} else {
				view.Error(httpErr.Message)
			}
		} else {
			view.Error("Failed to connect to server.")
		}
		debug.Print("Login error:", err)
	}
}
