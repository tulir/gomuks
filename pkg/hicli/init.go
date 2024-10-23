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
	if room.PreviewEventRowID != 0 {
		previewEvent, err := h.DB.Event.GetByRowID(ctx, room.PreviewEventRowID)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to get preview event for room")
		} else if previewEvent != nil {
			h.ReprocessExistingEvent(ctx, previewEvent)
			previewMember, err := h.DB.CurrentState.Get(ctx, room.ID, event.StateMember, previewEvent.Sender.String())
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("Failed to get preview member event for room")
			} else if previewMember != nil {
				syncRoom.Events = append(syncRoom.Events, previewMember)
				syncRoom.State[event.StateMember] = map[string]database.EventRowID{
					*previewMember.StateKey: previewMember.RowID,
				}
			}
			if previewEvent.LastEditRowID != nil {
				lastEdit, err := h.DB.Event.GetByRowID(ctx, *previewEvent.LastEditRowID)
				if err != nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get last edit for preview event")
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
		for {
			rooms, err := h.DB.Room.GetBySortTS(ctx, maxTS, batchSize)
			if err != nil {
				if ctx.Err() == nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get initial rooms to send to client")
				}
				return
			}
			payload := SyncComplete{
				Rooms:     make(map[id.RoomID]*SyncRoom, len(rooms)-1),
				LeftRooms: make([]id.RoomID, 0),
			}
			for _, room := range rooms {
				if room.SortingTimestamp == rooms[len(rooms)-1].SortingTimestamp {
					break
				}
				maxTS = room.SortingTimestamp.Time
				payload.Rooms[room.ID] = h.getInitialSyncRoom(ctx, room)
			}
			if !yield(&payload) || len(rooms) < batchSize {
				break
			}
		}
	}
}
