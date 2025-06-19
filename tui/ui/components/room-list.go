package components

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/tui/abstract"
)

// todo: temp
func lazyRoomName(obj *jsoncmd.SyncRoom, roomID id.RoomID) string {
	if obj.Meta != nil {
		if obj.Meta.Name != nil {
			return *obj.Meta.Name
		}
		if obj.Meta.CanonicalAlias != nil {
			return obj.Meta.CanonicalAlias.String()
		}
	}
	return roomID.String()
}

type RoomList struct {
	*mauview.Flex
	app      abstract.App
	ctx      context.Context
	elements map[id.RoomID]*mauview.Button
	focused  id.RoomID
}

func NewRoomList(ctx context.Context, app abstract.App) *RoomList {
	return &RoomList{
		app:      app,
		ctx:      ctx,
		Flex:     mauview.NewFlex().SetDirection(mauview.FlexRow),
		elements: make(map[id.RoomID]*mauview.Button),
	}
}

func (rl *RoomList) focusRoom(roomID id.RoomID) {
	if roomID == rl.focused {
		return
	}
	if button, exists := rl.elements[roomID]; exists {
		button.Focus()
	}
	rl.focused = roomID
	rl.app.Gmx().Log.Debug().Msgf("focused room: %s", string(roomID))
}

func (rl *RoomList) AddRoom(roomID id.RoomID, room *jsoncmd.SyncRoom) *RoomList {
	if _, exists := rl.elements[roomID]; exists {
		return rl
	}
	button := mauview.NewButton(lazyRoomName(room, roomID))
	button.SetOnClick(func() {
		rl.focusRoom(roomID)
	})
	rl.elements[roomID] = button
	rl.Flex.AddFixedComponent(button, 1)
	rl.app.App().Redraw()
	return rl
}

func (rl *RoomList) OnKeyEvent(event mauview.KeyEvent) bool {
	switch event.Key() {
	case tcell.KeyUp:
		// go to room above
		break
	case tcell.KeyDown:
		// go to room below
		break
	case tcell.KeyEsc:
		rl.app.Gmx().Stop()
		return true
	default:
		return false
	}
	return true
}
