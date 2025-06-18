package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/tui/abstract"
)

type LoginButtons struct {
	*mauview.Flex
	parent            *LoginView
	LoginWithPassword *mauview.Button
	LoginWithSSO      *mauview.Button
	Cancel            *mauview.Button
}

func NewLoginButtons(parent *LoginView) *LoginButtons {
	buttons := &LoginButtons{
		Flex:   mauview.NewFlex().SetDirection(mauview.FlexColumn),
		parent: parent,
		Cancel: mauview.NewButton("Cancel"),
	}
	if parent.supportedAuthFlows != nil {
		if parent.supportedAuthFlows.HasFlow(mautrix.AuthTypePassword) {
			buttons.LoginWithPassword = mauview.NewButton("Log in with password")
			buttons.AddProportionalComponent(buttons.LoginWithPassword, 1)
		}
		if parent.supportedAuthFlows.HasFlow(mautrix.AuthTypeSSO) {
			buttons.LoginWithSSO = mauview.NewButton("Log in with SSO")
			buttons.LoginWithSSO.SetStyle(tcell.StyleDefault)
			// TODO: /_gomuks/sso stuff
			buttons.AddProportionalComponent(buttons.LoginWithSSO, 1)
		}
	}
	buttons.AddProportionalComponent(buttons.Cancel, 1)
	return buttons
}

type LoginView struct {
	*mauview.Form
	app       abstract.App
	Container *mauview.Centerer

	userIDField        *mauview.InputField
	homeserverURLField *mauview.InputField
	passwordInputField *mauview.InputField

	homeserver         string
	supportedAuthFlows *mautrix.RespLoginFlows
	loginButtons       *LoginButtons
}

func (l *LoginView) refreshBtns(ctx context.Context) {
	if l.loginButtons != nil {
		l.RemoveComponent(l.loginButtons)
	}
	if l.passwordInputField != nil {
		l.RemoveFormItem(l.passwordInputField)
	}
	l.loginButtons = NewLoginButtons(l)
	l.loginButtons.Cancel.SetOnClick(func() {
		l.app.Gmx().Log.Debug().Msg("Login cancelled")
		l.app.Gmx().Stop()
	})
	l.AddComponent(l.loginButtons, 0, 3, 3, 1)
	if l.loginButtons.LoginWithPassword != nil {
		l.passwordInputField = mauview.NewInputField().
			SetPlaceholder("********").
			SetMaskCharacter('*')
		l.loginButtons.LoginWithPassword.SetOnClick(func() {
			l.app.Gmx().Log.Debug().Msg("Logging in with password")
			l.passwordLogin(ctx)
		})
		l.AddFormItem(l.passwordInputField, 2, 2, 1, 1).
			AddComponent(mauview.NewTextField().SetText("Password"), 0, 2, 1, 1)
	} else {
		l.passwordInputField = nil
	}
	l.app.App().Redraw()
}

func (l *LoginView) passwordLogin(ctx context.Context) {
	userID := id.UserID(l.userIDField.GetText())
	password := l.passwordInputField.GetText()
	homeserverUrl := l.homeserver
	if homeserverUrl == "" || userID == "" || password == "" {
		return
	}
	ok, err := l.app.Rpc().Login(ctx, &jsoncmd.LoginParams{
		HomeserverURL: homeserverUrl,
		Username:      userID.Localpart(),
		Password:      password,
	})
	if err != nil {
		l.app.Gmx().Log.Err(err).Msg("Failed to log in with password")
		return
	}
	if !ok {
		l.app.Gmx().Log.Error().Msg("Login with password failed for some reason")
		return
	}
	l.app.Gmx().Log.Info().Msg("Logged in successfully with password")
	// Hopefully control.go whisks us away now
}

func (l *LoginView) resolve(ctx context.Context) {
	l.homeserverURLField.SetPlaceholder("https://example.com")
	userID := id.UserID(l.userIDField.GetText())
	if l.homeserverURLField.GetText() != "" {
		return
	}
	l.homeserverURLField.SetPlaceholder("resolving...")
	wk, err := l.app.Rpc().DiscoverHomeserver(ctx, &jsoncmd.DiscoverHomeserverParams{UserID: userID})
	if err != nil {
		l.app.Gmx().Log.Error().Err(err).Msg("Failed to resolve homeserver URL")
		l.homeserverURLField.SetPlaceholder("err")
		return
	}
	url := "https://" + userID.Homeserver()
	if wk != nil && wk.Homeserver.BaseURL != "" {
		url = wk.Homeserver.BaseURL
	}
	loginFlows, err := l.app.Rpc().GetLoginFlows(ctx, &jsoncmd.GetLoginFlowsParams{HomeserverURL: url})
	if err != nil {
		l.app.Gmx().Log.Error().Err(err).Msg("Failed to get login flows")
		l.homeserverURLField.SetPlaceholder("err")
		return
	}
	if loginFlows == nil || len(loginFlows.Flows) == 0 {
		l.app.Gmx().Log.Error().Msg("No login flows available for the given homeserver")
		l.homeserverURLField.SetPlaceholder("bad server")
		return
	}
	if !loginFlows.HasFlow(mautrix.AuthTypePassword) && !loginFlows.HasFlow(mautrix.AuthTypeSSO) {
		l.app.Gmx().Log.Error().Msg("No supported login flows available for the given homeserver")
		l.homeserverURLField.SetPlaceholder("idk how to log in")
		return
	}
	l.supportedAuthFlows = loginFlows
	l.homeserver = url
	l.homeserverURLField.SetPlaceholder(url)
	l.refreshBtns(ctx)
	l.app.App().Redraw()
}

func (l *LoginView) OnFocusChange(ctx context.Context) func(from, to mauview.Component) {
	return func(from, to mauview.Component) {
		if from == l.userIDField {
			go l.resolve(ctx)
		}
	}
}

func NewLoginView(ctx context.Context, app abstract.App) *LoginView {
	v := &LoginView{
		app:                app,
		Form:               mauview.NewForm(),
		userIDField:        mauview.NewInputField().SetPlaceholder("@user:example.com"),
		homeserverURLField: mauview.NewInputField().SetPlaceholder("https://example.com"),
	}
	v.refreshBtns(ctx)
	v.SetRows([]int{1, 1, 1, 1}).SetColumns([]int{10, 2, 50})
	// User ID: ...
	// Homeserver URL: ...
	// Password: ... (may be nil field)
	// Log in with password | log in with sso | cancel
	v.AddFormItem(v.userIDField, 2, 0, 1, 1).
		AddFormItem(v.homeserverURLField, 2, 1, 1, 1).
		AddComponent(mauview.NewTextField().SetText("User ID"), 0, 0, 1, 1).
		AddComponent(mauview.NewTextField().SetText("Homeserver URL"), 0, 1, 1, 1)
	v.SetOnFocusChanged(v.OnFocusChange(ctx))
	v.Container = mauview.Center(mauview.NewBox(v).SetBorder(true).SetTitle("Log in to Matrix"), 64, 10).SetAlwaysFocusChild(true)
	return v
}
