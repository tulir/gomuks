package ui

import (
	"context"
	"time"

	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/tui/abstract"
)

// SyncingView is a view that displays a loading indicator while syncing
type SyncingView struct {
	*mauview.Flex
	app       abstract.App
	Container *mauview.Centerer
	bar       *mauview.ProgressBar
}

func (sv *SyncingView) Run(ctx context.Context) {
	ticker := time.Tick(100 * time.Millisecond)
	sv.app.App().SetRedrawTicker(100 * time.Millisecond)
	go func() {
		select {
		case <-ctx.Done():
			sv.app.App().SetRedrawTicker(1 * time.Minute)
			return
		case <-ticker:
			sv.bar.Increment(1)
		}
	}()
}

func NewSyncingView(app abstract.App) *SyncingView {
	s := &SyncingView{
		app:  app,
		Flex: mauview.NewFlex(),
		bar:  mauview.NewProgressBar(),
	}
	s.bar.SetIndeterminate(true)
	s.Container = mauview.Center(mauview.NewBox(s.bar).SetTitle("Syncing").SetBorder(true), 40, 3).SetAlwaysFocusChild(true)
	return s
}
