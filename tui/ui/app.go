package ui

import (
	"context"

	"maunium.net/go/mautrix/id"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc"
)

type Views struct {
	Authenticate *AuthenticateView
	LoginForm    *LoginFormView
	RoomList     *RoomList
	Syncing      *SyncingView
	Timeline     map[id.RoomID]*TimelineView
	Main         *MainView
	app          *App

	CurrentTimelineView *TimelineView
}

func (v *Views) HandleSync(ctx context.Context, sync *jsoncmd.SyncComplete) {
	for roomID, room := range sync.Rooms {
		timeline, exists := v.Timeline[roomID]
		if !exists {
			timeline = NewTimelineView(v.app, roomID)
			v.Timeline[roomID] = timeline
		}
		for _, evt := range room.Events {
			raw := evt.AsRawMautrix()
			_ = raw.Content.ParseRaw(raw.Type)
			timeline.AddEvent(evt.AsRawMautrix())
		}
	}
	v.RoomList.HandleSync(ctx, sync)
}

type App struct {
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

func (mv *App) OnEvent(ctx context.Context, evt any) {
	logger := mv.gmx.Log
	switch e := evt.(type) {
	case *jsoncmd.SyncComplete:
		mv.syncCounter++
		data := evt.(*jsoncmd.SyncComplete)
		mv.lastSync = data.Since
		mv.Views.HandleSync(ctx, evt.(*jsoncmd.SyncComplete))
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
		mv.app.SetRoot(mv.Views.Main)
	}
}

func (mv *App) QuitOnKey() func(event mauview.KeyEvent) mauview.KeyEvent {
	return func(event mauview.KeyEvent) mauview.KeyEvent {
		mv.gmx.Log.Debug().Any("key", event.Rune()).Msg("Key pressed")
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyCtrlC {
			mv.app.ForceStop()
		}
		return event
	}
}

func NewApp(ctx context.Context, gmx *gomuks.Gomuks, app *mauview.Application, rpc *rpc.GomuksRPC) *App {
	main := &App{
		gmx: gmx,
		rpc: rpc,
		app: app,
	}
	loginView := NewLoginForm(ctx, main)
	roomView := NewRoomList(main)
	syncingView := NewSyncingView(main)
	authView := NewAuthenticateView(ctx, main)
	views := &Views{
		Authenticate:        authView,
		LoginForm:           loginView,
		RoomList:            roomView,
		Syncing:             syncingView,
		Timeline:            make(map[id.RoomID]*TimelineView),
		CurrentTimelineView: NewTimelineView(main, "!offtopic-9:timedout.uk"),
	}
	main.Views = views
	views.Main = NewMainView(main)
	rpc.EventHandler = main.OnEvent
	return main
}
