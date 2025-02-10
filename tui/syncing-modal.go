// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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
	"time"

	"go.mau.fi/mauview"
)

type SyncingModal struct {
	parent   *MainView
	text     *mauview.TextView
	progress *mauview.ProgressBar
}

func NewSyncingModal(parent *MainView) (mauview.Component, *SyncingModal) {
	sm := &SyncingModal{
		parent:   parent,
		progress: mauview.NewProgressBar(),
		text:     mauview.NewTextView(),
	}
	return mauview.Center(
		mauview.NewBox(
			mauview.NewFlex().
				SetDirection(mauview.FlexRow).
				AddFixedComponent(sm.progress, 1).
				AddFixedComponent(mauview.Center(sm.text, 40, 1), 1)).
			SetTitle("Synchronizing"),
		42, 4).
		SetAlwaysFocusChild(true), sm
}

func (sm *SyncingModal) SetMessage(text string) {
	sm.text.SetText(text)
}

func (sm *SyncingModal) SetIndeterminate() {
	sm.progress.SetIndeterminate(true)
	sm.parent.parent.app.SetRedrawTicker(100 * time.Millisecond)
	sm.parent.parent.app.Redraw()
}

func (sm *SyncingModal) SetSteps(max int) {
	sm.progress.SetMax(max)
	sm.progress.SetIndeterminate(false)
	sm.parent.parent.app.SetRedrawTicker(1 * time.Minute)
	sm.parent.parent.Render()
}

func (sm *SyncingModal) Step() {
	sm.progress.Increment(1)
}

func (sm *SyncingModal) Close() {
	sm.parent.HideModal()
}
