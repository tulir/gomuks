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
import { NonNullCachedEventDispatcher } from "../util/eventdispatcher.ts"
import type {
	ContentURI,
	DBRoom,
	EncryptedEventContent,
	EventID,
	EventRowID,
	EventType,
	EventsDecryptedData,
	LazyLoadSummary,
	MemDBEvent,
	RawDBEvent,
	RoomID,
	SendCompleteData,
	SyncCompleteData,
	SyncRoom,
	TimelineRowTuple,
} from "./types"

function arraysAreEqual<T>(arr1?: T[], arr2?: T[]): boolean {
	if (!arr1 || !arr2) {
		return !arr1 && !arr2
	}
	if (arr1.length !== arr2.length) {
		return false
	}
	for (let i = 0; i < arr1.length; i++) {
		if (arr1[i] !== arr2[i]) {
			return false
		}
	}
	return true
}

function llSummaryIsEqual(ll1?: LazyLoadSummary, ll2?: LazyLoadSummary): boolean {
	return ll1?.["m.joined_member_count"] === ll2?.["m.joined_member_count"] &&
		ll1?.["m.invited_member_count"] === ll2?.["m.invited_member_count"] &&
		arraysAreEqual(ll1?.heroes, ll2?.heroes)
}

function visibleMetaIsEqual(meta1: DBRoom, meta2: DBRoom): boolean {
	return meta1.name === meta2.name &&
		meta1.avatar === meta2.avatar &&
		meta1.topic === meta2.topic &&
		meta1.canonical_alias === meta2.canonical_alias &&
		llSummaryIsEqual(meta1.lazy_load_summary, meta2.lazy_load_summary) &&
		meta1.encryption_event?.algorithm === meta2.encryption_event?.algorithm &&
		meta1.has_member_list === meta2.has_member_list
}

export function useRoomTimeline(room: RoomStateStore): (MemDBEvent | null)[] {
	return useSyncExternalStore(
		room.subscribeTimeline,
		() => room.timelineCache,
	)
}

type SubscribeFunc = (callback: () => void) => () => void

export class RoomStateStore {
	readonly roomID: RoomID
	readonly meta: NonNullCachedEventDispatcher<DBRoom>
	timeline: TimelineRowTuple[] = []
	timelineCache: (MemDBEvent | null)[] = []
	state: Map<EventType, Map<string, EventRowID>> = new Map()
	stateLoaded = false
	readonly eventsByRowID: Map<EventRowID, MemDBEvent> = new Map()
	readonly eventsByID: Map<EventID, MemDBEvent> = new Map()
	readonly timelineSubscribers: Set<() => void> = new Set()
	readonly pendingEvents: EventRowID[] = []
	paginating = false

	constructor(meta: DBRoom) {
		this.roomID = meta.room_id
		this.meta = new NonNullCachedEventDispatcher(meta)
	}

	subscribeTimeline: SubscribeFunc = callback => {
		this.timelineSubscribers.add(callback)
		return () => this.timelineSubscribers.delete(callback)
	}

	notifyTimelineSubscribers() {
		this.timelineCache = this.timeline.map(rt => {
			const evt = this.eventsByRowID.get(rt.event_rowid)
			if (!evt) {
				return null
			}
			evt.timeline_rowid = rt.timeline_rowid
			return evt
		}).concat(this.pendingEvents
			.map(rowID => this.eventsByRowID.get(rowID))
			.filter(evt => !!evt))
		for (const sub of this.timelineSubscribers) {
			sub()
		}
	}

	getStateEvent(type: EventType, stateKey: string): MemDBEvent | undefined {
		const rowID = this.state.get(type)?.get(stateKey)
		if (!rowID) {
			return
		}
		return this.eventsByRowID.get(rowID)
	}

	applyPagination(history: RawDBEvent[]) {
		// Pagination comes in newest to oldest, timeline is in the opposite order
		history.reverse()
		const newTimeline = history.map(evt => {
			this.applyEvent(evt)
			return { timeline_rowid: evt.timeline_rowid, event_rowid: evt.rowid }
		})
		this.timeline.splice(0, 0, ...newTimeline)
		this.notifyTimelineSubscribers()
	}

	applyEvent(evt: RawDBEvent, pending: boolean = false) {
		const memEvt = evt as MemDBEvent
		memEvt.mem = true
		memEvt.pending = pending
		if (pending) {
			memEvt.timeline_rowid = 1000000000000000 + memEvt.timestamp
		}
		if (evt.type === "m.room.encrypted" && evt.decrypted && evt.decrypted_type) {
			memEvt.type = evt.decrypted_type
			memEvt.encrypted = evt.content as EncryptedEventContent
			memEvt.content = evt.decrypted
		}
		delete evt.decrypted
		delete evt.decrypted_type
		if (memEvt.last_edit_rowid) {
			memEvt.last_edit = this.eventsByRowID.get(memEvt.last_edit_rowid)
			if (memEvt.last_edit) {
				memEvt.orig_content = memEvt.content
				memEvt.content = memEvt.last_edit.content["m.new_content"]
			}
		}
		this.eventsByRowID.set(memEvt.rowid, memEvt)
		this.eventsByID.set(memEvt.event_id, memEvt)
		if (!pending) {
			const pendingIdx = this.pendingEvents.indexOf(evt.rowid)
			if (pendingIdx !== -1) {
				this.pendingEvents.splice(pendingIdx, 1)
			}
		}
	}

	applySendComplete(evt: RawDBEvent) {
		const existingEvt = this.eventsByRowID.get(evt.rowid)
		if (existingEvt && !existingEvt.pending) {
			return
		}
		this.applyEvent(evt, true)
		this.notifyTimelineSubscribers()
	}

	applySync(sync: SyncRoom) {
		if (visibleMetaIsEqual(this.meta.current, sync.meta)) {
			this.meta.current = sync.meta
		} else {
			this.meta.emit(sync.meta)
		}
		for (const evt of sync.events) {
			this.applyEvent(evt)
		}
		for (const [evtType, changedEvts] of Object.entries(sync.state)) {
			let stateMap = this.state.get(evtType)
			if (!stateMap) {
				stateMap = new Map()
				this.state.set(evtType, stateMap)
			}
			for (const [key, rowID] of Object.entries(changedEvts)) {
				stateMap.set(key, rowID)
			}
		}
		if (sync.reset) {
			this.timeline = sync.timeline
			this.pendingEvents.splice(0, this.pendingEvents.length)
		} else {
			this.timeline.push(...sync.timeline)
		}
		this.notifyTimelineSubscribers()
	}

	applyDecrypted(decrypted: EventsDecryptedData) {
		let timelineChanged = false
		for (const evt of decrypted.events) {
			timelineChanged = timelineChanged || !!this.timeline.find(rt => rt.event_rowid === evt.rowid)
			this.applyEvent(evt)
		}
		if (timelineChanged) {
			this.notifyTimelineSubscribers()
		}
		if (decrypted.preview_event_rowid) {
			this.meta.current.preview_event_rowid = decrypted.preview_event_rowid
		}
	}
}

export interface RoomListEntry {
	room_id: RoomID
	sorting_timestamp: number
	preview_event?: MemDBEvent
	preview_sender?: MemDBEvent
	name: string
	avatar?: ContentURI
}

export class StateStore {
	readonly rooms: Map<RoomID, RoomStateStore> = new Map()
	readonly roomList = new NonNullCachedEventDispatcher<RoomListEntry[]>([])

	#roomListEntryChanged(entry: SyncRoom, oldEntry: RoomStateStore): boolean {
		return entry.meta.sorting_timestamp !== oldEntry.meta.current.sorting_timestamp ||
			entry.meta.preview_event_rowid !== oldEntry.meta.current.preview_event_rowid ||
			entry.events.findIndex(evt => evt.rowid === entry.meta.preview_event_rowid) !== -1
	}

	#makeRoomListEntry(entry: SyncRoom, room?: RoomStateStore): RoomListEntry {
		if (!room) {
			room = this.rooms.get(entry.meta.room_id)
		}
		const preview_event = room?.eventsByRowID.get(entry.meta.preview_event_rowid)
		const preview_sender = preview_event && room?.getStateEvent("m.room.member", preview_event.sender)
		return {
			room_id: entry.meta.room_id,
			sorting_timestamp: entry.meta.sorting_timestamp,
			preview_event,
			preview_sender,
			name: entry.meta.name ?? "Unnamed room",
			avatar: entry.meta.avatar,
		}
	}

	applySync(sync: SyncCompleteData) {
		const resyncRoomList = this.roomList.current.length === 0
		const changedRoomListEntries = new Map<RoomID, RoomListEntry>()
		for (const [roomID, data] of Object.entries(sync.rooms)) {
			let isNewRoom = false
			let room = this.rooms.get(roomID)
			if (!room) {
				room = new RoomStateStore(data.meta)
				this.rooms.set(roomID, room)
				isNewRoom = true
			}
			const roomListEntryChanged = !resyncRoomList && (isNewRoom || this.#roomListEntryChanged(data, room))
			room.applySync(data)
			if (roomListEntryChanged) {
				changedRoomListEntries.set(roomID, this.#makeRoomListEntry(data, room))
			}
		}

		let updatedRoomList: RoomListEntry[] | undefined
		if (resyncRoomList) {
			updatedRoomList = Object.values(sync.rooms).map(entry => this.#makeRoomListEntry(entry))
			updatedRoomList.sort((r1, r2) => r1.sorting_timestamp - r2.sorting_timestamp)
		} else if (changedRoomListEntries.size > 0) {
			updatedRoomList = this.roomList.current.filter(entry => !changedRoomListEntries.has(entry.room_id))
			for (const entry of changedRoomListEntries.values()) {
				if (updatedRoomList.length === 0 || entry.sorting_timestamp >=
					updatedRoomList[updatedRoomList.length - 1].sorting_timestamp) {
					updatedRoomList.push(entry)
				} else if (entry.sorting_timestamp <= 0 ||
					entry.sorting_timestamp < updatedRoomList[0]?.sorting_timestamp) {
					updatedRoomList.unshift(entry)
				} else {
					const indexToPushAt = updatedRoomList.findLastIndex(val =>
						val.sorting_timestamp <= entry.sorting_timestamp)
					updatedRoomList.splice(indexToPushAt + 1, 0, entry)
				}
			}
		}
		if (updatedRoomList) {
			this.roomList.emit(updatedRoomList)
		}
	}

	applySendComplete(data: SendCompleteData) {
		const room = this.rooms.get(data.event.room_id)
		if (!room) {
			// TODO log or something?
			return
		}
		room.applySendComplete(data.event)
	}

	applyDecrypted(decrypted: EventsDecryptedData) {
		const room = this.rooms.get(decrypted.room_id)
		if (!room) {
			// TODO log or something?
			return
		}
		room.applyDecrypted(decrypted)
		if (decrypted.preview_event_rowid) {
			const idx = this.roomList.current.findIndex(entry => entry.room_id === decrypted.room_id)
			if (idx !== -1) {
				const updatedRoomList = [...this.roomList.current]
				updatedRoomList[idx] = {
					...updatedRoomList[idx],
					preview_event: room.eventsByRowID.get(decrypted.preview_event_rowid),
				}
				this.roomList.emit(updatedRoomList)
			}
		}
	}
}
