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
import { getAvatarThumbnailURL } from "@/api/media.ts"
import { Preferences, getLocalStoragePreferences, getPreferenceProxy } from "@/api/types/preferences"
import { CustomEmojiPack, parseCustomEmojiPack } from "@/util/emoji"
import { NonNullCachedEventDispatcher } from "@/util/eventdispatcher.ts"
import { focused } from "@/util/focus.ts"
import toSearchableString from "@/util/searchablestring.ts"
import Subscribable, { MultiSubscribable, NoDataSubscribable } from "@/util/subscribable.ts"
import { getDisplayname } from "@/util/validation.ts"
import {
	ContentURI,
	EventRowID,
	EventsDecryptedData,
	ImagePack,
	ImagePackRooms,
	MemDBEvent,
	RoomID,
	RoomStateGUID,
	SendCompleteData,
	SyncCompleteData,
	SyncRoom,
	SyncToDevice,
	TypingEventData,
	UnknownEventContent,
	UserID,
	roomStateGUIDToString,
} from "../types"
import { InvitedRoomStore } from "./invitedroom.ts"
import { RoomStateStore } from "./room.ts"
import { DirectChatSpace, RoomListFilter, Space, SpaceEdgeStore, SpaceOrphansSpace, UnreadsSpace } from "./space.ts"

export interface RoomListEntry {
	room_id: RoomID
	dm_user_id?: UserID
	sorting_timestamp: number
	preview_event?: MemDBEvent
	preview_sender?: MemDBEvent
	name: string
	search_name: string
	avatar?: ContentURI
	unread_messages: number
	unread_notifications: number
	unread_highlights: number
	marked_unread: boolean
	is_invite?: boolean
}

export interface GCSettings {
	interval: number,
	lastOpenedCutoff: number,
}

export interface WidgetListener {
	onTimelineEvent(evt: MemDBEvent): void
	onStateEvent(evt: MemDBEvent): void
	onToDeviceEvent(evt: SyncToDevice): void
	onRoomChange(roomID: RoomID | null): void
}

window.gcSettings ??= {
	// Run garbage collection every 15 minutes.
	interval: 15 * 60 * 1000,
	// Run garbage collection to rooms not opened in the past 30 minutes.
	lastOpenedCutoff: 30 * 60 * 1000,
}

export class StateStore {
	userID: UserID = ""
	readonly rooms: Map<RoomID, RoomStateStore> = new Map()
	readonly inviteRooms: Map<RoomID, InvitedRoomStore> = new Map()
	readonly roomList = new NonNullCachedEventDispatcher<RoomListEntry[]>([])
	readonly roomListEntries = new Map<RoomID, RoomListEntry>()
	readonly topLevelSpaces = new NonNullCachedEventDispatcher<RoomID[]>([])
	readonly spaceEdges: Map<RoomID, SpaceEdgeStore> = new Map()
	readonly spaceOrphans = new SpaceOrphansSpace(this)
	readonly directChatsSpace = new DirectChatSpace()
	readonly unreadsSpace = new UnreadsSpace(this)
	readonly pseudoSpaces = [
		this.spaceOrphans,
		this.directChatsSpace,
		this.unreadsSpace,
	] as const
	currentRoomListQuery: string = ""
	currentRoomListFilter: RoomListFilter | null = null
	readonly accountData: Map<string, UnknownEventContent> = new Map()
	readonly accountDataSubs = new MultiSubscribable()
	readonly emojiRoomsSub = new Subscribable()
	readonly preferences = getPreferenceProxy(this)
	#frequentlyUsedEmoji: Map<string, number> | null = null
	#emojiPackKeys: RoomStateGUID[] | null = null
	#watchedRoomEmojiPacks: Record<string, CustomEmojiPack> | null = null
	#personalEmojiPack: CustomEmojiPack | null = null
	readonly preferenceSub = new NoDataSubscribable()
	readonly localPreferenceCache: Preferences = getLocalStoragePreferences("global_prefs", this.preferenceSub.notify)
	serverPreferenceCache: Preferences = {}
	switchRoom?: (roomID: RoomID | null) => void
	#activeRoomID: RoomID | null = null
	activeRoomIsPreview: boolean = false
	imageAuthToken?: string
	readonly widgetListeners: Set<WidgetListener> = new Set()

	get activeRoomID(): RoomID | null {
		return this.#activeRoomID
	}

	set activeRoomID(roomID: RoomID | null) {
		this.#activeRoomID = roomID
		this.widgetListeners.forEach(listener => listener.onRoomChange(roomID))
	}

	#roomListFilterFunc = (entry: RoomListEntry) => {
		if (this.currentRoomListQuery && !entry.search_name.includes(this.currentRoomListQuery)) {
			return false
		} else if (this.currentRoomListFilter && !this.currentRoomListFilter.include(entry)) {
			return false
		}
		return true
	}

	getSpaceByID(spaceID: string | undefined): RoomListFilter | null {
		if (!spaceID) {
			return null
		}
		const realSpace = this.spaceEdges.get(spaceID)
		if (realSpace) {
			return realSpace
		}
		for (const pseudoSpace of this.pseudoSpaces) {
			if (pseudoSpace.id === spaceID) {
				return pseudoSpace
			}
		}
		console.warn("Failed to find space", spaceID)
		return null
	}

	findMatchingSpace(room: RoomListEntry): Space | null {
		if (this.spaceOrphans.include(room)) {
			return this.spaceOrphans
		}
		for (const spaceID of this.topLevelSpaces.current) {
			const space = this.spaceEdges.get(spaceID)
			if (space?.include(room)) {
				return space
			}
		}
		if (this.directChatsSpace.include(room)) {
			return this.directChatsSpace
		}
		return null
	}

	get roomListFilterFunc(): ((entry: RoomListEntry) => boolean) | null {
		if (!this.currentRoomListFilter && !this.currentRoomListQuery) {
			return null
		}
		return this.#roomListFilterFunc
	}

	getFilteredRoomList(): RoomListEntry[] {
		const fn = this.roomListFilterFunc
		return fn ? this.roomList.current.filter(fn) : this.roomList.current
	}

	#shouldHideRoom(entry: SyncRoom): boolean {
		const cc = entry.meta.creation_content
		switch (cc?.type ?? "") {
		default:
			// The room is not a normal room
			return true
		case "":
		case "support.feline.policy.lists.msc.v1":
		case "org.matrix.msc3417.call":
		}
		const replacementRoom = entry.meta.tombstone?.replacement_room
		if (
			replacementRoom
			&& this.rooms.get(replacementRoom)?.meta.current.creation_content?.predecessor?.room_id
			=== entry.meta.room_id
		) {
			// The room is tombstoned and the replacement room is valid.
			return true
		}
		// Otherwise don't hide the room.
		return false
	}

	#roomListEntryChanged(entry: SyncRoom, oldEntry: RoomStateStore): boolean {
		return entry.meta.sorting_timestamp !== oldEntry.meta.current.sorting_timestamp ||
			entry.meta.unread_messages !== oldEntry.meta.current.unread_messages ||
			entry.meta.unread_notifications !== oldEntry.meta.current.unread_notifications ||
			entry.meta.unread_highlights !== oldEntry.meta.current.unread_highlights ||
			entry.meta.marked_unread !== oldEntry.meta.current.marked_unread ||
			entry.meta.preview_event_rowid !== oldEntry.meta.current.preview_event_rowid ||
			entry.meta.name !== oldEntry.meta.current.name ||
			entry.meta.avatar !== oldEntry.meta.current.avatar ||
			(entry.events ?? []).findIndex(evt => evt.rowid === entry.meta.preview_event_rowid) !== -1
	}

	#makeRoomListEntry(entry: SyncRoom, room?: RoomStateStore): RoomListEntry | null {
		if (!room) {
			room = this.rooms.get(entry.meta.room_id)
		}
		if (this.#shouldHideRoom(entry)) {
			if (room) {
				room.hidden = true
			}
			return null
		}
		if (room?.hidden) {
			room.hidden = false
		}
		const preview_event = room?.eventsByRowID.get(entry.meta.preview_event_rowid)
		const preview_sender = preview_event && room?.getStateEvent("m.room.member", preview_event.sender)
		const name = entry.meta.name ?? "Unnamed room"
		return {
			room_id: entry.meta.room_id,
			dm_user_id: entry.meta.dm_user_id,
			sorting_timestamp: entry.meta.sorting_timestamp,
			preview_event,
			preview_sender,
			name,
			search_name: toSearchableString(name),
			avatar: entry.meta.avatar,
			unread_messages: entry.meta.unread_messages,
			unread_notifications: entry.meta.unread_notifications,
			unread_highlights: entry.meta.unread_highlights,
			marked_unread: entry.meta.marked_unread,
		}
	}

	#applyUnreadModification(meta: RoomListEntry | null, oldMeta: RoomListEntry | undefined | null) {
		const someMeta = meta ?? oldMeta
		if (!someMeta) {
			return
		}
		if (this.spaceOrphans.include(someMeta)) {
			this.spaceOrphans.applyUnreads(meta, oldMeta)
			return
		}
		if (this.directChatsSpace.include(someMeta)) {
			this.directChatsSpace.applyUnreads(meta, oldMeta)
		}
		for (const space of this.spaceEdges.values()) {
			if (space.include(someMeta)) {
				space.applyUnreads(meta, oldMeta)
			}
		}
	}

	applySync(sync: SyncCompleteData) {
		let prevActiveRoom: RoomID | null = null
		if (sync.clear_state && this.rooms.size > 0) {
			console.info("Clearing state store as sync told to reset and there are rooms in the store")
			prevActiveRoom = this.activeRoomID
			this.clear()
		}
		const resyncRoomList = this.roomList.current.length === 0
		const changedRoomListEntries = new Map<RoomID, RoomListEntry | null>()
		if (sync.to_device?.length && this.widgetListeners.size > 0) {
			for (const listener of this.widgetListeners) {
				sync.to_device.forEach(listener.onToDeviceEvent)
			}
		}
		for (const data of sync.invited_rooms ?? []) {
			const room = new InvitedRoomStore(data, this)
			this.inviteRooms.set(room.room_id, room)
			if (!resyncRoomList) {
				changedRoomListEntries.set(room.room_id, room)
				this.#applyUnreadModification(room, this.roomListEntries.get(room.room_id))
				this.roomListEntries.set(room.room_id, room)
			}
			if (this.activeRoomID === room.room_id) {
				this.switchRoom?.(room.room_id)
			}
		}
		const hasInvites = this.inviteRooms.size > 0
		for (const [roomID, data] of Object.entries(sync.rooms ?? {})) {
			let isNewRoom = false
			let room = this.rooms.get(roomID)
			if (!room) {
				room = new RoomStateStore(data.meta, this)
				this.rooms.set(roomID, room)
				if (hasInvites) {
					this.inviteRooms.delete(roomID)
				}
				isNewRoom = true
			}
			const roomListEntryChanged = !resyncRoomList && (isNewRoom || this.#roomListEntryChanged(data, room))
			room.applySync(data)
			if (roomListEntryChanged) {
				const entry = this.#makeRoomListEntry(data, room)
				changedRoomListEntries.set(roomID, entry)
				this.#applyUnreadModification(entry, this.roomListEntries.get(roomID))
				if (entry) {
					this.roomListEntries.set(roomID, entry)
				} else {
					this.roomListEntries.delete(roomID)
				}
			}
			if (!resyncRoomList) {
				// When we join a valid replacement room, hide the tombstoned room.
				const predecessorID = data.meta.creation_content?.predecessor?.room_id
				if (
					isNewRoom
					&& typeof predecessorID === "string"
					&& this.rooms.get(predecessorID)?.meta.current.tombstone?.replacement_room === roomID) {
					changedRoomListEntries.set(predecessorID, null)
				}
			}

			if (window.Notification?.permission === "granted" && !focused.current && data.notifications) {
				for (const notification of data.notifications) {
					this.showNotification(room, notification.event_rowid, notification.sound)
				}
			}
			if (this.activeRoomID === roomID && this.activeRoomIsPreview) {
				this.switchRoom?.(roomID)
			}
		}
		for (const ad of Object.values(sync.account_data ?? {})) {
			if (ad.type === "io.element.recent_emoji") {
				this.#frequentlyUsedEmoji = null
			} else if (ad.type === "fi.mau.gomuks.preferences") {
				this.serverPreferenceCache = ad.content
				this.preferenceSub.notify()
			}
			this.accountData.set(ad.type, ad.content)
			this.accountDataSubs.notify(ad.type)
		}
		for (const roomID of sync.left_rooms ?? []) {
			if (this.activeRoomID === roomID) {
				this.switchRoom?.(null)
			}
			this.rooms.delete(roomID)
			changedRoomListEntries.set(roomID, null)
			this.#applyUnreadModification(null, this.roomListEntries.get(roomID))
		}

		let updatedRoomList: RoomListEntry[] | undefined
		if (resyncRoomList) {
			updatedRoomList = this.inviteRooms.values().toArray()
			updatedRoomList = updatedRoomList.concat(Object.values(sync.rooms ?? {})
				.map(entry => this.#makeRoomListEntry(entry))
				.filter(entry => entry !== null))
			updatedRoomList.sort((r1, r2) => r1.sorting_timestamp - r2.sorting_timestamp)
			for (const entry of updatedRoomList) {
				this.#applyUnreadModification(entry, undefined)
				this.roomListEntries.set(entry.room_id, entry)
			}
		} else if (changedRoomListEntries.size > 0) {
			updatedRoomList = this.roomList.current.filter(entry => !changedRoomListEntries.has(entry.room_id))
			for (const entry of changedRoomListEntries.values()) {
				if (!entry) {
					continue
				}
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
		if (sync.space_edges) {
			// Ensure all space stores exist first
			for (const spaceID of Object.keys(sync.space_edges)) {
				this.getSpaceStore(spaceID, true)
			}
			for (const [spaceID, children] of Object.entries(sync.space_edges ?? {})) {
				this.getSpaceStore(spaceID, true).children = children
			}
		}
		if (sync.top_level_spaces) {
			this.topLevelSpaces.emit(sync.top_level_spaces)
			this.spaceOrphans.children = sync.top_level_spaces.map(child_id => ({ child_id }))
		}
		if (prevActiveRoom) {
			// TODO this will fail if the room is not in the top 100 recent rooms
			this.switchRoom?.(prevActiveRoom)
		}
	}

	invalidateEmojiPackKeyCache() {
		this.#emojiPackKeys = null
		this.#watchedRoomEmojiPacks = null
	}

	invalidateEmojiPacksCache() {
		this.#watchedRoomEmojiPacks = null
		this.emojiRoomsSub.notify()
	}

	getPersonalEmojiPack(): CustomEmojiPack | null {
		if (this.#personalEmojiPack === null) {
			const pack = this.accountData.get("im.ponies.user_emotes")
			if (!pack || !pack.images) {
				return null
			}
			this.#personalEmojiPack = parseCustomEmojiPack(pack as ImagePack, "personal", "Personal pack")
		}
		return this.#personalEmojiPack
	}

	getEmojiPackKeys(): RoomStateGUID[] {
		if (this.#emojiPackKeys === null) {
			const emoteRooms = this.accountData.get("im.ponies.emote_rooms") as ImagePackRooms | undefined
			try {
				const emojiPacks: RoomStateGUID[] = []
				for (const [roomID, packs] of Object.entries(emoteRooms?.rooms ?? {})) {
					for (const pack of Object.keys(packs)) {
						emojiPacks.push({ room_id: roomID, type: "im.ponies.room_emotes", state_key: pack })
					}
				}
				this.#emojiPackKeys = emojiPacks
			} catch (err) {
				console.warn("Failed to parse emote rooms data", err, emoteRooms)
				this.#emojiPackKeys = []
			}
		}
		return this.#emojiPackKeys
	}

	getRoomEmojiPacks() {
		if (this.#watchedRoomEmojiPacks === null) {
			this.#watchedRoomEmojiPacks = Object.fromEntries(
				this.getEmojiPackKeys()
					.map(key => {
						const room = this.rooms.get(key.room_id)
						if (!room) {
							console.warn("Failed to find room for emoji pack", key)
							return null
						}
						const pack = room.getEmojiPack(key.state_key)
						if (!pack) {
							console.warn("Failed to find pack", key)
							return null
						}
						return [roomStateGUIDToString(key), pack]
					})
					.filter(pack => !!pack),
			)
		}
		return this.#watchedRoomEmojiPacks ?? {}
	}

	getSpaceStore(spaceID: RoomID, force: true): SpaceEdgeStore
	getSpaceStore(spaceID: RoomID): SpaceEdgeStore | null
	getSpaceStore(spaceID: RoomID, force?: true): SpaceEdgeStore | null {
		let store = this.spaceEdges.get(spaceID)
		if (!store) {
			if (!force && this.rooms.get(spaceID)?.meta.current.creation_content?.type !== "m.space") {
				return null
			}
			store = new SpaceEdgeStore(spaceID, this)
			this.spaceEdges.set(spaceID, store)
		}
		return store
	}

	get frequentlyUsedEmoji(): Map<string, number> {
		if (this.#frequentlyUsedEmoji === null) {
			const emojiData = this.accountData.get("io.element.recent_emoji")
			try {
				const recentList = emojiData?.recent_emoji as [string, number][] | undefined
				this.#frequentlyUsedEmoji = new Map(recentList?.toSorted(
					([, count1], [, count2]) => count2 - count1,
				))
			} catch (err) {
				console.warn("Failed to parse recent emoji data", err, emojiData?.recent_emoji)
				this.#frequentlyUsedEmoji = new Map()
			}
		}
		return this.#frequentlyUsedEmoji
	}

	showNotification(room: RoomStateStore, rowid: EventRowID, sound: boolean) {
		const evt = room.eventsByRowID.get(rowid)
		if (!evt || typeof evt.content.body !== "string") {
			return
		}
		let body = evt.content.body
		if (body.length > 400) {
			body = body.slice(0, 350) + " […]"
		}
		const memberEvt = room.getStateEvent("m.room.member", evt.sender)
		const icon = `${getAvatarThumbnailURL(evt.sender, memberEvt?.content)}&image_auth=${this.imageAuthToken}`
		const roomName = room.meta.current.name ?? "Unnamed room"
		const senderName = getDisplayname(evt.sender, memberEvt?.content)
		const title = senderName === roomName ? senderName : `${senderName} (${roomName})`
		if (sound) {
			(document.getElementById("default-notification-sound") as HTMLAudioElement)?.play()
		}
		const notif = new Notification(title, {
			body,
			icon,
			badge: "gomuks.png",
			// timestamp: evt.timestamp,
			// image: ...,
			silent: !sound,
			tag: rowid.toString(),
		})
		room.openNotifications.set(rowid, notif)
		notif.onclose = () => room.openNotifications.delete(rowid)
		notif.onclick = () => this.onClickNotification(room.roomID)
	}

	onClickNotification(roomID: RoomID) {
		if (this.switchRoom) {
			this.switchRoom(roomID)
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

	applyTyping(typing: TypingEventData) {
		const room = this.rooms.get(typing.room_id)
		if (!room) {
			// TODO log or something?
			return
		}
		room.applyTyping(typing.user_ids)
	}

	doGarbageCollection() {
		const maxLastOpened = Date.now() - window.gcSettings.lastOpenedCutoff
		let deletedEvents = 0
		let deletedState = 0
		for (const room of this.rooms.values()) {
			if (room.roomID === this.activeRoomID || room.lastOpened > maxLastOpened) {
				continue
			}
			const [de, ds] = room.doGarbageCollection()
			deletedEvents += de
			deletedState += ds
		}
		return { deletedEvents, deletedState } as const
	}

	clear() {
		this.rooms.clear()
		this.inviteRooms.clear()
		this.spaceEdges.clear()
		this.pseudoSpaces.forEach(space => space.clearUnreads())
		this.roomList.emit([])
		this.topLevelSpaces.emit([])
		this.accountData.clear()
		this.currentRoomListQuery = ""
		this.currentRoomListFilter = null
		this.#frequentlyUsedEmoji = null
		this.#emojiPackKeys = null
		this.#watchedRoomEmojiPacks = null
		this.#personalEmojiPack = null
		this.serverPreferenceCache = {}
		this.activeRoomID = null
	}
}
