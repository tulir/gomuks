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
import { useEffect, useState, useSyncExternalStore } from "react"

export function useEventAsState<T>(dispatcher?: EventDispatcher<T>): T | null {
	const [state, setState] = useState<T | null>(null)
	useEffect(() => dispatcher && dispatcher.listen(setState), [dispatcher])
	return state
}

export function useCachedEventAsState<T>(dispatcher: CachedEventDispatcher<T>): T | null {
	return useSyncExternalStore(
		dispatcher.listenChange,
		() => dispatcher.current,
	)
}

export function useNonNullEventAsState<T>(dispatcher: NonNullCachedEventDispatcher<T>): T {
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
}

export class NonNullCachedEventDispatcher<T> extends CachedEventDispatcher<T> {
	current: T

	constructor(cache: T) {
		super(cache)
		this.current = cache
	}
}
