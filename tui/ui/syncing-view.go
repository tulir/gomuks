package ui

import (
	"go.mau.fi/mauview"
)

type SyncingView struct {
	*mauview.Box
	app *App
}

func NewSyncingView(app *App) *SyncingView {
	// TODO: mauview.ProgressBar
	return &SyncingView{
		Box: mauview.NewBox(mauview.NewTextField().SetText("Syncing...")),
		app: app,
	}
}
