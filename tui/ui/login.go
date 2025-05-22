package ui

import (
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/gomuks"
)

type LoginFormView struct {
	*mauview.Form

	center *mauview.Centerer

	userIDLabel     *mauview.TextField
	passwordLabel   *mauview.TextField
	homeserverLabel *mauview.TextField
	// Homeserver should autofill but also be allowed to be overridden, like in the web UI.

	userIDField     *mauview.InputField
	passwordField   *mauview.InputField
	homeserverField *mauview.InputField

	err            *mauview.TextView
	hsInputEnabled bool // Should be false until the user filled in a fully qualified user ID.

	loginBtn  *mauview.Button
	cancelBtn *mauview.Button

	config *gomuks.Config
	parent *mauview.Application
}

func NewLoginForm(gmx *gomuks.Gomuks, app *mauview.Application) *LoginFormView {
	lf := &LoginFormView{
		Form:            mauview.NewForm(),
		userIDLabel:     mauview.NewTextField().SetText("User ID"),
		passwordLabel:   mauview.NewTextField().SetText("Password"),
		homeserverLabel: mauview.NewTextField().SetText("Homeserver"),

		userIDField: mauview.NewInputField().SetPlaceholder("@username:example.com"),
		passwordField: mauview.NewInputField().
			SetPlaceholder("password1234").
			SetMaskCharacter('*'),
		homeserverField: mauview.NewInputField().SetPlaceholder("(will autofill)"),

		err:            mauview.NewTextView().SetText(""),
		hsInputEnabled: false,

		loginBtn:  mauview.NewButton("Login"),
		cancelBtn: mauview.NewButton("Cancel"),

		config: &gmx.Config,
		parent: app,
	}
	lf.loginBtn.SetOnClick(func() {
		println("login button clicked")
		gmx.Stop()
	})
	lf.cancelBtn.SetOnClick(func() {
		println("cancel button clicked")
		gmx.Stop()
	})
	lf.SetColumns([]int{1, 10, 1, 30, 1}).
		SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	lf.
		AddFormItem(lf.userIDField, 3, 1, 1, 1).
		AddFormItem(lf.passwordField, 3, 3, 1, 1).
		AddFormItem(lf.homeserverField, 3, 5, 1, 1).
		AddFormItem(lf.loginBtn, 1, 7, 3, 1).
		AddFormItem(lf.cancelBtn, 1, 9, 3, 1).
		AddComponent(lf.userIDLabel, 1, 1, 1, 1).
		AddComponent(lf.passwordLabel, 1, 3, 1, 1).
		AddComponent(lf.homeserverLabel, 1, 5, 1, 1)

	lf.center = mauview.Center(mauview.NewBox(lf).SetTitle("Log in to Matrix"), 45, 13)

	return lf
}
