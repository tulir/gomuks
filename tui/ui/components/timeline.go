package components

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/abstract"
)

type TimelineComponent struct {
	*mauview.Flex

	app abstract.App
	ctx context.Context

	//timeline []database.Event
	elements map[id.EventID]mauview.Component
}

func NewTimeline(ctx context.Context, app abstract.App) *TimelineComponent {
	timeline := &TimelineComponent{
		Flex:     mauview.NewFlex(),
		app:      app,
		ctx:      ctx,
		elements: make(map[id.EventID]mauview.Component),
	}
	timeline.AddFixedComponent(mauview.NewTextField().SetText("Timeline"), 1)
	return timeline
}

func (t *TimelineComponent) AddEvent(evt *database.Event) {
	if _, exists := t.elements[evt.ID]; exists {
		// Event already exists in the timeline
		return
	}
	var content event.MessageEventContent
	if evt.Type != "m.room.message" {
		content = event.MessageEventContent{Body: "sent event: " + evt.Type}
	} else {
		if err := json.Unmarshal(evt.Content, &content); err != nil {
			content = event.MessageEventContent{Body: fmt.Sprintf("failed to parse content: %v", err)}
		}
	}
	timelineElement := NewMessage(t.ctx, t.app, evt)
	t.AddFixedComponent(timelineElement, 1)
	t.app.App().Redraw()
}
