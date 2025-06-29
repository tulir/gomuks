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
import type { MouseEvent } from "react"
import { CachedEventDispatcher, NonNullCachedEventDispatcher } from "../util/eventdispatcher.ts"
import RPCClient, { SendMessageParams } from "./rpc.ts"
import { RoomStateStore, StateStore, WidgetListener } from "./statestore"
import type {
	ClientState,
	ElementRecentEmoji,
	EventID,
	EventType,
	GomuksAndroidMessageToWeb,
	ImagePackRooms,
	RPCEvent,
	RawDBEvent,
	RelationType,
	RoomID,
	RoomStateGUID,
	SyncStatus,
	UserID,
} from "./types"

export default class Client {
	readonly state = new CachedEventDispatcher<ClientState>()
	readonly syncStatus = new NonNullCachedEventDispatcher<SyncStatus>({ type: "waiting", error_count: 0 })
	readonly initComplete = new NonNullCachedEventDispatcher<boolean>(false)
	readonly store = new StateStore()
	#stateRequests: RoomStateGUID[] = []
	#stateRequestPromise: Promise<void> | null = null
	#gcInterval: number | undefined
	#toDeviceRequested = false

	constructor(readonly rpc: RPCClient) {
		this.rpc.event.listen(this.#handleEvent)
		this.rpc.connect.listen(() => this.initComplete.emit(false))
		this.store.accountDataSubs.getSubscriber("im.ponies.emote_rooms")(() =>
			queueMicrotask(() => this.#handleEmoteRoomsChange()))
	}

	async #reallyStart(signal: AbortSignal) {
		try {
			const resp = await fetch("_gomuks/auth", {
				method: "POST",
				signal,
			})
			if (!resp.ok && !signal.aborted) {
				this.rpc.connect.emit({
					connected: false,
					reconnecting: false,
					error: `Authentication failed: ${resp.statusText}`,
				})
				return
			}
		} catch (err) {
			this.rpc.connect.emit({ connected: false, reconnecting: false, error: `Authentication failed: ${err}` })
		}
		if (signal.aborted) {
			return
		}
		console.log("Successfully authenticated, connecting to websocket")
		this.rpc.start()
		this.requestNotificationPermission()
	}

	async #reallyStartAndroid(signal: AbortSignal) {
		const androidListener = async (evt: CustomEventInit<string>) => {
			const evtData = JSON.parse(evt.detail ?? "{}") as GomuksAndroidMessageToWeb
			switch (evtData.type) {
			case "register_push":
				await this.rpc.registerPush({
					type: "fcm",
					device_id: evtData.device_id,
					data: evtData.token,
					encryption: evtData.encryption,
					expiration: evtData.expiration,
				})
				return
			case "auth":
				try {
					const resp = await fetch("_gomuks/auth?no_prompt=true", {
						method: "POST",
						headers: {
							Authorization: evtData.authorization,
						},
						signal,
					})
					if (!resp.ok && !signal.aborted) {
						console.error("Failed to authenticate:", resp.status, resp.statusText)
						window.dispatchEvent(new CustomEvent("GomuksWebMessageToAndroid", {
							detail: {
								event: "auth_fail",
								error: `${resp.statusText || resp.status}`,
							},
						}))
						return
					}
				} catch (err) {
					console.error("Failed to authenticate:", err)
					window.dispatchEvent(new CustomEvent("GomuksWebMessageToAndroid", {
						detail: {
							event: "auth_fail",
							error: `${err}`.replace(/^Error: /, ""),
						},
					}))
					return
				}
				if (signal.aborted) {
					return
				}
				console.log("Successfully authenticated, connecting to websocket")
				this.rpc.start()
				return
			}
		}
		const unsubscribeConnect = this.rpc.connect.listen(evt => {
			if (!evt.connected) {
				return
			}
			window.dispatchEvent(new CustomEvent("GomuksWebMessageToAndroid", {
				detail: { event: "connected" },
			}))
		})
		window.addEventListener("GomuksAndroidMessageToWeb", androidListener)
		signal.addEventListener("abort", () => {
			unsubscribeConnect()
			window.removeEventListener("GomuksAndroidMessageToWeb", androidListener)
		})
		window.dispatchEvent(new CustomEvent("GomuksWebMessageToAndroid", {
			detail: { event: "ready" },
		}))
	}

	requestNotificationPermission = (evt?: MouseEvent) => {
		window.Notification?.requestPermission().then(permission => {
			console.log("Notification permission:", permission)
			if (evt) {
				window.alert(`Notification permission: ${permission}`)
			}
		})
	}

	registerURIHandler = () => {
		navigator.registerProtocolHandler("matrix", "#/uri/%s")
	}

	addWidgetListener(listener: WidgetListener): () => void {
		this.store.widgetListeners.add(listener)
		// TODO only request to-device events if there are widgets that need them?
		if (!this.#toDeviceRequested) {
			this.#toDeviceRequested = true
			this.rpc.setListenToDevice(true)
		}
		return () => {
			this.store.widgetListeners.delete(listener)
			if (this.store.widgetListeners.size === 0 && this.#toDeviceRequested) {
				this.#toDeviceRequested = false
				this.rpc.setListenToDevice(false)
			}
		}
	}

	start(): () => void {
		const abort = new AbortController()
		if (window.gomuksAndroid) {
			this.#reallyStartAndroid(abort.signal)
		} else {
			this.#reallyStart(abort.signal)
		}
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
			this.store.userID = ev.data.is_logged_in ? ev.data.user_id : ""
		} else if (ev.command === "sync_status") {
			this.syncStatus.emit(ev.data)
		} else if (ev.command === "init_complete") {
			this.initComplete.emit(true)
		} else if (ev.command === "sync_complete") {
			this.store.applySync(ev.data)
		} else if (ev.command === "events_decrypted") {
			this.store.applyDecrypted(ev.data)
		} else if (ev.command === "send_complete") {
			this.store.applySendComplete(ev.data)
		} else if (ev.command === "image_auth_token") {
			this.store.imageAuthToken = ev.data
		} else if (ev.command === "typing") {
			this.store.applyTyping(ev.data)
		}
	}

	requestMemberEvent(room: RoomStateStore | RoomID | undefined, userID: UserID) {
		if (typeof room === "string") {
			room = this.store.rooms.get(room)
		}
		if (!room || room.state.get("m.room.member")?.has(userID) || room.requestedMembers.has(userID)) {
			return null
		}
		room.requestedMembers.add(userID)
		this.#stateRequests.push({ room_id: room.roomID, type: "m.room.member", state_key: userID })
		if (this.#stateRequestPromise === null) {
			this.#stateRequestPromise = new Promise(this.#doStateRequestsPromise)
		}
		return this.#stateRequestPromise
	}

	#doStateRequestsPromise = (resolve: () => void) => {
		window.queueMicrotask(() => {
			const reqs = this.#stateRequests
			this.#stateRequestPromise = null
			this.#stateRequests = []
			this.loadSpecificRoomState(reqs)
				.catch(err => console.error("Failed to load room state", reqs, err))
				.finally(resolve)
		})
	}

	requestEvent(room: RoomStateStore | RoomID | undefined, eventID: EventID, unredact?: boolean) {
		if (typeof room === "string") {
			room = this.store.rooms.get(room)
		}
		if (!room || (!unredact && room.eventsByID.has(eventID)) ||room.requestedEvents.has(eventID)) {
			return
		}
		room.requestedEvents.add(eventID)
		this.rpc.getEvent(room.roomID, eventID, unredact).then(
			evt => {
				room.applyEvent(evt, false, unredact)
				if (unredact) {
					room.notifyTimelineSubscribers()
				}
			},
			err => {
				console.error(`Failed to fetch event ${eventID}`, err)
				if (unredact) {
					room.requestedEvents.delete(eventID)
					window.alert(`Failed to get unredacted content: ${err}`)
				}
			},
		)
	}

	async getRelatedEvents(room: RoomStateStore | RoomID | undefined, eventID: EventID, relationType?: RelationType) {
		if (typeof room === "string") {
			room = this.store.rooms.get(room)
		}
		if (!room) {
			return []
		}
		const events = await this.rpc.getRelatedEvents(room.roomID, eventID, relationType)
		return events.map(evt => room.getOrApplyEvent(evt))
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

	async sendEvent(
		roomID: RoomID, type: EventType, content: unknown, disableEncryption: boolean = false,
	): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		}
		const dbEvent = await this.rpc.sendEvent(roomID, type, content, disableEncryption)
		this.#handleOutgoingEvent(dbEvent, room)
	}

	async sendMessage(params: SendMessageParams): Promise<void> {
		const room = this.store.rooms.get(params.room_id)
		if (!room) {
			throw new Error("Room not found")
		}
		const dbEvent = await this.rpc.sendMessage(params)
		if (dbEvent) {
			this.#handleOutgoingEvent(dbEvent, room)
		}
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

	async resetTimeline(roomID: RoomID): Promise<void> {
		const room = this.store.rooms.get(roomID)
		if (!room) {
			throw new Error("Room not found")
		} else if (room.paginating) {
			throw new Error("Already paginating")
		}
		room.paginating = true
		try {
			// This part isn't actually required, but it makes it look like something is happening.
			// If the reset is done without the flash, the user might think nothing happened.
			room.timeline = []
			room.hasMoreHistory = false
			room.notifyTimelineSubscribers()

			console.log("Requesting 50 messages of history and a timeline reset in", roomID)
			const resp = await this.rpc.paginate(roomID, 0, 50, true)
			room.hasMoreHistory = resp.has_more
			room.applyPagination(resp.events, resp.related_events, resp.receipts)
		} finally {
			room.paginating = false
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
			room.applyPagination(resp.events, resp.related_events, resp.receipts)
		} finally {
			room.paginating = false
		}
	}

	clearState() {
		this.initComplete.emit(false)
		this.syncStatus.emit({ type: "waiting", error_count: 0 })
		this.state.clearCache()
		this.store.clear()
	}

	async logout() {
		await this.rpc.logout()
		this.clearState()
		localStorage.clear()
	}
}
