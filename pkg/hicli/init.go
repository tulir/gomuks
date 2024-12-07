package hicli

import (
	"context"
	"iter"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

func (h *HiClient) getInitialSyncRoom(ctx context.Context, room *database.Room) *SyncRoom {
	syncRoom := &SyncRoom{
		Meta:          room,
		Events:        make([]*database.Event, 0, 2),
		Timeline:      make([]database.TimelineRowTuple, 0),
		State:         map[event.Type]map[string]database.EventRowID{},
		Notifications: make([]SyncNotification, 0),
	}
	ad, err := h.DB.AccountData.GetAllRoom(ctx, h.Account.UserID, room.ID)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get room account data")
		if ctx.Err() != nil {
			return nil
		}
		syncRoom.AccountData = make(map[event.Type]*database.AccountData)
	} else {
		syncRoom.AccountData = make(map[event.Type]*database.AccountData, len(ad))
		for _, data := range ad {
			syncRoom.AccountData[event.Type{Type: data.Type, Class: event.AccountDataEventType}] = data
		}
	}
	if room.PreviewEventRowID != 0 {
		previewEvent, err := h.DB.Event.GetByRowID(ctx, room.PreviewEventRowID)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get preview event for room")
			if ctx.Err() != nil {
				return nil
			}
		} else if previewEvent != nil {
			h.ReprocessExistingEvent(ctx, previewEvent)
			previewMember, err := h.DB.CurrentState.Get(ctx, room.ID, event.StateMember, previewEvent.Sender.String())
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get preview member event for room")
			} else if previewMember != nil {
				syncRoom.Events = append(syncRoom.Events, previewMember)
				syncRoom.State[event.StateMember] = map[string]database.EventRowID{
					*previewMember.StateKey: previewMember.RowID,
				}
			}
			if previewEvent.LastEditRowID != nil {
				lastEdit, err := h.DB.Event.GetByRowID(ctx, *previewEvent.LastEditRowID)
				if err != nil {
					zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get last edit for preview event")
				} else if lastEdit != nil {
					h.ReprocessExistingEvent(ctx, lastEdit)
					syncRoom.Events = append(syncRoom.Events, lastEdit)
				}
			}
			syncRoom.Events = append(syncRoom.Events, previewEvent)
		}
	}
	return syncRoom
}

func (h *HiClient) GetInitialSync(ctx context.Context, batchSize int) iter.Seq[*SyncComplete] {
	return func(yield func(*SyncComplete) bool) {
		maxTS := time.Now().Add(1 * time.Hour)
		for i := 0; ; i++ {
			rooms, err := h.DB.Room.GetBySortTS(ctx, maxTS, batchSize)
			if err != nil {
				if ctx.Err() == nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get initial rooms to send to client")
				}
				return
			}
			payload := SyncComplete{
				Rooms:       make(map[id.RoomID]*SyncRoom, len(rooms)-1),
				LeftRooms:   make([]id.RoomID, 0),
				AccountData: make(map[event.Type]*database.AccountData),
			}
			if i == 0 {
				payload.ClearState = true
			}
			for _, room := range rooms {
				if room.SortingTimestamp == rooms[len(rooms)-1].SortingTimestamp {
					break
				}
				maxTS = room.SortingTimestamp.Time
				payload.Rooms[room.ID] = h.getInitialSyncRoom(ctx, room)
				if ctx.Err() != nil {
					return
				}
			}
			if !yield(&payload) || len(rooms) < batchSize {
				break
			}
		}
		// This is last so that the frontend would know about all rooms before trying to fetch custom emoji packs
		ad, err := h.DB.AccountData.GetAllGlobal(ctx, h.Account.UserID)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to get global account data")
			return
		}
		payload := SyncComplete{
			Rooms:       make(map[id.RoomID]*SyncRoom, 0),
			LeftRooms:   make([]id.RoomID, 0),
			AccountData: make(map[event.Type]*database.AccountData, len(ad)),
		}
		for _, data := range ad {
			payload.AccountData[event.Type{Type: data.Type, Class: event.AccountDataEventType}] = data
		}
		yield(&payload)
	}
}
