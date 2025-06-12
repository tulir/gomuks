package ui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type AuthenticateView struct {
	*mauview.Form
	Container     *mauview.Box
	passwordField *mauview.InputField
	errorField    *mauview.TextField
	submitButton  *mauview.Button

	app        *MainView
	pingTicker *time.Ticker
}

func (av *AuthenticateView) pingLoop(ctx context.Context) {
	for {
		select {
		case <-av.pingTicker.C:
			if !av.app.rpcAuthenticated {
				av.app.gmx.Log.Debug().Msg("skipping ping, not authenticated")
				break
			}
			if _, err := av.app.rpc.Ping(ctx, &jsoncmd.PingParams{LastReceivedID: av.app.rpc.LastReqID}); err != nil {
				av.app.gmx.Log.Error().Msg("failed to ping gomuks RPC: " + err.Error())
				// This is bad, do something here.
			}
		case <-ctx.Done():
			return
		}
	}
}

func (av *AuthenticateView) TryAuthenticate(ctx context.Context) {
	username := av.app.gmx.Config.Web.Username
	password := av.passwordField.GetText()
	if len(password) == 0 {
		av.errorField.SetText("password cannot be empty")
		return
	}

	av.app.gmx.Log.Debug().Str("username", username).Str("password", password).Msg("authenticating with gomuks RPC")
	if err := av.app.rpc.Authenticate(ctx, username, password); err != nil {
		av.errorField.SetText("failed to authenticate: " + err.Error())
		av.app.gmx.Log.Err(err).Msg("failed to authenticate with gomuks RPC")
		return
	}
	if err := av.app.rpc.Connect(ctx); err != nil {
		av.errorField.SetText("failed to connect: " + err.Error())
		av.app.gmx.Log.Err(err).Msg("failed to connect to gomuks RPC")
		return
	}
	av.app.rpcAuthenticated = true
	av.pingTicker.Reset(30 * time.Second) // re-start the ticker if it was stopped
}

func NewAuthenticateView(ctx context.Context, app *MainView) *AuthenticateView {
	v := &AuthenticateView{
		Form:          mauview.NewForm(),
		passwordField: mauview.NewInputField().SetPlaceholder("Password").SetMaskCharacter('*'),
		errorField:    mauview.NewTextField().SetText("").SetTextColor(tcell.ColorRed),
		app:           app,
		pingTicker:    time.NewTicker(30 * time.Second),
	}
	v.SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1})
	v.SetColumns([]int{1, 2})
	v.AddComponent(mauview.NewTextField().SetText("Password"), 1, 1, 8, 1)
	v.AddFormItem(v.passwordField, 9, 1, 24, 1)

	btn := mauview.NewButton("Submit")
	btn.SetOnClick(func() {
		v.errorField.SetText("authenticating...")
		v.TryAuthenticate(ctx)
	})
	v.AddComponent(btn, 1, 3, 11, 1)
	v.submitButton = mauview.NewButton("Submit")
	v.AddComponent(v.errorField, 1, 4, 24, 4)
	v.Container = mauview.NewBox(v).SetTitle("Sign in to Gomuks")
	v.Container.SetKeyCaptureFunc(app.QuitOnKey())
	go v.pingLoop(ctx)
	return v
}
