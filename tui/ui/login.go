package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"go.mau.fi/mauview"
	"go.mau.fi/mauview/mauview-test/debug"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

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

func (lfv *LoginFormView) resolveWellKnown(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	hsUrl := ""
	defer func(x *string) {
		lfv.homeserverField.SetPlaceholder("(will autofill)")
		lfv.homeserverField.SetText(hsUrl)
	}(&hsUrl)
	userID := id.UserID(lfv.userIDField.GetText())
	if userID == "" {
		lfv.err.SetText("Invalid user ID")
		return
	}
	_, hs, err := userID.ParseAndValidate()
	if err != nil {
		lfv.err.SetText("Invalid user ID: " + err.Error())
		return
	}
	logger.Debug().Stringer("user_id", userID).Msg("resolving homeserver from user ID")
	lfv.homeserverField.SetPlaceholder("Resolving " + hs + "...")
	lfv.homeserverField.SetText("")
	lfv.hsInputEnabled = false
	resp, err := mautrix.DiscoverClientAPI(ctx, hs)
	if err != nil {
		logger.Warn().Err(err).Stringer("user_id", userID).Msg("Failed to resolve homeserver from user ID")
		lfv.err.SetText("Failed to resolve homeserver: " + err.Error())
		return
	}
	if resp == nil {
		logger.Warn().Stringer("user_id", userID).Msg("No usable response from homeserver discovery")
		hsUrl = "https://" + hs
	} else if resp.Homeserver.BaseURL != "" {
		logger.Debug().
			Stringer("user_id", userID).
			Str("homeserver", resp.Homeserver.BaseURL).
			Msg("Resolved homeserver from user ID")
		hsUrl = resp.Homeserver.BaseURL
	}
}

func NewLoginForm(ctx context.Context, gmx *gomuks.Gomuks, app *mauview.Application) *LoginFormView {
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

		err: mauview.NewTextView().SetText("").SetTextColor(tcell.ColorRed),

		loginBtn:  mauview.NewButton("Login"),
		cancelBtn: mauview.NewButton("Cancel"),

		config: &gmx.Config,
		parent: app,
	}
	lf.loginBtn.SetOnClick(func() {
		debug.Print("login button clicked")
		gmx.Stop()
	})
	lf.cancelBtn.SetOnClick(func() {
		debug.Print("cancel button clicked")
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
		AddComponent(lf.homeserverLabel, 1, 5, 1, 1).
		AddComponent(lf.err, 1, 11, 5, 1)

	lf.center = mauview.Center(mauview.NewBox(lf).SetTitle("Log in to Matrix"), 45, 13)
	lf.SetOnFocusChanged(func(from, to mauview.Component) {
		if from == lf.userIDField {
			go lf.resolveWellKnown(ctx)
		}
	})
	return lf
}
