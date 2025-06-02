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
	"os"

	"github.com/rs/zerolog"

	"go.mau.fi/gomuks/tui/ui"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/gomuks"
)

type GomuksTUI struct {
	*gomuks.Gomuks
	App *mauview.Application
}

func New(gmx *gomuks.Gomuks) *GomuksTUI {
	return &GomuksTUI{Gomuks: gmx}
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

func (gt *GomuksTUI) Run() {
	ctx, cancel := context.WithCancel(InitLogger(context.Background()))
	defer cancel()
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("gomuks TUI starting up")
	gmxlog := logger.With().Str("component", "gomuks").Logger()
	gt.Gomuks.Log = &gmxlog
	gt.App = mauview.NewApplication()
	form := ui.NewLoginForm(ctx, gt.Gomuks, gt.App)
	view := mauview.NewBox(form.Container).SetBorder(false)
	view.SetKeyCaptureFunc(func(event mauview.KeyEvent) mauview.KeyEvent {
		if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
			logger.Debug().Msg("gomuks TUI exiting, escape key pressed")
			gt.App.ForceStop()
		}
		return event
	})
	form.Container.SetAlwaysFocusChild(true)
	gt.App.SetRoot(view)
	go func() {
		logger.Debug().Msg("waiting for interrupt")
		gt.Gomuks.WaitForInterrupt()
		logger.Debug().Msg("gomuks TUI interrupt received, stopping app")
		gt.App.ForceStop()
	}()
	logger.Trace().Msg("starting app")
	err := gt.App.Start()
	logger.Trace().Err(err).Msg("finished app")
	if err != nil {
		panic(err)
	}
}
