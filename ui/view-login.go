// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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
	"math"

	"github.com/mattn/go-runewidth"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
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
	error      *mauview.TextView

	loginButton *mauview.Button
	quitButton  *mauview.Button

	loading bool

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
	view.homeserver.SetPlaceholder("https://example.com").SetText(hs).SetTextColor(tcell.ColorWhite)
	view.username.SetPlaceholder("@user:example.com").SetText(string(ui.gmx.Config().UserID)).SetTextColor(tcell.ColorWhite)
	view.password.SetPlaceholder("correct horse battery staple").SetMaskCharacter('*').SetTextColor(tcell.ColorWhite)

	view.quitButton.
		SetOnClick(func() { ui.gmx.Stop(true) }).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetForegroundColor(tcell.ColorWhite).
		SetFocusedForegroundColor(tcell.ColorWhite)
	view.loginButton.
		SetOnClick(view.Login).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetForegroundColor(tcell.ColorWhite).
		SetFocusedForegroundColor(tcell.ColorWhite)

	view.
		SetColumns([]int{1, 10, 1, 30, 1}).
		SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	view.
		AddFormItem(view.username, 3, 1, 1, 1).
		AddFormItem(view.password, 3, 3, 1, 1).
		AddFormItem(view.homeserver, 3, 5, 1, 1).
		AddFormItem(view.loginButton, 1, 7, 3, 1).
		AddFormItem(view.quitButton, 1, 9, 3, 1).
		AddComponent(view.usernameLabel, 1, 1, 1, 1).
		AddComponent(view.passwordLabel, 1, 3, 1, 1).
		AddComponent(view.homeserverLabel, 1, 5, 1, 1)
	view.SetOnFocusChanged(view.focusChanged)
	view.FocusNextItem()
	ui.loginView = view

	view.container = mauview.Center(mauview.NewBox(view).SetTitle("Log in to Matrix"), 45, 13)
	view.container.SetAlwaysFocusChild(true)
	return view.container
}

func (view *LoginView) resolveWellKnown() {
	_, homeserver, err := id.UserID(view.username.GetText()).Parse()
	if err != nil {
		return
	}
	view.homeserver.SetText("Resolving...")
	resp, err := mautrix.DiscoverClientAPI(homeserver)
	if err != nil {
		view.homeserver.SetText("")
		view.Error(err.Error())
	} else if resp != nil {
		view.homeserver.SetText(resp.Homeserver.BaseURL)
		view.parent.Render()
	}
}

func (view *LoginView) focusChanged(from, to mauview.Component) {
	if from == view.username && view.homeserver.GetText() == "" {
		go view.resolveWellKnown()
	}
}

func (view *LoginView) Error(err string) {
	if len(err) == 0 && view.error != nil {
		debug.Print("Hiding error")
		view.RemoveComponent(view.error)
		view.container.SetHeight(13)
		view.SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1})
		view.error = nil
	} else if len(err) > 0 {
		debug.Print("Showing error", err)
		if view.error == nil {
			view.error = mauview.NewTextView().SetTextColor(tcell.ColorRed)
			view.AddComponent(view.error, 1, 11, 3, 1)
		}
		view.error.SetText(err)
		errorHeight := int(math.Ceil(float64(runewidth.StringWidth(err)) / 41))
		view.container.SetHeight(14 + errorHeight)
		view.SetRow(11, errorHeight)
	}

	view.parent.Render()
}

func (view *LoginView) actuallyLogin(hs, mxid, password string) {
	debug.Printf("Logging into %s as %s...", hs, mxid)
	view.config.HS = hs

	if err := view.matrix.InitClient(false); err != nil {
		debug.Print("Init error:", err)
		view.Error(err.Error())
	} else if err = view.matrix.Login(mxid, password); err != nil {
		if httpErr, ok := err.(mautrix.HTTPError); ok {
			if httpErr.RespError != nil && len(httpErr.RespError.Err) > 0 {
				view.Error(httpErr.RespError.Err)
			} else if len(httpErr.Message) > 0 {
				view.Error(httpErr.Message)
			} else {
				view.Error(err.Error())
			}
		} else {
			view.Error(err.Error())
		}
		debug.Print("Login error:", err)
	}
	view.loading = false
	view.loginButton.SetText("Login")
}

func (view *LoginView) Login() {
	if view.loading {
		return
	}
	hs := view.homeserver.GetText()
	mxid := view.username.GetText()
	password := view.password.GetText()

	view.loading = true
	view.loginButton.SetText("Logging in...")
	go view.actuallyLogin(hs, mxid, password)
}
