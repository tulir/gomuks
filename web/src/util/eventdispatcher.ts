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

export function useEventAsState<T>(dispatcher: NonNullCachedEventDispatcher<T>): T
export function useEventAsState<T>(dispatcher: CachedEventDispatcher<T>): T | null
export function useEventAsState<T>(dispatcher: CachedEventDispatcher<T>): T | null {
	return useSyncExternalStore(
		dispatcher.listenChange,
		() => dispatcher.current,
	)
}

export class EventDispatcher<T> {
	#listeners: ((data: T) => void)[] = []

	listenChange = (listener: () => void) => this.#listen(listener)

	listen(listener: (data: T) => void): () => void {
		return this.#listen(listener)
	}

	once(listener: (data: T) => void): () => void {
		let unsub: (() => void) | undefined = undefined
		const wrapped = (data: T) => {
			unsub?.()
			listener(data)
		}
		unsub = this.#listen(wrapped)
		return unsub
	}

	#listen(listener: (data: T) => void): () => void {
		this.#listeners.push(listener)
		return () => {
			const idx = this.#listeners.indexOf(listener)
			if (idx >= 0) {
				this.#listeners.splice(idx, 1)
			}
		}
	}

	emit(data: T) {
		for (const listener of this.#listeners) {
			listener(data)
		}
	}
}

export class CachedEventDispatcher<T> extends EventDispatcher<T> {
	current: T | null

	constructor(cache?: T | null) {
		super()
		this.current = cache ?? null
	}

	emit(data: T) {
		if (!Object.is(this.current, data)) {
			this.current = data
			super.emit(data)
		}
	}

	listen(listener: (data: T) => void): () => void {
		const unlisten = super.listen(listener)
		if (this.current !== null) {
			listener(this.current)
		}
		return unlisten
	}

	clearCache() {
		this.current = null
	}
}

export class NonNullCachedEventDispatcher<T> extends CachedEventDispatcher<T> {
	current: T

	constructor(cache: T) {
		super(cache)
		this.current = cache
	}

	clearCache() {
		throw new Error("Cannot clear cache of NonNullCachedEventDispatcher")
	}
}
