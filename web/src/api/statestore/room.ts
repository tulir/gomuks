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
import { NonNullCachedEventDispatcher } from "@/util/eventdispatcher.ts"
import Subscribable, { MultiSubscribable } from "@/util/subscribable.ts"
import type {
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
	SyncRoom,
	TimelineRowTuple,
} from "../types"

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

export class RoomStateStore {
	readonly roomID: RoomID
	readonly meta: NonNullCachedEventDispatcher<DBRoom>
	timeline: TimelineRowTuple[] = []
	timelineCache: (MemDBEvent | null)[] = []
	state: Map<EventType, Map<string, EventRowID>> = new Map()
	stateLoaded = false
	readonly eventsByRowID: Map<EventRowID, MemDBEvent> = new Map()
	readonly eventsByID: Map<EventID, MemDBEvent> = new Map()
	readonly timelineSub = new Subscribable()
	readonly stateSubs = new MultiSubscribable()
	readonly eventSubs = new MultiSubscribable()
	readonly requestedEvents: Set<EventID> = new Set()
	readonly openNotifications: Map<EventRowID, Notification> = new Map()
	readonly pendingEvents: EventRowID[] = []
	paginating = false
	paginationRequestedForRow = -1
	readUpToRow = -1

	constructor(meta: DBRoom) {
		this.roomID = meta.room_id
		this.meta = new NonNullCachedEventDispatcher(meta)
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
		this.timelineSub.notify()
	}

	stateSubKey(eventType: EventType, stateKey: string): string {
		return `${eventType}:${stateKey}`
	}

	getStateEvent(type: EventType, stateKey: string): MemDBEvent | undefined {
		const rowID = this.state.get(type)?.get(stateKey)
		if (!rowID) {
			return
		}
		return this.eventsByRowID.get(rowID)
	}

	getPinnedEvents(): EventID[] {
		const pinnedList = this.getStateEvent("m.room.pinned_events", "")?.content?.pinned
		if (Array.isArray(pinnedList)) {
			return pinnedList.filter(evtID => typeof evtID === "string")
		}
		return []
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
				memEvt.local_content = memEvt.last_edit.local_content
			}
		} else if (memEvt.relation_type === "m.replace" && memEvt.relates_to) {
			const editTarget = this.eventsByID.get(memEvt.relates_to)
			if (editTarget?.last_edit_rowid === memEvt.rowid && !editTarget.last_edit) {
				this.eventsByRowID.set(editTarget.rowid, {
					...editTarget,
					last_edit: memEvt,
					orig_content: editTarget.content,
					content: memEvt.content["m.new_content"],
					local_content: memEvt.local_content,
				})
				this.eventSubs.notify(editTarget.event_id)
			}
		}
		this.eventsByRowID.set(memEvt.rowid, memEvt)
		this.eventsByID.set(memEvt.event_id, memEvt)
		this.requestedEvents.delete(memEvt.event_id)
		if (!pending) {
			const pendingIdx = this.pendingEvents.indexOf(memEvt.rowid)
			if (pendingIdx !== -1) {
				this.pendingEvents.splice(pendingIdx, 1)
			}
		}
		this.eventSubs.notify(memEvt.event_id)
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
				this.stateSubs.notify(this.stateSubKey(evtType, key))
			}
		}
		if (sync.reset) {
			this.timeline = sync.timeline
			this.pendingEvents.splice(0, this.pendingEvents.length)
		} else {
			this.timeline.push(...sync.timeline)
		}
		if (sync.meta.unread_notifications === 0 && sync.meta.unread_highlights === 0) {
			for (const notif of this.openNotifications.values()) {
				notif.close()
			}
			this.openNotifications.clear()
		}
		this.notifyTimelineSubscribers()
	}

	applyFullState(state: RawDBEvent[]) {
		const newStateMap: Map<EventType, Map<string, EventRowID>> = new Map()
		for (const evt of state) {
			if (evt.state_key === undefined) {
				throw new Error(`Event ${evt.event_id} is missing state key`)
			}
			this.applyEvent(evt)
			let stateMap = newStateMap.get(evt.type)
			if (!stateMap) {
				stateMap = new Map()
				newStateMap.set(evt.type, stateMap)
			}
			stateMap.set(evt.state_key, evt.rowid)
		}
		this.state = newStateMap
		this.stateLoaded = true
		for (const [evtType, stateMap] of newStateMap) {
			for (const [key] of stateMap) {
				this.stateSubs.notify(this.stateSubKey(evtType, key))
			}
		}
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
