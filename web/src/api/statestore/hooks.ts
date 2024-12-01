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
import { useEffect, useMemo, useState, useSyncExternalStore } from "react"
import { CustomEmojiPack } from "@/util/emoji"
import type { EventID, EventType, MemDBEvent, UnknownEventContent } from "../types"
import { Preferences, preferences } from "../types/preferences"
import { StateStore } from "./main.ts"
import { RoomStateStore } from "./room.ts"

export function useRoomTimeline(room: RoomStateStore): (MemDBEvent | null)[] {
	return useSyncExternalStore(
		room.timelineSub.subscribe,
		() => room.timelineCache,
	)
}

export function useRoomState(
	room?: RoomStateStore, type?: EventType, stateKey: string | undefined = "",
): MemDBEvent | null {
	const isNoop = !room || !type || stateKey === undefined
	return useSyncExternalStore(
		isNoop ? noopSubscribe : room.stateSubs.getSubscriber(room.stateSubKey(type, stateKey)),
		isNoop ? returnNull : (() => room.getStateEvent(type, stateKey) ?? null),
	)
}

export function useRoomMembers(room?: RoomStateStore): MemDBEvent[] {
	return useSyncExternalStore(
		room ? room.stateSubs.getSubscriber("m.room.member") : noopSubscribe,
		room ? room.getMembers : () => [],
	)
}

const noopSubscribe = () => () => {}
const returnNull = () => null

export function useRoomEvent(room: RoomStateStore, eventID: EventID | null): MemDBEvent | null {
	return useSyncExternalStore(
		eventID ? room.eventSubs.getSubscriber(eventID) : noopSubscribe,
		eventID ? (() => room.eventsByID.get(eventID) ?? null) : returnNull,
	)
}

export function useAccountData(ss: StateStore, type: EventType): UnknownEventContent | null {
	return useSyncExternalStore(
		ss.accountDataSubs.getSubscriber(type),
		() => ss.accountData.get(type) ?? null,
	)
}

export function useRoomAccountData(room: RoomStateStore | null, type: EventType): UnknownEventContent | null {
	return useSyncExternalStore(
		room ? room.accountDataSubs.getSubscriber(type) : noopSubscribe,
		() => room?.accountData.get(type) ?? null,
	)
}

export function usePreferences(ss: StateStore, room: RoomStateStore | null) {
	useSyncExternalStore(ss.preferenceSub.subscribe, ss.preferenceSub.getData)
	useSyncExternalStore(room?.preferenceSub.subscribe ?? noopSubscribe, room?.preferenceSub.getData ?? returnNull)
}

export function usePreference<T extends keyof Preferences>(
	ss: StateStore, room: RoomStateStore | null, key: T,
): typeof preferences[T]["defaultValue"] {
	const [val, setVal] = useState(
		(room ? room.preferences[key] : ss.preferences[key]) ?? preferences[key].defaultValue,
	)
	useEffect(() => {
		const checkChanges = () => {
			setVal((room ? room.preferences[key] : ss.preferences[key]) ?? preferences[key].defaultValue)
		}
		const unsubMain = ss.preferenceSub.subscribe(checkChanges)
		const unsubRoom = room?.preferenceSub.subscribe(checkChanges)
		return () => {
			unsubMain()
			unsubRoom?.()
		}
	}, [ss, room, key])
	return val
}

export function useCustomEmojis(
	ss: StateStore, room: RoomStateStore,
): CustomEmojiPack[] {
	const personalPack = useSyncExternalStore(
		ss.accountDataSubs.getSubscriber("im.ponies.user_emotes"),
		() => ss.getPersonalEmojiPack(),
	)
	const watchedRoomPacks = useSyncExternalStore(
		ss.emojiRoomsSub.subscribe,
		() => ss.getRoomEmojiPacks(),
	)
	const specialRoomPacks = useSyncExternalStore<Record<string, CustomEmojiPack>>(
		room.stateSubs.getSubscriber("im.ponies.room_emotes"),
		() => room.preferences.show_room_emoji_packs ? room.getAllEmojiPacks() : {},
	)
	return useMemo(() => {
		const allPacksObject = { ...watchedRoomPacks, ...specialRoomPacks }
		if (personalPack) {
			allPacksObject.personal = personalPack
		}
		return Object.values(allPacksObject)
	}, [personalPack, watchedRoomPacks, specialRoomPacks])
}
