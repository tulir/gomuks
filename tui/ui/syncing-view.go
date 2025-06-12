package ui

import (
	"go.mau.fi/mauview"
)

type SyncingView struct {
	*mauview.Box
	app *MainView
}

func NewSyncingView(app *MainView) *SyncingView {
	return &SyncingView{
		Box: mauview.NewBox(mauview.NewTextField().SetText("Syncing...")),
		app: app,
	}
}
