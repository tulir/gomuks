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

package gomuks

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/exzerolog"
	"go.mau.fi/util/ptr"
	"go.mau.fi/zeroconfig"
	"golang.org/x/net/http2"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type Gomuks struct {
	Log    *zerolog.Logger
	Server *http.Server
	Client *hicli.HiClient

	Version          string
	Commit           string
	LinkifiedVersion string
	BuildTime        time.Time

	ConfigDir string
	DataDir   string
	CacheDir  string
	TempDir   string
	LogDir    string

	FrontendFS    embed.FS
	indexWithETag []byte
	frontendETag  string

	Config      Config
	DisableAuth bool

	stopOnce sync.Once
	stopChan chan struct{}

	EventBuffer *EventBuffer

	// Maps from temporary MXC URIs from by the media repository for URL
	// previews to permanent MXC URIs suitable for sending in an inline preview
	temporaryMXCToPermanent         map[id.ContentURIString]id.ContentURIString
	temporaryMXCToEncryptedFileInfo map[id.ContentURIString]*event.EncryptedFileInfo
	TUI                             tui
}

type tui interface {
	Run()
}

func NewGomuks() *Gomuks {
	return &Gomuks{
		stopChan: make(chan struct{}),

		temporaryMXCToPermanent:         map[id.ContentURIString]id.ContentURIString{},
		temporaryMXCToEncryptedFileInfo: map[id.ContentURIString]*event.EncryptedFileInfo{},
	}
}

func (gmx *Gomuks) InitDirectories() {
	// We need 4 directories: config, data, cache, logs
	//
	// 1. If GOMUKS_ROOT is set, all directories are created under that.
	// 2. If GOMUKS_*_HOME is set, that value is used as the directory.
	// 3. Use system-specific defaults as below
	//
	// *nix:
	// - Config: $XDG_CONFIG_HOME/gomuks or $HOME/.config/gomuks
	// - Data: $XDG_DATA_HOME/gomuks or $HOME/.local/share/gomuks
	// - Cache: $XDG_CACHE_HOME/gomuks or $HOME/.cache/gomuks
	// - Logs: $XDG_STATE_HOME/gomuks or $HOME/.local/state/gomuks
	//
	// Windows:
	// - Config and Data: %AppData%\gomuks
	// - Cache: %LocalAppData%\gomuks
	// - Logs: %LocalAppData%\gomuks\logs
	//
	// macOS:
	// - Config and Data: $HOME/Library/Application Support/gomuks
	// - Cache: $HOME/Library/Caches/gomuks
	// - Logs: $HOME/Library/Logs/gomuks
	if gomuksRoot := os.Getenv("GOMUKS_ROOT"); gomuksRoot != "" {
		exerrors.PanicIfNotNil(os.MkdirAll(gomuksRoot, 0700))
		gmx.CacheDir = filepath.Join(gomuksRoot, "cache")
		gmx.ConfigDir = filepath.Join(gomuksRoot, "config")
		gmx.DataDir = filepath.Join(gomuksRoot, "data")
		gmx.LogDir = filepath.Join(gomuksRoot, "logs")
	} else {
		homeDir := exerrors.Must(os.UserHomeDir())
		if cacheDir := os.Getenv("GOMUKS_CACHE_HOME"); cacheDir != "" {
			gmx.CacheDir = cacheDir
		} else {
			gmx.CacheDir = filepath.Join(exerrors.Must(os.UserCacheDir()), "gomuks")
		}
		if configDir := os.Getenv("GOMUKS_CONFIG_HOME"); configDir != "" {
			gmx.ConfigDir = configDir
		} else {
			gmx.ConfigDir = filepath.Join(exerrors.Must(os.UserConfigDir()), "gomuks")
		}
		if dataDir := os.Getenv("GOMUKS_DATA_HOME"); dataDir != "" {
			gmx.DataDir = dataDir
		} else if dataDir = os.Getenv("XDG_DATA_HOME"); dataDir != "" {
			gmx.DataDir = filepath.Join(dataDir, "gomuks")
		} else if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			gmx.DataDir = gmx.ConfigDir
		} else {
			gmx.DataDir = filepath.Join(homeDir, ".local", "share", "gomuks")
		}
		if logDir := os.Getenv("GOMUKS_LOGS_HOME"); logDir != "" {
			gmx.LogDir = logDir
		} else if logDir = os.Getenv("XDG_STATE_HOME"); logDir != "" {
			gmx.LogDir = filepath.Join(logDir, "gomuks")
		} else if runtime.GOOS == "darwin" {
			gmx.LogDir = filepath.Join(homeDir, "Library", "Logs", "gomuks")
		} else if runtime.GOOS == "windows" {
			gmx.LogDir = filepath.Join(gmx.CacheDir, "logs")
		} else {
			gmx.LogDir = filepath.Join(homeDir, ".local", "state", "gomuks")
		}
	}
	if gmx.TempDir = os.Getenv("GOMUKS_TMPDIR"); gmx.TempDir == "" {
		gmx.TempDir = filepath.Join(gmx.CacheDir, "tmp")
	}
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.ConfigDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.CacheDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.TempDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.DataDir, 0700))
	exerrors.PanicIfNotNil(os.MkdirAll(gmx.LogDir, 0700))
	defaultFileWriter.FileConfig.Filename = filepath.Join(gmx.LogDir, "gomuks.log")
}

func (gmx *Gomuks) SetupLog() {
	if gmx.TUI != nil {
		// Remove stdout and stderr writers if TUI is enabled
		gmx.Config.Logging.Writers = slices.DeleteFunc(gmx.Config.Logging.Writers, func(config zeroconfig.WriterConfig) bool {
			return config.Type == zeroconfig.WriterTypeStdout || config.Type == zeroconfig.WriterTypeStderr
		})
	}
	gmx.Log = exerrors.Must(gmx.Config.Logging.Compile())
	exzerolog.SetupDefaults(gmx.Log)
}

func (gmx *Gomuks) StartClient() {
	hicli.HTMLSanitizerImgSrcTemplate = "_gomuks/media/%s/%s?encrypted=false"
	rawDB, err := dbutil.NewFromConfig("gomuks", dbutil.Config{
		PoolConfig: dbutil.PoolConfig{
			Type:         "sqlite3-fk-wal",
			URI:          fmt.Sprintf("file:%s/gomuks.db?_txlock=immediate", gmx.DataDir),
			MaxOpenConns: 5,
			MaxIdleConns: 1,
		},
	}, dbutil.ZeroLogger(gmx.Log.With().Str("component", "hicli").Str("db_section", "main").Logger()))
	if err != nil {
		gmx.Log.WithLevel(zerolog.FatalLevel).Err(err).Msg("Failed to open database")
		os.Exit(10)
	}
	ctx := gmx.Log.WithContext(context.Background())
	gmx.Client = hicli.New(
		rawDB,
		nil,
		gmx.Log.With().Str("component", "hicli").Logger(),
		[]byte("meow"),
		gmx.HandleEvent,
	)
	gmx.Client.LogoutFunc = gmx.Logout
	httpClient := gmx.Client.Client.Client
	httpClient.Transport.(*http.Transport).ForceAttemptHTTP2 = false
	if !gmx.Config.Matrix.DisableHTTP2 {
		h2, err := http2.ConfigureTransports(httpClient.Transport.(*http.Transport))
		if err != nil {
			gmx.Log.WithLevel(zerolog.FatalLevel).Err(err).Msg("Failed to configure HTTP/2")
			os.Exit(13)
		}
		h2.ReadIdleTimeout = 30 * time.Second
	}
	userID, err := gmx.Client.DB.Account.GetFirstUserID(ctx)
	if err != nil {
		gmx.Log.WithLevel(zerolog.FatalLevel).Err(err).Msg("Failed to get first user ID")
		os.Exit(11)
	}
	err = gmx.Client.Start(ctx, userID, nil)
	if err != nil {
		gmx.Log.WithLevel(zerolog.FatalLevel).Err(err).Msg("Failed to start client")
		os.Exit(12)
	}
	gmx.Log.Info().Stringer("user_id", userID).Msg("Client started")
}

func (gmx *Gomuks) HandleEvent(evt any) {
	gmx.EventBuffer.Push(evt)
	syncComplete, ok := evt.(*jsoncmd.SyncComplete)
	if ok && ptr.Val(syncComplete.Since) != "" {
		go gmx.SendPushNotifications(syncComplete)
	}
}

func (gmx *Gomuks) Stop() {
	gmx.stopOnce.Do(func() {
		close(gmx.stopChan)
	})
}

func (gmx *Gomuks) WaitForInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	select {
	case <-c:
	case <-gmx.stopChan:
	}
}

func (gmx *Gomuks) DirectStop() {
	for _, closer := range gmx.EventBuffer.GetClosers() {
		closer(websocket.StatusServiceRestart, "Server shutting down")
	}
	gmx.Client.Stop()
	if gmx.Server != nil {
		err := gmx.Server.Close()
		if err != nil {
			gmx.Log.Error().Err(err).Msg("Failed to close server")
		}
	}
}

func (gmx *Gomuks) Run() {
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
		Msg("Initializing gomuks")
	gmx.StartServer()
	gmx.StartClient()
	gmx.Log.Info().Msg("Initialization complete")
	if gmx.TUI != nil {
		gmx.TUI.Run()
	} else {
		gmx.WaitForInterrupt()
	}
	gmx.Log.Info().Msg("Shutting down...")
	gmx.DirectStop()
	gmx.Log.Info().Msg("Shutdown complete")
	os.Exit(0)
}
