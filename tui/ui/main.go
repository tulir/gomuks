package ui

import (
	"context"

	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/tui/abstract"
	"go.mau.fi/gomuks/tui/ui/components"
)

type MainView struct {
	*mauview.Flex
	app abstract.App
	ctx context.Context

	RoomList *components.RoomList
	// TODO: timeline
	// TODO: members
}

func NewMainView(ctx context.Context, app abstract.App) *MainView {
	m := &MainView{
		Flex: mauview.NewFlex(),
		app:  app,
		ctx:  ctx,
	}

	m.RoomList = components.NewRoomList(ctx, app)
	m.AddProportionalComponent(m.RoomList, 1)
	m.AddProportionalComponent(mauview.NewFlex(), 4)
	m.AddProportionalComponent(mauview.NewFlex(), 1)
	// rooms: x1
	// timeline: x4
	// members: x1
	// ?#
	return m
}
