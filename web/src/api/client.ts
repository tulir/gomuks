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
import type {
	ClientWellKnown, DBEvent, EventID, EventRowID, EventType, RoomID, TimelineRowID, UserID,
} from "./types/hitypes.ts"
import { ClientState, RPCEvent } from "./types/hievents.ts"
import { RPCClient } from "./rpc.ts"
import { CachedEventDispatcher } from "../util/eventdispatcher.ts"
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

	request<Req, Resp>(command: string, data: Req): Promise<Resp> {
		return this.rpc.request(command, data)
	}

	sendMessage(room_id: RoomID, event_type: EventType, content: Record<string, unknown>): Promise<DBEvent> {
		return this.request("send_message", { room_id, event_type, content })
	}

	ensureGroupSessionShared(room_id: RoomID): Promise<boolean> {
		return this.request("ensure_group_session_shared", { room_id })
	}

	getEvent(room_id: RoomID, event_id: EventID): Promise<DBEvent> {
		return this.request("get_event", { room_id, event_id })
	}

	getEventsByRowIDs(row_ids: EventRowID[]): Promise<DBEvent[]> {
		return this.request("get_events_by_row_ids", { row_ids })
	}

	paginate(room_id: RoomID, max_timeline_id: TimelineRowID, limit: number): Promise<DBEvent[]> {
		return this.request("paginate", { room_id, max_timeline_id, limit })
	}

	paginateServer(room_id: RoomID, limit: number): Promise<DBEvent[]> {
		return this.request("paginate_server", { room_id, limit })
	}

	discoverHomeserver(user_id: UserID): Promise<ClientWellKnown> {
		return this.request("discover_homeserver", { user_id })
	}

	login(homeserver_url: string, username: string, password: string): Promise<boolean> {
		return this.request("login", { homeserver_url, username, password })
	}

	verify(recovery_key: string): Promise<boolean> {
		return this.request("verify", { recovery_key })
	}
}
