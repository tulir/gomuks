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
		gt.clientState = e
	case *jsoncmd.ImageAuthToken:
		gt.imageAuthToken = string(*e)
	// TODO: handle sync_status
	case *jsoncmd.InitComplete:
		gt.initSyncDone = true
	case *jsoncmd.SyncComplete:
		gt.mainView.OnSync(e)
	}
	if !gt.clientState.IsLoggedIn {
		gt.SwitchRoot(gt.loginView.Container)
	} else if !gt.clientState.IsVerified {
		gt.SwitchRoot(gt.verifyView.Container)
	} else if !gt.initSyncDone {
		gt.SwitchRoot(gt.syncView.Container)
	} else {
		gt.SwitchRoot(gt.mainView)
	}
}
