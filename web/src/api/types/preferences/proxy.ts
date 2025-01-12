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
import type { RoomStateStore, StateStore } from "@/api/statestore"
import { Preferences, isValidPreferenceKey, preferences } from "./preferences.ts"
import { PreferenceContext, PreferenceValueType } from "./types.ts"

const prefKeys = Object.keys(preferences)

export function getPreferenceProxy(store: StateStore, room?: RoomStateStore): Required<Preferences> {
	return new Proxy({}, {
		set(): boolean {
			throw new Error("The preference proxy is read-only")
		},
		get(_target: never, key: keyof Preferences | symbol): PreferenceValueType | undefined {
			if (typeof key !== "string") {
				return
			}
			const pref = preferences[key]
			if (!pref) {
				return
			}
			let val: typeof pref.defaultValue | undefined
			for (const ctx of pref.allowedContexts) {
				if (ctx === PreferenceContext.Account) {
					val = store.serverPreferenceCache?.[key]
				} else if (ctx === PreferenceContext.Device) {
					val = store.localPreferenceCache?.[key]
				} else if (ctx === PreferenceContext.RoomAccount && room) {
					val = room.serverPreferenceCache?.[key]
				} else if (ctx === PreferenceContext.RoomDevice && room) {
					val = room.localPreferenceCache?.[key]
				} else if (ctx === PreferenceContext.Config) {
					// TODO
				}
				if (val !== undefined) {
					return val
				}
			}
			return pref.defaultValue
		},
		ownKeys(): string[] {
			return prefKeys
		},
		getOwnPropertyDescriptor(_target: never, key: string | symbol): PropertyDescriptor | undefined {
			return isValidPreferenceKey(key) ? {
				configurable: true,
				enumerable: true,
				writable: false,
			} : undefined
		},
	}) as Required<Preferences>
}
