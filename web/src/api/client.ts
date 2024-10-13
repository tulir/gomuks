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
import { CachedEventDispatcher } from "../util/eventdispatcher.ts"
import RPCClient, { SendMessageParams } from "./rpc.ts"
import { RoomStateStore, StateStore } from "./statestore"
import type { ClientState, EventID, EventRowID, EventType, RPCEvent, RoomID, UserID } from "./types"

export default class Client {
	readonly state = new CachedEventDispatcher<ClientState>()
	readonly store = new StateStore()

	constructor(readonly rpc: RPCClient) {
		this.rpc.event.listen(this.#handleEvent)
	}

	get userID(): UserID {
		return this.state.current?.is_logged_in ? this.state.current.user_id : ""
	}

	#handleEvent = (ev: RPCEvent) => {
		if (ev.command === "client_state") {
			this.state.emit(ev.data)
		} else if (ev.command === "sync_complete") {
			this.store.applySync(ev.data)
		} else if (ev.command === "events_decrypted") {
			this.store.applyDecrypted(ev.data)
		} else if (ev.command === "send_complete") {
			this.store.applySendComplete(ev.data)
		}
	}

	requestEvent(room: RoomStateStore | RoomID | undefined, eventID: EventID) {
		if (typeof room === "string") {
			room = this.store.rooms.get(room)
		}
		if (!room || room.eventsByID.has(eventID)) {
			return
		}
		const sub = room.getEventSubscriber(eventID)
		if (sub.requested) {
			return
		}
		sub.requested = true
		this.rpc.getEvent(room.roomID, eventID).then(
			evt => room.applyEvent(evt),
			err => console.error(`Failed to fetch event ${eventID}`, err),
		)
	}

	async sendMessage(params: SendMessageParams): Promise<void> {
		const room = this.store.rooms.get(params.room_id)
		if (!room) {
			throw new Error("Room not found")
		}
		const dbEvent = await this.rpc.sendMessage(params)
		if (!room.eventsByRowID.has(dbEvent.rowid)) {
			room.pendingEvents.push(dbEvent.rowid)
			room.applyEvent(dbEvent, true)
			room.notifyTimelineSubscribers()
		}
	}

	async loadRoomState(roomID: RoomID, refetch = false): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		}
		const state = await this.rpc.getRoomState(roomID, room.meta.current.has_member_list, refetch)
		const newStateMap: Map<EventType, Map<string, EventRowID>> = new Map()
		for (const evt of state) {
			if (evt.state_key === undefined) {
				throw new Error(`Event ${evt.event_id} is missing state key`)
			}
			room.applyEvent(evt)
			let stateMap = newStateMap.get(evt.type)
			if (!stateMap) {
				stateMap = new Map()
				newStateMap.set(evt.type, stateMap)
			}
			stateMap.set(evt.state_key, evt.rowid)
		}
		room.state = newStateMap
		room.stateLoaded = true
	}

	async loadMoreHistory(roomID: RoomID): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		}
		if (room.paginating) {
			return
		}
		room.paginating = true
		try {
			const oldestRowID = room.timeline[0]?.timeline_rowid
			const resp = await this.rpc.paginate(roomID, oldestRowID ?? 0, 100)
			if (room.timeline[0]?.timeline_rowid !== oldestRowID) {
				throw new Error("Timeline changed while loading history")
			}
			room.applyPagination(resp.events)
		} finally {
			room.paginating = false
		}
	}
}
