package ui

import (
	"fmt"
	"sync"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type TimelineView struct {
	*mauview.Flex
	RoomID id.RoomID

	events    []*event.Event
	eventRows map[id.EventID]*mauview.TextField
	evtLock   sync.Mutex

	app *App
}

func NewTimelineView(app *App, roomID id.RoomID) *TimelineView {
	tl := &TimelineView{
		Flex:      mauview.NewFlex(),
		RoomID:    roomID,
		app:       app,
		events:    make([]*event.Event, 0),
		eventRows: make(map[id.EventID]*mauview.TextField),
	}

	tl.SetDirection(mauview.FlexRow)
	tl.AddProportionalComponent(mauview.NewTextField().SetText("Room ID: "+roomID.String()), 1)
	tl.AddProportionalComponent(mauview.NewTextField().SetText("Room ID a: "+roomID.String()), 1)
	tl.AddProportionalComponent(mauview.NewTextField().SetText("Room ID b: "+roomID.String()), 1)
	tl.AddProportionalComponent(mauview.NewTextField().SetText("Room ID c: "+roomID.String()), 1)
	tl.AddProportionalComponent(mauview.NewTextField().SetText("Room ID d: "+roomID.String()), 1)
	tl.AddProportionalComponent(mauview.NewTextField().SetText("Room ID e: "+roomID.String()), 1)
	return tl
}

func (tl *TimelineView) AddEvent(evt *event.Event) {
	tl.evtLock.Lock()
	defer tl.evtLock.Unlock()
	_, exists := tl.eventRows[evt.ID]
	if exists {
		tl.app.gmx.Log.Warn().Str("event_id", evt.ID.String()).Msg("Event already exists in timeline")
		return
	}
	tl.events = append(tl.events, evt)

	value := mauview.NewTextField()
	value.SetText(fmt.Sprintf("%s sent event type %s (%s)", evt.Sender, evt.Type, evt.ID))
	tl.eventRows[evt.ID] = value
	tl.AddProportionalComponent(value, 1)
}
