// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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
	"context"
	"errors"
	"os"
	"time"

	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/tui/ui"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/rpc"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
)

type GomuksTUI struct {
	*gomuks.Gomuks
	rpc        *rpc.GomuksRPC
	app        *mauview.Application
	pingTicker *time.Ticker

	initSyncDone   bool
	clientState    *jsoncmd.ClientState
	imageAuthToken string

	authView   *ui.AuthenticateView
	syncView   *ui.SyncingView
	loginView  *ui.LoginView
	verifyView *ui.VerifySessionView
	mainView   *ui.MainView

	rooms map[id.RoomID]*jsoncmd.SyncRoom
}

func New(gmx *gomuks.Gomuks) *GomuksTUI {
	return &GomuksTUI{
		Gomuks:      gmx,
		rooms:       make(map[id.RoomID]*jsoncmd.SyncRoom),
		clientState: &jsoncmd.ClientState{},
	}
}

func init() {
	mauview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	mauview.Styles.PrimaryTextColor = tcell.ColorDefault
	mauview.Styles.BorderColor = tcell.ColorDefault
	mauview.Styles.ContrastBackgroundColor = tcell.ColorDarkGreen
	if tcellDB := os.Getenv("TCELLDB"); len(tcellDB) == 0 {
		if info, err := os.Stat("/usr/share/tcell/database"); err == nil && info.IsDir() {
			os.Setenv("TCELLDB", "/usr/share/tcell/database")
		}
	}
}

func (gt *GomuksTUI) Gmx() *gomuks.Gomuks {
	return gt.Gomuks
}

func (gt *GomuksTUI) Rpc() *rpc.GomuksRPC {
	return gt.rpc
}

func (gt *GomuksTUI) App() *mauview.Application {
	return gt.app
}

func (gt *GomuksTUI) PingTicker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-gt.pingTicker.C:
			if gt.rpc != nil {
				_, err := gt.rpc.Ping(ctx, &jsoncmd.PingParams{LastReceivedID: gt.rpc.LastReqID})
				if err != nil && !errors.Is(err, rpc.ErrNotConnectedToWebsocket) {
					gt.Gomuks.Log.Err(err).Msg("failed to ping gomuks over RPC")
				}
			}
		}
	}
}

func (gt *GomuksTUI) InitViews(ctx context.Context) {
	gt.authView = ui.NewAuthenticateView(ctx, gt)
	gt.syncView = ui.NewSyncingView(gt)
	gt.loginView = ui.NewLoginView(ctx, gt)
	gt.verifyView = ui.NewVerifySessionView(ctx, gt)
	gt.mainView = ui.NewMainView(ctx, gt)

	// Set the initial view to the authentication view
	gt.app.SetRoot(gt.authView.Container)
}

func (gt *GomuksTUI) Run() {
	logger := gt.Gomuks.Log
	gt.app = mauview.NewApplication()
	rpcClient, err := rpc.NewGomuksRPC("http://" + gt.Gomuks.Config.Web.ListenAddress)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create gomuks RPC client")
		return
	}
	gt.rpc = rpcClient
	gt.rpc.EventHandler = gt.OnEvent
	gt.pingTicker = time.NewTicker(30 * time.Second)
	defer gt.pingTicker.Stop()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	go gt.PingTicker(ctx)
	gt.InitViews(ctx)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Any("error", r).Msgf("gomuks TUI panicked")
				cancelCtx()
				gt.app.ForceStop()
			}
		}()
		logger.Debug().Msg("waiting for interrupt")
		gt.Gomuks.WaitForInterrupt()
		logger.Warn().Msg("gomuks TUI interrupt received, stopping app")
		cancelCtx()
		gt.app.ForceStop()
		logger.Debug().Msg("app stopped")
	}()
	logger.Trace().Msg("starting app")
	err = gt.app.Start()
	logger.Trace().Err(err).Msg("finished app")
	if err != nil {
		panic(err)
	}
}
