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

	"maunium.net/go/gomuks/beeper"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
)

type LoginView struct {
	*mauview.Form

	container *mauview.Centerer

	emailLabel *mauview.TextField
	codeLabel  *mauview.TextField

	email *mauview.InputField
	code  *mauview.InputField
	error *mauview.TextView

	loginButton *mauview.Button
	quitButton  *mauview.Button

	loading bool
	session string

	matrix ifc.MatrixContainer
	config *config.Config
	parent *GomuksUI
}

func (ui *GomuksUI) NewLoginView() mauview.Component {
	view := &LoginView{
		Form: mauview.NewForm(),

		emailLabel: mauview.NewTextField().SetText("Email"),
		codeLabel:  mauview.NewTextField().SetText("Code"),

		email: mauview.NewInputField(),
		code:  mauview.NewInputField(),

		loginButton: mauview.NewButton("Login"),
		quitButton:  mauview.NewButton("Quit"),

		matrix: ui.gmx.Matrix(),
		config: ui.gmx.Config(),
		parent: ui,
	}

	view.email.SetPlaceholder("example@example.com").SetTextColor(tcell.ColorWhite)
	view.code.SetPlaceholder("123456").SetTextColor(tcell.ColorWhite)

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
		SetColumns([]int{1, 5, 1, 30, 1}).
		SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	view.
		AddFormItem(view.email, 3, 1, 1, 1).
		AddFormItem(view.code, 3, 3, 1, 1).
		AddFormItem(view.loginButton, 1, 5, 3, 1).
		AddFormItem(view.quitButton, 1, 7, 3, 1).
		AddComponent(view.emailLabel, 1, 1, 1, 1).
		AddComponent(view.codeLabel, 1, 3, 1, 1)
	view.SetOnFocusChanged(view.focusChanged)
	view.FocusNextItem()
	ui.loginView = view

	view.container = mauview.Center(mauview.NewBox(view).SetTitle("Log in to Matrix"), 40, 11)
	view.container.SetAlwaysFocusChild(true)
	return view.container
}

func (view *LoginView) emailAuthFlow() {
	view.Error("")

	resp, err := beeper.StartLogin()
	if err != nil {
		view.code.SetText("")
		view.Error(err.Error())
		return
	}
	view.code.SetPlaceholder("Talking to Beeper servers…")
	view.parent.Render()

	err = beeper.SendLoginEmail(resp.RequestID, view.email.GetText())
	if err != nil {
		view.code.SetText("")
		view.code.SetPlaceholder("123456")
		view.Error(err.Error())
		return
	}
	view.session = resp.RequestID
	view.code.SetPlaceholder("Check your inbox…")
	view.parent.Render()
}

func (view *LoginView) focusChanged(from, to mauview.Component) {
	if from == view.email {
		go view.emailAuthFlow()
	}
}

func (view *LoginView) Error(err string) {
	if len(err) == 0 && view.error != nil {
		debug.Print("Hiding error")
		view.RemoveComponent(view.error)
		view.container.SetHeight(11)
		view.SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1})
		view.error = nil
	} else if len(err) > 0 {
		debug.Print("Showing error", err)
		if view.error == nil {
			view.error = mauview.NewTextView().SetTextColor(tcell.ColorRed)
			view.AddComponent(view.error, 1, 9, 3, 1)
		}
		view.error.SetText(err)
		errorHeight := int(math.Ceil(float64(runewidth.StringWidth(err)) / 41))
		view.container.SetHeight(12 + errorHeight)
		view.SetRow(11, errorHeight)
	}

	view.parent.Render()
}

func (view *LoginView) actuallyLogin(session, code string) {
	debug.Printf("Logging into Beeper with code %s...", code)
	view.config.HS = "https://matrix.beeper.com"

	if err := view.matrix.InitClient(false); err != nil {
		debug.Print("Init error:", err)
		view.Error(err.Error())
	} else if err = view.matrix.BeeperLogin(session, code); err != nil {
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
	code := view.code.GetText()

	view.loading = true
	view.loginButton.SetText("Logging in...")
	go view.actuallyLogin(view.session, code)
}
