package ui

import (
	"context"
	"net/url"

	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type LoginFormView struct {
	*mauview.Form

	Container *mauview.Box

	userIDLabel      *mauview.TextField
	passwordLabel    *mauview.TextField
	homeserverLabel  *mauview.TextField
	recoveryKeyLabel *mauview.TextField
	// Homeserver should autofill but also be allowed to be overridden, like in the web UI.

	userIDField      *mauview.InputField
	passwordField    *mauview.InputField
	homeserverField  *mauview.InputField
	recoveryKeyField *mauview.InputField

	err *mauview.TextView

	loginBtn  *mauview.Button
	cancelBtn *mauview.Button

	app        *MainView
	loginFlows *mautrix.RespLoginFlows
}

func NewLoginForm(ctx context.Context, app *MainView) *LoginFormView {
	lf := &LoginFormView{
		Form:             mauview.NewForm(),
		userIDLabel:      mauview.NewTextField().SetText("User ID"),
		passwordLabel:    mauview.NewTextField().SetText("Password"),
		homeserverLabel:  mauview.NewTextField().SetText("Homeserver"),
		recoveryKeyLabel: mauview.NewTextField().SetText("Recovery key"),

		userIDField: mauview.NewInputField().SetPlaceholder("@username:example.com"),
		passwordField: mauview.NewInputField().
			SetPlaceholder("password1234").
			SetMaskCharacter('*'),
		homeserverField:  mauview.NewInputField().SetPlaceholder("(will autofill)"),
		recoveryKeyField: mauview.NewInputField().SetPlaceholder("ABCD EFGH IJKL MNOP QRST UVWX YZ01 2345 6789 0ABC DEFG HIJK"),

		err: mauview.NewTextView().SetText("").SetTextColor(tcell.ColorRed),

		loginBtn:  mauview.NewButton("Login"),
		cancelBtn: mauview.NewButton("Cancel"),

		app: app,
	}
	lf.loginBtn.SetOnClick(func() {
		lf.Login(ctx)
	})
	lf.cancelBtn.SetOnClick(func() {
		app.gmx.DirectStop()
	})
	lf.SetColumns([]int{1, 13, 1, 73, 1}).
		SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	lf.
		AddFormItem(lf.userIDField, 3, 1, 1, 1).
		AddFormItem(lf.passwordField, 3, 3, 1, 1).
		AddFormItem(lf.homeserverField, 3, 5, 1, 1).
		AddFormItem(lf.recoveryKeyField, 3, 7, 1, 2).
		AddFormItem(lf.loginBtn, 1, 9, 3, 1).
		AddFormItem(lf.cancelBtn, 1, 11, 3, 1).
		AddComponent(lf.userIDLabel, 1, 1, 1, 1).
		AddComponent(lf.passwordLabel, 1, 3, 1, 1).
		AddComponent(lf.homeserverLabel, 1, 5, 1, 1).
		AddComponent(lf.recoveryKeyLabel, 1, 7, 1, 2).
		AddComponent(lf.err, 1, 13, 5, 1)

	lf.Container = mauview.NewBox(lf).SetTitle("Log in to Matrix")
	lf.Container.SetKeyCaptureFunc(app.QuitOnKey())
	lf.SetOnFocusChanged(func(from, to mauview.Component) {
		if from == lf.userIDField {
			go lf.resolveWellKnown(ctx)
		}
	})
	return lf
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
	_, hs, err := userID.Parse()
	if err != nil {
		lfv.err.SetText("Invalid user ID: " + err.Error())
		return
	}
	logger.Debug().Stringer("user_id", userID).Msg("resolving homeserver from user ID")
	lfv.homeserverField.SetPlaceholder("Resolving " + hs + "...")
	lfv.homeserverField.SetText("")
	resp, err := lfv.app.rpc.DiscoverHomeserver(ctx, &jsoncmd.DiscoverHomeserverParams{UserID: userID})
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

	loginFlows, err := lfv.app.rpc.GetLoginFlows(ctx, &jsoncmd.GetLoginFlowsParams{HomeserverURL: hsUrl})
	if err != nil {
		logger.Warn().Err(err).Stringer("user_id", userID).Msg("Failed to get login flows")
		lfv.err.SetText("Failed to get login flows: " + err.Error())
		return
	}
	if loginFlows == nil || len(loginFlows.Flows) == 0 {
		logger.Warn().Stringer("user_id", userID).Msg("No login flows available for the homeserver")
		lfv.err.SetText("No login flows available for the homeserver")
		return
	}
	lfv.loginFlows = loginFlows
	if !loginFlows.HasFlow(mautrix.AuthTypePassword) {
		logger.Warn().Stringer("user_id", userID).Msg("No password login flow available for the homeserver")
		lfv.err.SetText("No password login flow available for the homeserver")
		return
	}
	lfv.homeserverField.SetText(hsUrl)
}

func (lfv *LoginFormView) Login(ctx context.Context) {
	parsedUrl, err := url.Parse(lfv.homeserverField.GetText())
	if err != nil {
		lfv.err.SetText("Invalid homeserver URL: " + err.Error())
		return
	}
	userID := id.UserID(lfv.userIDField.GetText())
	if _, _, err = userID.Parse(); err != nil {
		lfv.err.SetText("Invalid user ID: " + err.Error())
		return
	}
	password := lfv.passwordField.GetText()
	if password == "" {
		lfv.err.SetText("Password is required")
		return
	}
	recoveryKey := lfv.recoveryKeyField.GetText()
	if recoveryKey == "" {
		lfv.err.SetText("Security key is required")
		return
	}
	ok, err := lfv.app.rpc.Login(ctx, &jsoncmd.LoginParams{
		HomeserverURL: parsedUrl.String(),
		Username:      userID.Localpart(),
		Password:      password,
	})
	if err != nil {
		lfv.err.SetText("Login failed: " + err.Error())
		zerolog.Ctx(ctx).Error().Err(err).Msg("Login failed")
		return
	} else if ok {
		lfv.err.SetText("Login successful!")
		zerolog.Ctx(ctx).Info().Msg("Login successful")
	} else {
		lfv.err.SetText("Login failed: unknown error")
		zerolog.Ctx(ctx).Error().Msg("Login failed: unknown error")
		return
	}
}
