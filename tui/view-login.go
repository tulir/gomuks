// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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

package tui

import (
	"math"

	"github.com/mattn/go-runewidth"

	"context"
	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
)

type LoginView struct {
	*mauview.Form

	container *mauview.Centerer

	homeserverLabel *mauview.TextField
	idLabel         *mauview.TextField
	passwordLabel   *mauview.TextField

	homeserver *mauview.InputField
	id         *mauview.InputField
	password   *mauview.InputField
	error      *mauview.TextView

	loginButton *mauview.Button
	quitButton  *mauview.Button

	loading bool

	parent *GomuksTUI
}

func (gt *GomuksTUI) NewLoginView() mauview.Component {
	view := &LoginView{
		Form: mauview.NewForm(),

		idLabel:         mauview.NewTextField().SetText("User ID"),
		passwordLabel:   mauview.NewTextField().SetText("Password"),
		homeserverLabel: mauview.NewTextField().SetText("Homeserver"),

		id:         mauview.NewInputField(),
		password:   mauview.NewInputField(),
		homeserver: mauview.NewInputField(),

		loginButton: mauview.NewButton("Login"),
		quitButton:  mauview.NewButton("Quit"),

		parent: gt,
	}

	view.homeserver.SetPlaceholder("https://example.com").SetText("").SetTextColor(tcell.ColorWhite)
	view.id.SetPlaceholder("@user:example.com").SetText("").SetTextColor(tcell.ColorWhite)
	view.password.SetPlaceholder("correct horse battery staple").SetMaskCharacter('*').SetTextColor(tcell.ColorWhite)

	view.quitButton.
		SetOnClick(gt.App.ForceStop).
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
		AddFormItem(view.id, 3, 1, 1, 1).
		AddFormItem(view.password, 3, 3, 1, 1).
		AddFormItem(view.homeserver, 3, 5, 1, 1).
		AddFormItem(view.loginButton, 1, 7, 3, 1).
		AddFormItem(view.quitButton, 1, 9, 3, 1).
		AddComponent(view.idLabel, 1, 1, 1, 1).
		AddComponent(view.passwordLabel, 1, 3, 1, 1).
		AddComponent(view.homeserverLabel, 1, 5, 1, 1)
	view.FocusNextItem()
	gt.loginView = view

	view.container = mauview.Center(mauview.NewBox(view).SetTitle("Log in to Matrix"), 45, 13)
	view.container.SetAlwaysFocusChild(true)
	return view.container
}
func (view *LoginView) Error(err string) {
	if len(err) == 0 && view.error != nil {
		view.RemoveComponent(view.error)
		view.container.SetHeight(13)
		view.SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1})
		view.error = nil
	} else if len(err) > 0 {
		if view.error == nil {
			view.error = mauview.NewTextView().SetTextColor(tcell.ColorRed)
			view.AddComponent(view.error, 1, 11, 3, 1)
		}
		view.error.SetText(err)
		errorHeight := int(math.Ceil(float64(runewidth.StringWidth(err)) / 41))
		view.container.SetHeight(14 + errorHeight)
		view.SetRow(11, errorHeight)
	}

	view.parent.App.Redraw()
}
func (view *LoginView) actuallyLogin(ctx context.Context, hs, mxid, password string) {

	view.loading = true
	view.loginButton.SetText("Logging in...")
	err := view.parent.Client.LoginPassword(ctx, hs, mxid, password)
	if err == nil {
		view.loginButton.SetText("Logged in")
	} else {
		view.Error(err.Error())
	}
	view.loading = false
	view.loginButton.SetText("Login")
}

func (view *LoginView) Login() {
	if view.loading {
		return
	}
	hs := view.homeserver.GetText()
	mxid := view.id.GetText()
	password := view.password.GetText()
	ctx := context.TODO()
	go view.actuallyLogin(ctx, hs, mxid, password)
}
