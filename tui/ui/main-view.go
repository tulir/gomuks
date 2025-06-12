package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc"
)

type Views struct {
	LoginForm    *LoginFormView
	Syncing      *SyncingView
	RoomList     *RoomList
	Authenticate *AuthenticateView
}

type MainView struct {
	gmx *gomuks.Gomuks
	app *mauview.Application
	rpc *rpc.GomuksRPC

	Views *Views

	syncCounter      int
	lastSync         *string
	rpcAuthenticated bool
	initDone         bool
	imageAuthToken   string
}

func (mv *MainView) OnEvent(ctx context.Context, evt any) {
	logger := mv.gmx.Log
	switch e := evt.(type) {
	case *jsoncmd.SyncComplete:
		mv.syncCounter++
		data := evt.(*jsoncmd.SyncComplete)
		mv.lastSync = data.Since
		mv.Views.RoomList.HandleSync(ctx, evt.(*jsoncmd.SyncComplete))
	case *jsoncmd.InitComplete:
		mv.initDone = true
	case *jsoncmd.ImageAuthToken:
		p := evt.(*jsoncmd.ImageAuthToken)
		mv.imageAuthToken = string(*p)
	default:
		logger.Warn().Type("type", e).Any("evt", evt).Msg("unhandled event")
	}

	if !mv.rpcAuthenticated {
		mv.app.SetRoot(mv.Views.Authenticate.Container)
	} else if !mv.initDone && mv.syncCounter == 0 {
		mv.app.SetRoot(mv.Views.LoginForm.Container)
	} else if !mv.initDone && mv.syncCounter > 0 {
		mv.app.SetRoot(mv.Views.Syncing.Box)
	} else {
		mv.app.SetRoot(mv.Views.RoomList.Grid)
	}
}

func (mv *MainView) QuitOnKey() func(event mauview.KeyEvent) mauview.KeyEvent {
	return func(event mauview.KeyEvent) mauview.KeyEvent {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyCtrlC {
			mv.app.ForceStop()
		}
		return event
	}
}

func NewMainView(ctx context.Context, gmx *gomuks.Gomuks, app *mauview.Application, rpc *rpc.GomuksRPC) *MainView {
	main := &MainView{
		gmx: gmx,
		rpc: rpc,
		app: app,
	}
	loginView := NewLoginForm(ctx, main)
	roomView := NewRoomList(main)
	syncingView := NewSyncingView(main)
	authView := NewAuthenticateView(ctx, main)
	views := &Views{
		LoginForm:    loginView,
		Syncing:      syncingView,
		RoomList:     roomView,
		Authenticate: authView,
	}
	main.Views = views
	rpc.EventHandler = main.OnEvent
	return main
}
