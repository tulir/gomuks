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
import { CachedEventDispatcher, NonNullCachedEventDispatcher } from "../util/eventdispatcher.ts"
import RPCClient, { SendMessageParams } from "./rpc.ts"
import { RoomStateStore, StateStore } from "./statestore"
import type {
	ClientState,
	ElementRecentEmoji,
	EventID,
	EventType,
	ImagePackRooms,
	RPCEvent,
	RawDBEvent,
	RoomID,
	RoomStateGUID,
	SyncStatus,
	UserID,
} from "./types"

export default class Client {
	readonly state = new CachedEventDispatcher<ClientState>()
	readonly syncStatus = new NonNullCachedEventDispatcher<SyncStatus>({ type: "waiting", error_count: 0 })
	readonly store = new StateStore()
	#stateRequests: RoomStateGUID[] = []
	#stateRequestQueued = false
	#gcInterval: number | undefined

	constructor(readonly rpc: RPCClient) {
		this.rpc.event.listen(this.#handleEvent)
		this.store.accountDataSubs.getSubscriber("im.ponies.emote_rooms")(() =>
			queueMicrotask(() => this.#handleEmoteRoomsChange()))
	}

	async #reallyStart(signal: AbortSignal) {
		try {
			const resp = await fetch("_gomuks/auth", {
				method: "POST",
				signal,
			})
			if (!resp.ok) {
				this.rpc.connect.emit({
					connected: false,
					error: new Error(`Authentication failed: ${resp.statusText}`),
				})
				return
			}
		} catch (err) {
			const error = err instanceof Error ? err : new Error(`${err}`)
			this.rpc.connect.emit({ connected: false, error })
		}
		if (signal.aborted) {
			return
		}
		console.log("Successfully authenticated, connecting to websocket")
		this.rpc.start()
		Notification.requestPermission()
			.then(permission => console.log("Notification permission:", permission))
	}

	start(): () => void {
		const abort = new AbortController()
		this.#reallyStart(abort.signal)
		this.#gcInterval = setInterval(() => {
			console.log("Garbage collection completed:", this.store.doGarbageCollection())
		}, window.gcSettings.interval)
		return () => {
			abort.abort()
			this.rpc.stop()
			clearInterval(this.#gcInterval)
		}
	}

	get userID(): UserID {
		return this.state.current?.is_logged_in ? this.state.current.user_id : ""
	}

	#handleEvent = (ev: RPCEvent) => {
		if (ev.command === "client_state") {
			this.state.emit(ev.data)
		} else if (ev.command === "sync_status") {
			this.syncStatus.emit(ev.data)
		} else if (ev.command === "sync_complete") {
			this.store.applySync(ev.data)
		} else if (ev.command === "events_decrypted") {
			this.store.applyDecrypted(ev.data)
		} else if (ev.command === "send_complete") {
			this.store.applySendComplete(ev.data)
		} else if (ev.command === "image_auth_token") {
			this.store.imageAuthToken = ev.data
		}
	}

	requestMemberEvent(room: RoomStateStore | RoomID | undefined, userID: UserID) {
		if (typeof room === "string") {
			room = this.store.rooms.get(room)
		}
		if (!room || room.state.get("m.room.member")?.has(userID) || room.requestedMembers.has(userID)) {
			return
		}
		room.requestedMembers.add(userID)
		this.#stateRequests.push({ room_id: room.roomID, type: "m.room.member", state_key: userID })
		if (!this.#stateRequestQueued) {
			this.#stateRequestQueued = true
			window.queueMicrotask(this.doStateRequests)
		}
	}

	doStateRequests = () => {
		const reqs = this.#stateRequests
		this.#stateRequestQueued = false
		this.#stateRequests = []
		this.loadSpecificRoomState(reqs).catch(err => console.error("Failed to load room state", reqs, err))
	}

	requestEvent(room: RoomStateStore | RoomID | undefined, eventID: EventID) {
		if (typeof room === "string") {
			room = this.store.rooms.get(room)
		}
		if (!room || room.eventsByID.has(eventID) || room.requestedEvents.has(eventID)) {
			return
		}
		room.requestedEvents.add(eventID)
		this.rpc.getEvent(room.roomID, eventID).then(
			evt => room.applyEvent(evt),
			err => console.error(`Failed to fetch event ${eventID}`, err),
		)
	}

	async pinMessage(room: RoomStateStore, evtID: EventID, wantPinned: boolean) {
		const pinnedEvents = room.getPinnedEvents()
		const currentlyPinned = pinnedEvents.includes(evtID)
		if (currentlyPinned === wantPinned) {
			return
		}
		if (wantPinned) {
			pinnedEvents.push(evtID)
		} else {
			const idx = pinnedEvents.indexOf(evtID)
			if (idx !== -1) {
				pinnedEvents.splice(idx, 1)
			}
		}
		await this.rpc.setState(room.roomID, "m.room.pinned_events", "", { pinned: pinnedEvents })
	}

	async resendEvent(txnID: string): Promise<void> {
		const dbEvent = await this.rpc.resendEvent(txnID)
		const room = this.store.rooms.get(dbEvent.room_id)
		room?.applyEvent(dbEvent, true)
		room?.notifyTimelineSubscribers()
	}

	#handleOutgoingEvent(dbEvent: RawDBEvent, room: RoomStateStore) {
		if (!room.eventsByRowID.has(dbEvent.rowid)) {
			if (!room.pendingEvents.includes(dbEvent.rowid)) {
				room.pendingEvents.push(dbEvent.rowid)
			}
			room.applyEvent(dbEvent, true)
			room.notifyTimelineSubscribers()
		}
	}

	async sendEvent(roomID: RoomID, type: EventType, content: unknown): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		}
		const dbEvent = await this.rpc.sendEvent(roomID, type, content)
		this.#handleOutgoingEvent(dbEvent, room)
	}

	async sendMessage(params: SendMessageParams): Promise<void> {
		const room = this.store.rooms.get(params.room_id)
		if (!room) {
			throw new Error("Room not found")
		}
		const dbEvent = await this.rpc.sendMessage(params)
		this.#handleOutgoingEvent(dbEvent, room)
	}

	async subscribeToEmojiPack(pack: RoomStateGUID, subscribe: boolean = true) {
		const emoteRooms = (this.store.accountData.get("im.ponies.emote_rooms") ?? {}) as ImagePackRooms
		if (!emoteRooms.rooms) {
			emoteRooms.rooms = {}
		}
		if (!emoteRooms.rooms[pack.room_id]) {
			emoteRooms.rooms[pack.room_id] = {}
		}
		if (emoteRooms.rooms[pack.room_id][pack.state_key]) {
			if (subscribe) {
				return
			}
			delete emoteRooms.rooms[pack.room_id][pack.state_key]
		} else {
			if (!subscribe) {
				return
			}
			emoteRooms.rooms[pack.room_id][pack.state_key] = {}
		}
		console.log("Changing subscription state for emoji pack", pack, "to", subscribe)
		await this.rpc.setAccountData("im.ponies.emote_rooms", emoteRooms)
	}

	async incrementFrequentlyUsedEmoji(targetEmoji: string) {
		const content = Object.assign({}, this.store.accountData.get("io.element.recent_emoji")) as ElementRecentEmoji
		if (!Array.isArray(content.recent_emoji)) {
			content.recent_emoji = []
		}
		let found = false
		for (const [idx, [emoji, count]] of content.recent_emoji.entries()) {
			if (emoji === targetEmoji) {
				content.recent_emoji.splice(idx, 1)
				content.recent_emoji.unshift([emoji, count + 1])
				found = true
				break
			}
		}
		if (!found) {
			content.recent_emoji.unshift([targetEmoji, 1])
		}
		if (content.recent_emoji.length > 100) {
			content.recent_emoji.pop()
		}
		this.store.accountData.set("io.element.recent_emoji", content)
		await this.rpc.setAccountData("io.element.recent_emoji", content)
	}

	#handleEmoteRoomsChange() {
		this.store.invalidateEmojiPackKeyCache()
		const keys = this.store.getEmojiPackKeys()
		console.log("Loading subscribed emoji pack states", keys)
		this.loadSpecificRoomState(keys).then(
			() => this.store.emojiRoomsSub.notify(),
			err => console.error("Failed to load emote rooms", err),
		)
	}

	async loadSpecificRoomState(keys: RoomStateGUID[]): Promise<void> {
		const missingKeys = keys.filter(key => {
			const room = this.store.rooms.get(key.room_id)
			return room && room.getStateEvent(key.type, key.state_key) === undefined
		})
		if (missingKeys.length === 0) {
			return
		}
		const events = await this.rpc.getSpecificRoomState(missingKeys)
		for (const evt of events) {
			this.store.rooms.get(evt.room_id)?.applyState(evt)
		}
	}

	async loadRoomState(
		roomID: RoomID, { omitMembers, refetch } = { omitMembers: true, refetch: false },
	): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		}
		if (!omitMembers) {
			room.membersRequested = true
			console.log("Requesting full member list for", roomID)
		}
		const state = await this.rpc.getRoomState(roomID, !omitMembers, !room.meta.current.has_member_list, refetch)
		room.applyFullState(state, omitMembers)
		if (!omitMembers && !room.meta.current.has_member_list) {
			room.meta.current.has_member_list = true
		}
	}

	async loadMoreHistory(roomID: RoomID): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		} else if (room.paginating) {
			throw new Error("Already paginating")
		}
		room.paginating = true
		try {
			const oldestRowID = room.timeline[0]?.timeline_rowid
			// Request 50 messages at a time first, increase batch size when going further
			const count = room.timeline.length < 100 ? 50 : 100
			console.log("Requesting", count, "messages of history in", roomID)
			const resp = await this.rpc.paginate(roomID, oldestRowID ?? 0, count)
			if (room.timeline[0]?.timeline_rowid !== oldestRowID) {
				throw new Error("Timeline changed while loading history")
			}
			room.hasMoreHistory = resp.has_more
			room.applyPagination(resp.events)
		} finally {
			room.paginating = false
		}
	}

	async logout() {
		await this.rpc.logout()
		localStorage.clear()
		this.store.clear()
	}
}
