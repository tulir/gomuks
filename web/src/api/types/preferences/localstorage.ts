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
import { Preferences, isValidPreferenceKey, preferences } from "./preferences.ts"
import { PreferenceContext } from "./types.ts"

function getObjectFromLocalStorage(key: string): Preferences {
	const localStorageVal = localStorage.getItem(key)
	if (localStorageVal) {
		try {
			return JSON.parse(localStorageVal)
		} catch {
			return {}
		}
	}
	return {}
}

const globalPrefKeys = Object.entries(preferences)
	.filter(([,pref]) => pref.allowedContexts.includes(PreferenceContext.Device))
	.map(([key]) => key)
const roomPrefKeys = Object.entries(preferences)
	.filter(([,pref]) => pref.allowedContexts.includes(PreferenceContext.RoomDevice))
	.map(([key]) => key)

export function getLocalStoragePreferences(localStorageKey: string, onChange: () => void): Preferences {
	return new Proxy(getObjectFromLocalStorage(localStorageKey), {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		set(target: Preferences, key: string | symbol, newValue: any): boolean {
			if (!isValidPreferenceKey(key)) {
				return false
			}
			target[key] = newValue
			localStorage.setItem(localStorageKey, JSON.stringify(target))
			onChange()
			return true
		},
		deleteProperty(target: Preferences, key: string | symbol): boolean {
			if (!isValidPreferenceKey(key)) {
				return false
			}
			delete target[key]
			if (Object.keys(target).length === 0) {
				localStorage.removeItem(localStorageKey)
			} else {
				localStorage.setItem(localStorageKey, JSON.stringify(target))
			}
			onChange()
			return true
		},
		ownKeys(): string[] {
			return localStorageKey === "global_prefs" ? globalPrefKeys : roomPrefKeys
		},
		getOwnPropertyDescriptor(_target: never, key: string | symbol): PropertyDescriptor | undefined {
			const keySet = localStorageKey === "global_prefs" ? globalPrefKeys : roomPrefKeys
			return (typeof key === "string" && keySet.includes(key)) ? {
				configurable: true,
				enumerable: true,
				writable: true,
			} : undefined
		},
	})
}
