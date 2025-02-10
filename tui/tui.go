// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
	"os"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/gomuks"
)

type View string

// Allowed views in GomuksTUI
type GomuksTUI struct {
	*gomuks.Gomuks
	App       *mauview.Application
	mainView  *MainView
	loginView *LoginView
}

func New(gmx *gomuks.Gomuks) *GomuksTUI {
	return &GomuksTUI{
		Gomuks: gmx,
		App:    mauview.NewApplication(),
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

func (gt *GomuksTUI) Run() {
	gt.App = mauview.NewApplication()
	if !gt.Client.IsLoggedIn() {
		gt.App.SetRoot(gt.NewLoginView())
	} else {
		gt.App.SetRoot(gt.NewMainView())
	}
	err := gt.App.Start()
	if err != nil {
		panic(err)
	}
}
