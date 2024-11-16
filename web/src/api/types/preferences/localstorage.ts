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
import { Preferences, existingPreferenceKeys, preferences } from "./preferences.ts"
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

export function getLocalStoragePreferences(localStorageKey: string, onChange: () => void): Preferences {
	return new Proxy(getObjectFromLocalStorage(localStorageKey), {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		set(target: Preferences, key: keyof Preferences, newValue: any): boolean {
			if (!existingPreferenceKeys.has(key)) {
				return false
			}
			target[key] = newValue
			localStorage.setItem(localStorageKey, JSON.stringify(target))
			onChange()
			return true
		},
		deleteProperty(target: Preferences, key: keyof Preferences): boolean {
			if (!existingPreferenceKeys.has(key)) {
				return false
			}
			delete target[key]
			if (Object.keys(target).length === 0) {
				localStorage.removeItem(localStorageKey)
			}
			onChange()
			return true
		},
		ownKeys(): string[] {
			console.warn("localStorage preference proxy ownKeys called")
			// This is only for debugging, so the performance doesn't matter that much
			return Object.entries(preferences)
				.filter(([,pref]) =>
					pref.allowedContexts.includes(localStorageKey === "global_prefs"
						? PreferenceContext.Device : PreferenceContext.RoomDevice))
				.map(([key]) => key)
		},
	})
}
