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

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.mau.fi/util/exhttp"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/version"
	"go.mau.fi/gomuks/web"
)

type PointableHandler struct {
	handler http.Handler
}

var _ http.Handler = (*PointableHandler)(nil)

func (p *PointableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.handler.ServeHTTP(w, r)
}

type CommandHandler struct {
	Gomuks *gomuks.Gomuks
	Ctx    context.Context
	App    *application.App
}

func (c *CommandHandler) HandleCommand(cmd *hicli.JSONCommand) *hicli.JSONCommand {
	return c.Gomuks.Client.SubmitJSONCommand(c.Ctx, cmd)
}

func (c *CommandHandler) Init() {
	c.Gomuks.Log.Info().Msg("Sending initial state to client")
	c.App.EmitEvent("hicli_event", &hicli.JSONCommandCustom[*jsoncmd.ClientState]{
		Command: jsoncmd.EventClientState,
		Data:    c.Gomuks.Client.State(),
	})
	c.App.EmitEvent("hicli_event", &hicli.JSONCommandCustom[*jsoncmd.SyncStatus]{
		Command: jsoncmd.EventSyncStatus,
		Data:    c.Gomuks.Client.SyncStatus.Load(),
	})
	if c.Gomuks.Client.IsLoggedIn() {
		go func() {
			log := c.Gomuks.Log
			ctx := log.WithContext(context.TODO())
			var roomCount int
			for payload := range c.Gomuks.Client.GetInitialSync(ctx, 100) {
				roomCount += len(payload.Rooms)
				marshaledPayload, err := json.Marshal(&payload)
				if err != nil {
					log.Err(err).Msg("Failed to marshal initial rooms to send to client")
					return
				}
				c.App.EmitEvent("hicli_event", &hicli.JSONCommand{
					Command:   jsoncmd.EventSyncComplete,
					RequestID: 0,
					Data:      marshaledPayload,
				})
			}
			if ctx.Err() != nil {
				return
			}
			c.App.EmitEvent("hicli_event", &hicli.JSONCommand{
				Command:   jsoncmd.EventInitComplete,
				RequestID: 0,
			})
			log.Info().Int("room_count", roomCount).Msg("Sent initial rooms to client")
		}()
	}
}

func main() {
	gmx := gomuks.NewGomuks()
	gmx.Version = version.Version
	gmx.Commit = version.Commit
	gmx.LinkifiedVersion = version.LinkifiedVersion
	gmx.BuildTime = version.ParsedBuildTime
	gmx.DisableAuth = true
	exhttp.AutoAllowCORS = false
	hicli.InitialDeviceDisplayName = "gomuks desktop"

	gmx.InitDirectories()
	err := gmx.LoadConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(9)
	}
	gmx.SetupLog()
	gmx.Log.Info().
		Str("version", gmx.Version).
		Str("go_version", runtime.Version()).
		Time("built_at", gmx.BuildTime).
		Msg("Initializing gomuks desktop")
	gmx.StartClient()
	gmx.Log.Info().Msg("Initialization complete, starting desktop app")

	cmdCtx, cancelCmdCtx := context.WithCancel(context.Background())
	ch := &CommandHandler{Gomuks: gmx, Ctx: cmdCtx}
	app := application.New(application.Options{
		Name:        "gomuks-desktop",
		Description: "A Matrix client written in Go and React",
		Services: []application.Service{
			application.NewService(
				&PointableHandler{gmx.CreateAPIRouter()},
				application.ServiceOptions{Route: "/_gomuks"},
			),
			application.NewService(ch),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(web.Frontend),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			cancelCmdCtx()
			gmx.Log.Info().Msg("Shutting down...")
			gmx.DirectStop()
			gmx.Log.Info().Msg("Shutdown complete")
		},
	})
	ch.App = app

	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "gomuks desktop",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	gmx.EventBuffer.Subscribe(0, nil, func(command *hicli.JSONCommand) {
		app.EmitEvent("hicli_event", command)
	})

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
