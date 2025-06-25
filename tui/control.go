package tui

import (
	"context"
	"reflect"

	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (gt *GomuksTUI) SwitchRoot(new mauview.Component) {
	gt.app.SetRoot(new)
}

func (gt *GomuksTUI) OnEvent(ctx context.Context, evt any) {
	gt.Log.Debug().Interface("event", evt).Stringer("type", reflect.TypeOf(evt)).Msg("new event from websocket")
	switch e := evt.(type) {
	case *jsoncmd.ClientState:
		gt.Log.Debug().Interface("state", e).Msg("Received new client state")
		gt.clientState = e
	case *jsoncmd.ImageAuthToken:
		gt.Log.Debug().Msg("Received image authentication token")
		gt.imageAuthToken = string(*e)
	// TODO: handle sync_status
	case *jsoncmd.InitComplete:
		gt.Log.Debug().Msg("Initialization complete")
		gt.initSyncDone = true
	case *jsoncmd.SyncComplete:
		gt.Log.Debug().Msg("New sync received")
		gt.mainView.OnSync(e)
	}
	if !gt.clientState.IsLoggedIn {
		gt.Log.Debug().Msg("Switching to login view")
		gt.SwitchRoot(gt.loginView.Container)
	} else if !gt.clientState.IsVerified {
		gt.Log.Debug().Msg("Switching to verification view")
		gt.SwitchRoot(gt.verifyView.Container)
	} else if !gt.initSyncDone {
		gt.Log.Debug().Msg("Switching to sync waiter view")
		gt.SwitchRoot(gt.syncView.Container)
	} else {
		gt.Log.Debug().Msg("Switching to main view")
		gt.SwitchRoot(gt.mainView)
	}
}
