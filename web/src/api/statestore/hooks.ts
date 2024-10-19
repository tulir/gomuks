// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
import { useSyncExternalStore } from "react"
import type { EventID, MemDBEvent } from "../types"
import { RoomStateStore } from "./room.ts"

export function useRoomTimeline(room: RoomStateStore): (MemDBEvent | null)[] {
	return useSyncExternalStore(
		room.timelineSub.subscribe,
		() => room.timelineCache,
	)
}

const noopSubscribe = () => () => {}
const returnNull = () => null

export function useRoomEvent(room: RoomStateStore, eventID: EventID | null): MemDBEvent | null {
	return useSyncExternalStore(
		eventID ? room.getEventSubscriber(eventID).subscribe : noopSubscribe,
		eventID ? (() => room.eventsByID.get(eventID) ?? null) : returnNull,
	)
}
