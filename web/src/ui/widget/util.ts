// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import type { IRoomEvent } from "matrix-widget-api"
import type { RoomStateStore } from "@/api/statestore"
import type { MemDBEvent } from "@/api/types"

export function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null
}

export function notNull<T>(value: T | null | undefined): value is T {
	return value !== null && value !== undefined
}

export function memDBEventToIRoomEvent(evt: MemDBEvent): IRoomEvent {
	return {
		type: evt.type,
		sender: evt.sender,
		event_id: evt.event_id,
		room_id: evt.room_id,
		state_key: evt.state_key,
		origin_server_ts: evt.timestamp,
		content: evt.content,
		unsigned: evt.unsigned,
	}
}

export function * iterRoomTimeline(room: RoomStateStore, since: string | undefined) {
	const tc = room.timelineCache
	for (let i = tc.length - 1; i >= 0; i--) {
		const evt = tc[i]!
		if (evt.event_id === since) {
			return
		}
		yield evt
	}
}

export function filterEvent(eventType: string, msgtype: string | undefined, stateKey: string | undefined) {
	return (evt: MemDBEvent) => evt.type === eventType
		&& (!msgtype || evt.content.msgtype === msgtype)
		&& (!stateKey || evt.state_key === stateKey)
}
