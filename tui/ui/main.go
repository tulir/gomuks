package ui

import (
	"context"
	"encoding/json"
	"sync"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"

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
	MemberLists map[id.RoomID]*components.MemberList
	syncLock    sync.Mutex

	memberListElement *components.MemberList
}

func (m *MainView) OnSync(resp *jsoncmd.SyncComplete) {
	m.syncLock.Lock()
	defer m.syncLock.Unlock()
	for _, leftRoomID := range resp.LeftRooms {
		// Remove data for rooms we left
		delete(m.MemberLists, leftRoomID)
	}
	// Add invited rooms to the top of the room list
	for _, room := range resp.InvitedRooms {
		//m.RoomList.AddRoom(room.ID)
		// bad!
		m.RoomList.AddRoom(room.ID, &jsoncmd.SyncRoom{})
	}

	// Process joined rooms
	for roomID, room := range resp.Rooms {
		existingRoom := m.RoomList.Elements[roomID]
		if existingRoom != nil {
			if room.Meta != nil && room.Meta.Name != nil && *room.Meta.Name != "" {
				// Update existing room name
				existingRoom.SetText(*room.Meta.Name)
			}
		} else {
			// Add new room
			m.RoomList.AddRoom(roomID, room)
		}
	}
}

func (m *MainView) OnRoomSelected(old, new id.RoomID) {
	if old == new {
		m.app.Gmx().Log.Debug().Msgf("ignoring room switch from %s to itself", old)
		return
	}
	memberlist, ok := m.MemberLists[new]
	if !ok {
		memberlist = components.NewMemberList(m.ctx, m.app, []id.UserID{}, nil)
		m.MemberLists[new] = memberlist
	}
	m.app.Gmx().Log.Debug().Msgf("switching to room view for %s from %s", old, new)
	evts, err := m.app.Rpc().GetRoomState(m.ctx, &jsoncmd.GetRoomStateParams{RoomID: new, IncludeMembers: true})
	if err == nil {
		for _, evt := range evts {
			if evt.Type == "m.room.member" {
				var content event.MemberEventContent
				if err = json.Unmarshal(evt.Content, &content); err != nil {
					continue
				}
				if content.Membership == "join" {
					memberlist.Members = append(memberlist.Members, id.UserID(*evt.StateKey))
				}
			}
		}
	}
	m.memberListElement = memberlist
	m.memberListElement.Render()
}

func NewMainView(ctx context.Context, app abstract.App) *MainView {
	m := &MainView{
		Flex:              mauview.NewFlex(),
		app:               app,
		ctx:               ctx,
		MemberLists:       make(map[id.RoomID]*components.MemberList),
		memberListElement: components.NewMemberList(ctx, app, []id.UserID{}, nil),
	}
	m.MemberLists[""] = m.memberListElement

	m.RoomList = components.NewRoomList(ctx, app, m.OnRoomSelected)
	m.AddProportionalComponent(m.RoomList, 1)
	m.AddProportionalComponent(mauview.NewFlex(), 4)
	m.AddProportionalComponent(m.memberListElement, 1)
	// rooms: x1
	// timeline: x4
	// members: x1
	// ?#
	return m
}
