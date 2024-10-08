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
import type {
	RoomID,
} from "./types/hitypes.ts"
import type { ClientState, RPCEvent } from "./types/hievents.ts"
import type RPCClient from "./rpc.ts"
import { StateStore } from "./statestore.ts"

export default class Client {
	readonly state = new CachedEventDispatcher<ClientState>()
	readonly store = new StateStore()

	constructor(readonly rpc: RPCClient) {
		this.rpc.event.listen(this.#handleEvent)
	}

	#handleEvent = (ev: RPCEvent) => {
		if (ev.command === "client_state") {
			this.state.emit(ev.data)
		} else if (ev.command === "sync_complete") {
			this.store.applySync(ev.data)
		} else if (ev.command === "events_decrypted") {
			this.store.applyDecrypted(ev.data)
		}
	}

	async loadMoreHistory(roomID: RoomID): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		}
		const oldestRowID = room.timeline.current[0]?.timeline_rowid
		const resp = await this.rpc.paginate(roomID, oldestRowID ?? 0, 100)
		if (room.timeline.current[0]?.timeline_rowid !== oldestRowID) {
			throw new Error("Timeline changed while loading history")
		}
		room.applyPagination(resp.events)
	}
}
