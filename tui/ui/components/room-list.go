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
	app           abstract.App
	ctx           context.Context
	Elements      map[id.RoomID]*mauview.Button
	focused       id.RoomID
	onFocusChange func(old, new id.RoomID)
}

func NewRoomList(ctx context.Context, app abstract.App, onFocusChange func(old, new id.RoomID)) *RoomList {
	rl := &RoomList{
		app:           app,
		ctx:           ctx,
		Flex:          mauview.NewFlex().SetDirection(mauview.FlexRow),
		Elements:      make(map[id.RoomID]*mauview.Button),
		onFocusChange: onFocusChange,
	}
	rl.AddFixedComponent(mauview.NewTextField().SetText("Room List"), 1)
	return rl
}

func (rl *RoomList) focusRoom(roomID id.RoomID) {
	old := rl.focused
	if roomID == rl.focused {
		return
	}
	if button, exists := rl.Elements[roomID]; exists {
		button.Focus()
	}
	rl.focused = roomID
	for otherRoomID := range rl.Elements {
		if otherRoomID != roomID {
			if button, exists := rl.Elements[otherRoomID]; exists {
				button.Blur()
			}
		}
	}
	rl.app.Gmx().Log.Debug().Msgf("focused room: %s", string(roomID))
	rl.onFocusChange(old, roomID)
}

func (rl *RoomList) AddRoom(roomID id.RoomID, room *jsoncmd.SyncRoom) *RoomList {
	// todo: re-rendering should be its own func
	if _, exists := rl.Elements[roomID]; exists {
		return rl
	}
	button := mauview.NewButton(lazyRoomName(room, roomID))
	button.SetOnClick(func() {
		rl.focusRoom(roomID)
	})
	rl.Elements[roomID] = button
	rl.Flex.AddFixedComponent(button, 1)
	rl.app.App().Redraw()
	return rl
}

func (rl *RoomList) OnKeyEvent(event mauview.KeyEvent) bool {
	switch event.Key() {
	case tcell.KeyUp:
		// go to room above
		var prevRoomID id.RoomID
		for roomID := range rl.Elements {
			if roomID == rl.focused {
				rl.focusRoom(prevRoomID)
				return true
			}
			prevRoomID = roomID
		}
	case tcell.KeyDown:
		// go to room below
		roomIDs := make([]id.RoomID, 0, len(rl.Elements))
		for roomID := range rl.Elements {
			roomIDs = append(roomIDs, roomID)
		}
		for i, roomID := range roomIDs {
			if roomID == rl.focused {
				if i+1 < len(roomIDs) {
					rl.focusRoom(roomIDs[i+1])
				} else {
					rl.focusRoom(roomIDs[0]) // wrap around
				}
				return true
			}
		}
	case tcell.KeyEsc:
		rl.app.Gmx().Stop()
		return true
	default:
		return false
	}
	return true
}
