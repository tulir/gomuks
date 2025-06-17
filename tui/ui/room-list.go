package ui

import (
	"context"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type RoomList struct {
	*mauview.Grid
	selected id.RoomID
	entries  map[id.RoomID]*mauview.Button

	invited map[id.RoomID]*database.InvitedRoom
	joined  map[id.RoomID]*jsoncmd.SyncRoom
	left    map[id.RoomID]struct{}

	app *App
}

func (rl *RoomList) onInviteClick(ctx context.Context, roomID id.RoomID) func() {
	return func() {
		if rl.selected != roomID {
			rl.selected = roomID
		} else {
			rl.app.gmx.Log.Info().Stringer("room", roomID).Msg("Joining room")
			resp, err := rl.app.rpc.JoinRoom(ctx, &jsoncmd.JoinRoomParams{
				RoomIDOrAlias: roomID.String(),
			})
			if err != nil {
				rl.app.gmx.Log.Error().Err(err).Stringer("room", roomID).Msg("Failed to join room")
				return
			}
			rl.app.gmx.Log.Info().Stringer("room", resp.RoomID).Msg("Joined room")
		}
	}
}

func (rl *RoomList) OnClick(ctx context.Context, roomID id.RoomID) func() {
	return func() {
		if rl.selected != roomID {
			rl.selected = roomID
		}
		if _, exists := rl.invited[roomID]; exists {
			rl.onInviteClick(ctx, roomID)
		} else {
			timeline, exists := rl.app.Views.Timeline[roomID]
			if !exists {
				timeline = NewTimelineView(rl.app, roomID)
				rl.app.Views.Timeline[roomID] = timeline
			}
			rl.app.Views.CurrentTimelineView = timeline
		}
	}
}

func (rl *RoomList) refresh(ctx context.Context) {
	for _, entry := range rl.entries {
		rl.RemoveComponent(entry)
	}
	rl.entries = make(map[id.RoomID]*mauview.Button)
	y := 1
	for roomID := range rl.invited {
		label := mauview.NewButton("(invite) " + roomID.String())
		label.SetOnClick(rl.onInviteClick(ctx, roomID))
		rl.AddComponent(label, 1, y, 1, 1)
		rl.entries[roomID] = label
		y++
	}
	for roomID := range rl.joined {
		name := roomID.String()
		if rl.joined[roomID].Meta.Name != nil {
			name = *rl.joined[roomID].Meta.Name
		}
		label := mauview.NewButton("(joined) " + name)
		rl.AddComponent(label, 1, y, 1, 1)
		rl.entries[roomID] = label
		y++
	}
	for roomID := range rl.left {
		label := mauview.NewButton("(left) " + roomID.String())
		rl.AddComponent(label, 1, y, 1, 1)
		rl.entries[roomID] = label
		y++
	}
}

func (rl *RoomList) HandleSync(ctx context.Context, sync *jsoncmd.SyncComplete) {
	for _, inviteRoom := range sync.InvitedRooms {
		rl.invited[inviteRoom.ID] = inviteRoom
		rl.app.gmx.Log.Debug().Stringer("room_id", inviteRoom.ID).Msg("Added invited room")
	}
	for _, roomID := range sync.LeftRooms {
		rl.left[roomID] = struct{}{}
		rl.app.gmx.Log.Debug().Stringer("room_id", roomID).Msg("Added left room")
	}
	// Any rooms not in leftRooms or InvitedRooms are considered joined
	for roomID, room := range sync.Rooms {
		_, invited := rl.invited[roomID]
		_, left := rl.left[roomID]
		if !invited && !left {
			rl.joined[roomID] = room
			rl.app.gmx.Log.Debug().Stringer("room_id", roomID).Msg("Added joined room")
		}
	}
	rl.refresh(ctx)
}

func NewRoomList(app *App) *RoomList {
	rl := &RoomList{
		app:     app,
		Grid:    mauview.NewGrid(),
		entries: make(map[id.RoomID]*mauview.Button),
		invited: make(map[id.RoomID]*database.InvitedRoom),
		joined:  make(map[id.RoomID]*jsoncmd.SyncRoom),
		left:    make(map[id.RoomID]struct{}),
	}
	// 1 column, 25 rows
	rl.Grid.SetColumns([]int{1})
	rl.Grid.SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	return rl
}
