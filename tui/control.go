package tui

import (
	"context"

	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (gt *GomuksTUI) SwitchRoot(new mauview.Component) {
	gt.app.SetRoot(new)
}

func (gt *GomuksTUI) OnEvent(ctx context.Context, evt any) {
	gt.Log.Debug().Interface("event", evt).Msg("new event from websocket")
	switch e := evt.(type) {
	case *jsoncmd.ClientState:
		gt.clientState = e
		if !e.IsLoggedIn {
			gt.SwitchRoot(gt.loginView.Container)
		} else if !e.IsVerified {
			gt.SwitchRoot(gt.syncView.Container) // TODO
			gt.syncView.Run(ctx)                 // TODO also: this doesnt allow us to reset the refresh ticker
		} else {
			// is an update even needed here?
		}
	case *jsoncmd.ImageAuthToken:
		gt.imageAuthToken = string(*e)
	// TODO: handle sync_status
	case *jsoncmd.InitComplete:
		gt.initSyncDone = true
	case *jsoncmd.SyncComplete:
		for roomID, room := range e.Rooms {
			gt.rooms[roomID] = room
		}
	}
	if !gt.clientState.IsLoggedIn {
		gt.SwitchRoot(gt.loginView.Container)
	}
}
