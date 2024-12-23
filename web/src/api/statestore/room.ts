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
import { Preferences, getLocalStoragePreferences, getPreferenceProxy } from "@/api/types/preferences"
import { CustomEmojiPack, parseCustomEmojiPack } from "@/util/emoji"
import { NonNullCachedEventDispatcher } from "@/util/eventdispatcher.ts"
import toSearchableString from "@/util/searchablestring.ts"
import Subscribable, { MultiSubscribable, NoDataSubscribable } from "@/util/subscribable.ts"
import { getDisplayname } from "@/util/validation.ts"
import {
	ContentURI,
	DBReceipt,
	DBRoom,
	EncryptedEventContent,
	EventID,
	EventRowID,
	EventType,
	EventsDecryptedData,
	ImagePack,
	LazyLoadSummary,
	MemDBEvent,
	MemReceipt,
	MemberEventContent,
	PowerLevelEventContent,
	RawDBEvent,
	RoomID,
	SyncRoom,
	TimelineRowTuple,
	UnknownEventContent,
	UserID,
	roomStateGUIDToString,
} from "../types"
import type { StateStore } from "./main.ts"

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

export interface AutocompleteMemberEntry {
	userID: UserID
	displayName: string
	avatarURL?: ContentURI
	searchString: string
	event: MemDBEvent
}

const collator = new Intl.Collator()

const UNSENT_TIMELINE_ROWID_BASE = 1000000000000000

export class RoomStateStore {
	readonly roomID: RoomID
	readonly meta: NonNullCachedEventDispatcher<DBRoom>
	timeline: TimelineRowTuple[] = []
	timelineCache: (MemDBEvent | null)[] = []
	editTargets: EventRowID[] = []
	state: Map<EventType, Map<string, EventRowID>> = new Map()
	stateLoaded = false
	typing: UserID[] = []
	fullMembersLoaded = false
	readonly eventsByRowID: Map<EventRowID, MemDBEvent> = new Map()
	readonly eventsByID: Map<EventID, MemDBEvent> = new Map()
	readonly timelineSub = new Subscribable()
	readonly typingSub = new Subscribable()
	readonly stateSubs = new MultiSubscribable()
	readonly eventSubs = new MultiSubscribable()
	readonly receiptsByEventID: Map<EventID, MemReceipt[]> = new Map()
	readonly receiptsByUserID: Map<UserID, MemReceipt> = new Map()
	readonly receiptSubs = new MultiSubscribable()
	readonly requestedEvents: Set<EventID> = new Set()
	readonly requestedMembers: Set<UserID> = new Set()
	readonly accountData: Map<string, UnknownEventContent> = new Map()
	readonly accountDataSubs = new MultiSubscribable()
	readonly openNotifications: Map<EventRowID, Notification> = new Map()
	readonly #emojiPacksCache: Map<string, CustomEmojiPack | null> = new Map()
	readonly preferences: Required<Preferences>
	readonly localPreferenceCache: Preferences
	readonly preferenceSub = new NoDataSubscribable()
	serverPreferenceCache: Preferences = {}
	#membersCache: AutocompleteMemberEntry[] | null = null
	membersRequested: boolean = false
	#allPacksCache: Record<string, CustomEmojiPack> | null = null
	lastOpened: number = 0
	readonly pendingEvents: EventRowID[] = []
	paginating = false
	paginationRequestedForRow = -1
	readUpToRow = -1
	hasMoreHistory = true
	hidden = false

	constructor(meta: DBRoom, private parent: StateStore) {
		this.roomID = meta.room_id
		this.meta = new NonNullCachedEventDispatcher(meta)
		this.localPreferenceCache = getLocalStoragePreferences(`prefs-${this.roomID}`, this.preferenceSub.notify)
		this.preferences = getPreferenceProxy(parent, this)
	}

	#updateTimelineCache() {
		const ownMessages: EventRowID[] = []
		this.timelineCache = this.timeline.map(rt => {
			const evt = this.eventsByRowID.get(rt.event_rowid)
			if (!evt) {
				return null
			}
			evt.timeline_rowid = rt.timeline_rowid
			if (
				evt.sender === this.parent.userID
				&& evt.type === "m.room.message"
				&& evt.relation_type !== "m.replace"
			) {
				ownMessages.push(evt.rowid)
			}
			return evt
		}).concat(this.pendingEvents
			.map(rowID => this.eventsByRowID.get(rowID))
			.filter(evt => !!evt))
		this.editTargets = ownMessages
	}

	notifyTimelineSubscribers() {
		this.#updateTimelineCache()
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

	getEmojiPack(key: string): CustomEmojiPack | null {
		if (!this.#emojiPacksCache.has(key)) {
			const pack = this.getStateEvent("im.ponies.room_emotes", key)?.content
			if (!pack || !pack.images) {
				this.#emojiPacksCache.set(key, null)
				return null
			}
			const fallbackName = key === ""
				? this.meta.current.name : `${this.meta.current.name} - ${key}`
			const packID = roomStateGUIDToString({
				room_id: this.roomID,
				type: "im.ponies.room_emotes",
				state_key: key,
			})
			this.#emojiPacksCache.set(key, parseCustomEmojiPack(pack as ImagePack, packID, fallbackName))
		}
		return this.#emojiPacksCache.get(key) ?? null
	}

	getAllEmojiPacks(): Record<string, CustomEmojiPack> {
		if (this.#allPacksCache === null) {
			this.#allPacksCache = Object.fromEntries(
				this.state.get("im.ponies.room_emotes")?.keys()
					.map(stateKey => {
						const pack = this.getEmojiPack(stateKey)
						return pack ? [pack.id, pack] : null
					})
					.filter((res): res is [string, CustomEmojiPack] => !!res) ?? [],
			)
		}
		return this.#allPacksCache
	}

	#fillMembersCache() {
		const memberEvtIDs = this.state.get("m.room.member")
		if (!memberEvtIDs) {
			return
		}
		const powerLevels: PowerLevelEventContent = this.getStateEvent("m.room.power_levels", "")?.content ?? {}
		const membersCache = memberEvtIDs.values()
			.map(rowID => this.eventsByRowID.get(rowID))
			.filter((evt): evt is MemDBEvent => !!evt && evt.content.membership === "join")
			.map((evt): AutocompleteMemberEntry => ({
				userID: evt.state_key!,
				displayName: getDisplayname(evt.state_key!, evt.content as MemberEventContent),
				avatarURL: evt.content?.avatar_url,
				searchString: toSearchableString(`${evt.content?.displayname ?? ""}${evt.state_key!.slice(1)}`),
				event: evt,
			}))
			.toArray()
		membersCache.sort((a, b) => {
			const aPower = powerLevels.users?.[a.userID] ?? powerLevels.users_default ?? 0
			const bPower = powerLevels.users?.[b.userID] ?? powerLevels.users_default ?? 0
			if (aPower !== bPower) {
				return bPower - aPower
			} else if (a.displayName === b.displayName) {
				return a.userID.localeCompare(b.userID)
			} else {
				return collator.compare(a.displayName, b.displayName)
			}
		})
		this.#membersCache = membersCache
		return membersCache
	}

	getMembers = (): AutocompleteMemberEntry[] => {
		if (this.#membersCache === null) {
			this.#fillMembersCache()
		}
		return this.#membersCache ?? []
	}

	getPinnedEvents(): EventID[] {
		const pinnedList = this.getStateEvent("m.room.pinned_events", "")?.content?.pinned
		if (Array.isArray(pinnedList)) {
			return pinnedList.filter(evtID => typeof evtID === "string")
		}
		return []
	}

	applyPagination(history: RawDBEvent[], related: RawDBEvent[], allReceipts: Record<EventID, DBReceipt[]>) {
		// Pagination comes in newest to oldest, timeline is in the opposite order
		history.reverse()
		const newTimeline = history.map(evt => {
			this.applyEvent(evt)
			return { timeline_rowid: evt.timeline_rowid, event_rowid: evt.rowid }
		})
		for (const evt of related) {
			if (!this.eventsByRowID.has(evt.rowid)) {
				this.applyEvent(evt)
			}
		}
		this.timeline.splice(0, 0, ...newTimeline)
		this.notifyTimelineSubscribers()
		for (const [evtID, receipts] of Object.entries(allReceipts)) {
			this.applyReceipts(receipts, evtID, true)
		}
	}

	applyReceipts(receipts: DBReceipt[], evtID: EventID, override: boolean) {
		const evt = this.eventsByID.get(evtID)
		if (!evt?.timeline_rowid) {
			return
		}
		const filtered = receipts.filter(receipt => this.applyReceipt(receipt, evt))
		filtered.sort((a, b) => a.timestamp - b.timestamp)
		if (override) {
			this.receiptsByEventID.set(evtID, filtered)
		} else {
			const existing = this.receiptsByEventID.get(evtID) ?? []
			this.receiptsByEventID.set(evtID, existing.concat(filtered))
		}
		this.receiptSubs.notify(evtID)
	}

	applyReceipt(receipt: DBReceipt, evt: MemDBEvent): receipt is MemReceipt {
		const existingReceipt = this.receiptsByUserID.get(receipt.user_id)
		if (existingReceipt) {
			if (existingReceipt.timeline_rowid >= evt.timeline_rowid) {
				return false
			}
			const oldArr = this.receiptsByEventID.get(existingReceipt.event_id)
			if (oldArr) {
				const updated = oldArr.filter(r => r !== existingReceipt)
				if (updated.length !== oldArr.length) {
					if (updated.length === 0) {
						this.receiptsByEventID.delete(existingReceipt.event_id)
					} else {
						this.receiptsByEventID.set(existingReceipt.event_id, updated)
					}
					this.receiptSubs.notify(existingReceipt.event_id)
				}
			}
		}
		const memReceipt = receipt as MemReceipt
		memReceipt.timeline_rowid = evt.timeline_rowid > UNSENT_TIMELINE_ROWID_BASE ? 1 : evt.timeline_rowid
		memReceipt.event_rowid = evt.rowid
		this.receiptsByUserID.set(receipt.user_id, memReceipt)
		return true
	}

	applyEvent(evt: RawDBEvent, pending: boolean = false) {
		const memEvt = evt as MemDBEvent
		memEvt.mem = true
		memEvt.pending = pending
		if (pending) {
			memEvt.timeline_rowid = UNSENT_TIMELINE_ROWID_BASE + memEvt.timestamp
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
		return memEvt
	}

	applySendComplete(evt: RawDBEvent) {
		const existingEvt = this.eventsByRowID.get(evt.rowid)
		if (existingEvt && !existingEvt.pending) {
			return
		}
		this.applyEvent(evt, true)
		this.notifyTimelineSubscribers()
	}

	invalidateStateCaches(evtType: string, key: string) {
		if (evtType === "im.ponies.room_emotes") {
			this.#emojiPacksCache.delete(key)
			this.#allPacksCache = null
			this.parent.invalidateEmojiPacksCache()
		} else if (evtType === "m.room.member") {
			this.#membersCache = null
			this.requestedMembers.delete(key as UserID)
		} else if (evtType === "m.room.power_levels") {
			this.#membersCache = null
		}
		this.stateSubs.notify(this.stateSubKey(evtType, key))
	}

	applySync(sync: SyncRoom) {
		if (visibleMetaIsEqual(this.meta.current, sync.meta)) {
			this.meta.current = sync.meta
		} else {
			this.meta.emit(sync.meta)
		}
		for (const ad of Object.values(sync.account_data)) {
			if (ad.type === "fi.mau.gomuks.preferences") {
				this.serverPreferenceCache = ad.content
				this.preferenceSub.notify()
			}
			this.accountData.set(ad.type, ad.content)
			this.accountDataSubs.notify(ad.type)
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
				this.invalidateStateCaches(evtType, key)
			}
			this.stateSubs.notify(evtType)
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
		for (const [evtID, receipts] of Object.entries(sync.receipts)) {
			this.applyReceipts(receipts, evtID, false)
		}
	}

	applyState(evt: RawDBEvent) {
		if (evt.state_key === undefined) {
			throw new Error(`Event ${evt.event_id} is missing state key`)
		}
		this.applyEvent(evt)
		let stateMap = this.state.get(evt.type)
		if (!stateMap) {
			stateMap = new Map()
			this.state.set(evt.type, stateMap)
		}
		stateMap.set(evt.state_key, evt.rowid)
		this.invalidateStateCaches(evt.type, evt.state_key)
		this.stateSubs.notify(evt.type)
	}

	applyFullState(state: RawDBEvent[], omitMembers: boolean) {
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
		this.#emojiPacksCache.clear()
		this.#allPacksCache = null
		if (omitMembers) {
			newStateMap.set("m.room.member", this.state.get("m.room.member") ?? new Map())
		} else {
			this.#membersCache = null
		}
		this.state = newStateMap
		this.stateLoaded = true
		this.fullMembersLoaded = this.fullMembersLoaded || !omitMembers
		for (const [evtType, stateMap] of newStateMap) {
			if (omitMembers && evtType === "m.room.member") {
				continue
			}
			for (const [key] of stateMap) {
				this.stateSubs.notify(this.stateSubKey(evtType, key))
			}
			this.stateSubs.notify(evtType)
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

	applyTyping(users: string[]) {
		this.typing = users
		this.typingSub.notify()
	}

	doGarbageCollection() {
		const memberEventsToKeep = new Set<UserID>()
		const eventsToKeep = new Set<EventRowID>()
		if (this.meta.current.preview_event_rowid) {
			eventsToKeep.add(this.meta.current.preview_event_rowid)
			const previewEvt = this.eventsByRowID.get(this.meta.current.preview_event_rowid)
			if (previewEvt) {
				if (previewEvt.last_edit_rowid) {
					eventsToKeep.add(previewEvt.last_edit_rowid)
				}
				memberEventsToKeep.add(previewEvt?.sender ?? "")
			}
		}
		const newState = new Map<EventType, Map<string, EventRowID>>()
		let deletedState = 0
		newState.set("m.room.member", new Map<string, EventRowID>(
			this.state.get("m.room.member")?.entries().filter(([key, eventRowID]) => {
				if (memberEventsToKeep.has(key)) {
					eventsToKeep.add(eventRowID)
					return true
				} else {
					deletedState++
					return false
				}
			}) ?? [],
		))
		const emotes = this.state.get("im.ponies.room_emotes")
		if (emotes) {
			newState.set("im.ponies.room_emotes", emotes)
			for (const rowid of emotes.values()) {
				eventsToKeep.add(rowid)
			}
		}
		this.state = newState
		this.stateLoaded = false
		this.fullMembersLoaded = false
		this.membersRequested = false
		this.#membersCache = null
		this.paginationRequestedForRow = -1
		this.hasMoreHistory = true
		this.timeline = []
		this.notifyTimelineSubscribers()
		const eventsToKeepList = this.eventsByRowID.values()
			.filter(evt => eventsToKeep.has(evt.rowid))
			.toArray()
		const deletedEvents = this.eventsByRowID.size - eventsToKeep.size
		this.eventsByRowID.clear()
		this.eventsByID.clear()
		this.receiptsByEventID.clear()
		this.receiptsByUserID.clear()
		for (const evt of eventsToKeepList) {
			this.eventsByRowID.set(evt.rowid, evt)
			this.eventsByID.set(evt.event_id, evt)
		}
		return [deletedEvents, deletedState] as const
	}
}
