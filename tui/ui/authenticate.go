package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/tui/abstract"
)

// AuthenticateView is used to authenticate the Gomuks RPC
type AuthenticateView struct {
	*mauview.Form
	Container *mauview.Centerer
	ctx       context.Context
	app       abstract.App

	passwordField *mauview.InputField
	errorBody     *mauview.TextField
}

func NewAuthenticateView(ctx context.Context, app abstract.App) *AuthenticateView {
	a := &AuthenticateView{
		Form: mauview.NewForm(),
		ctx:  ctx,
		app:  app,
	}
	a.SetRows([]int{1, 1, 1, 1})
	a.SetColumns([]int{8, 2, 24})
	a.Container = mauview.Center(mauview.NewBox(a).SetTitle("Sign in to Gomuks").SetBorder(true), 36, 6).SetAlwaysFocusChild(true)

	a.passwordField = mauview.NewInputField()
	a.errorBody = mauview.NewTextField().SetText("...")

	submitButton := mauview.NewButton("Submit")
	submitButton.SetOnClick(func() {
		a.app.Gmx().Log.Debug().Msg("Authenticating...")
		a.Authenticate(ctx)
	})
	cancelButton := mauview.NewButton("Cancel")
	cancelButton.SetOnClick(func() {
		a.app.Gmx().Log.Debug().Msg("Authentication cancelled")
		a.app.Gmx().Stop()
	})

	a.AddFormItem(a.passwordField, 2, 0, 1, 1).
		AddComponent(mauview.NewTextField().SetText("Password"), 0, 0, 1, 1).
		AddComponent(a.errorBody, 0, 1, 3, 1).
		AddComponent(submitButton, 0, 2, 3, 1).
		AddComponent(cancelButton, 0, 3, 3, 1)
	a.FocusNextItem()

	return a
}

func (a *AuthenticateView) Authenticate(ctx context.Context) {
	a.errorBody.SetText("Connecting...").SetTextColor(tcell.ColorDimGrey)
	username := a.app.Gmx().Config.Web.Username
	password := a.passwordField.GetText()

	a.app.Gmx().Log.Debug().Str("username", username).Str("password", password).Msg("Authenticating...")
	err := a.app.Rpc().Authenticate(ctx, username, password)
	if err != nil {
		a.app.Gmx().Log.Err(err).Msg("Failed to authenticate")
		a.errorBody.SetText(err.Error()).SetTextColor(tcell.ColorRed)
		return
	}
	a.app.Gmx().Log.Debug().Msg("Authentication successful")
	if err = a.app.Rpc().Connect(ctx); err != nil {
		a.app.Gmx().Log.Err(err).Msg("Failed to connect to Gomuks RPC")
		a.errorBody.SetText(err.Error()).SetTextColor(tcell.ColorRed)
		return
	}
	a.errorBody.SetText("Waiting...").SetTextColor(tcell.ColorDefault)
}
