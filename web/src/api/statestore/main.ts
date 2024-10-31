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
import { getAvatarURL } from "@/api/media.ts"
import { CustomEmojiPack, parseCustomEmojiPack } from "@/util/emoji"
import { NonNullCachedEventDispatcher } from "@/util/eventdispatcher.ts"
import { focused } from "@/util/focus.ts"
import toSearchableString from "@/util/searchablestring.ts"
import Subscribable, { MultiSubscribable } from "@/util/subscribable.ts"
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
	UnknownEventContent,
	UserID,
	roomStateGUIDToString,
} from "../types"
import { RoomStateStore } from "./room.ts"

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
}

export class StateStore {
	readonly rooms: Map<RoomID, RoomStateStore> = new Map()
	readonly roomList = new NonNullCachedEventDispatcher<RoomListEntry[]>([])
	currentRoomListFilter: string = ""
	readonly accountData: Map<string, UnknownEventContent> = new Map()
	readonly accountDataSubs = new MultiSubscribable()
	readonly emojiRoomsSub = new Subscribable()
	#frequentlyUsedEmoji: Map<string, number> | null = null
	#emojiPackKeys: RoomStateGUID[] | null = null
	#watchedRoomEmojiPacks: Record<string, CustomEmojiPack> | null = null
	#personalEmojiPack: CustomEmojiPack | null = null
	switchRoom?: (roomID: RoomID | null) => void
	activeRoomID?: RoomID
	imageAuthToken?: string

	getFilteredRoomList(): RoomListEntry[] {
		if (!this.currentRoomListFilter) {
			return this.roomList.current
		}
		return this.roomList.current.filter(entry => entry.search_name.includes(this.currentRoomListFilter))
	}

	#shouldHideRoom(entry: SyncRoom): boolean {
		const cc = entry.meta.creation_content
		if ((cc?.type ?? "") !== "") {
			// The room is not a normal room
			return true
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
			entry.meta.preview_event_rowid !== oldEntry.meta.current.preview_event_rowid ||
			entry.events.findIndex(evt => evt.rowid === entry.meta.preview_event_rowid) !== -1
	}

	#makeRoomListEntry(entry: SyncRoom, room?: RoomStateStore): RoomListEntry | null {
		if (this.#shouldHideRoom(entry)) {
			return null
		}
		if (!room) {
			room = this.rooms.get(entry.meta.room_id)
		}
		const preview_event = room?.eventsByRowID.get(entry.meta.preview_event_rowid)
		const preview_sender = preview_event && room?.getStateEvent("m.room.member", preview_event.sender)
		const name = entry.meta.name ?? "Unnamed room"
		return {
			room_id: entry.meta.room_id,
			dm_user_id: entry.meta.lazy_load_summary?.heroes?.length === 1
				? entry.meta.lazy_load_summary.heroes[0] : undefined,
			sorting_timestamp: entry.meta.sorting_timestamp,
			preview_event,
			preview_sender,
			name,
			search_name: toSearchableString(name),
			avatar: entry.meta.avatar,
			unread_messages: entry.meta.unread_messages,
			unread_notifications: entry.meta.unread_notifications,
			unread_highlights: entry.meta.unread_highlights,
		}
	}

	applySync(sync: SyncCompleteData) {
		const resyncRoomList = this.roomList.current.length === 0
		const changedRoomListEntries = new Map<RoomID, RoomListEntry | null>()
		for (const [roomID, data] of Object.entries(sync.rooms)) {
			let isNewRoom = false
			let room = this.rooms.get(roomID)
			if (!room) {
				room = new RoomStateStore(data.meta, this)
				this.rooms.set(roomID, room)
				isNewRoom = true
			}
			const roomListEntryChanged = !resyncRoomList && (isNewRoom || this.#roomListEntryChanged(data, room))
			room.applySync(data)
			if (roomListEntryChanged) {
				changedRoomListEntries.set(roomID, this.#makeRoomListEntry(data, room))
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

			if (Notification.permission === "granted" && !focused.current) {
				for (const notification of data.notifications) {
					this.showNotification(room, notification.event_rowid, notification.sound)
				}
			}
		}
		for (const ad of Object.values(sync.account_data)) {
			if (ad.type === "io.element.recent_emoji") {
				this.#frequentlyUsedEmoji = null
			}
			this.accountData.set(ad.type, ad.content)
			this.accountDataSubs.notify(ad.type)
		}
		for (const roomID of sync.left_rooms) {
			if (this.activeRoomID === roomID) {
				this.switchRoom?.(null)
			}
			this.rooms.delete(roomID)
			changedRoomListEntries.set(roomID, null)
		}

		let updatedRoomList: RoomListEntry[] | undefined
		if (resyncRoomList) {
			updatedRoomList = Object.values(sync.rooms)
				.map(entry => this.#makeRoomListEntry(entry))
				.filter(entry => entry !== null)
			updatedRoomList.sort((r1, r2) => r1.sorting_timestamp - r2.sorting_timestamp)
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
		const icon = `${getAvatarURL(evt.sender, memberEvt?.content)}&image_auth=${this.imageAuthToken}`
		const roomName = room.meta.current.name ?? "Unnamed room"
		const senderName = memberEvt?.content.displayname ?? evt.sender
		const title = senderName === roomName ? senderName : `${senderName} (${roomName})`
		const notif = new Notification(title, {
			body,
			icon,
			badge: "/gomuks.png",
			// timestamp: evt.timestamp,
			// image: ...,
			tag: rowid.toString(),
		})
		room.openNotifications.set(rowid, notif)
		notif.onclose = () => room.openNotifications.delete(rowid)
		notif.onclick = () => this.onClickNotification(room.roomID)
		if (sound) {
			// TODO play sound
		}
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
}
