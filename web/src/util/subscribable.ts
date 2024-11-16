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

export type Subscriber = () => void
export type SubscribeFunc = (callback: Subscriber) => () => void

export default class Subscribable {
	readonly subscribers: Set<Subscriber> = new Set()

	subscribe: SubscribeFunc = callback => {
		this.subscribers.add(callback)
		return () => this.subscribers.delete(callback)
	}

	notify() {
		for (const sub of this.subscribers) {
			sub()
		}
	}
}

export class NoDataSubscribable extends Subscribable {
	data: number = 0

	notify = () => {
		this.data++
		super.notify()
	}

	getData = () => this.data
}

export class MultiSubscribable {
	readonly subscribers: Map<string, Set<Subscriber>> = new Map()
	readonly subscribeFuncs: Map<string, SubscribeFunc> = new Map()

	getSubscriber(key: string): SubscribeFunc {
		let subscribe = this.subscribeFuncs.get(key)
		if (!subscribe) {
			const subs = new Set<Subscriber>()
			subscribe = callback => {
				subs.add(callback)
				return () => {
					subs.delete(callback)
					// if (subs.size === 0 && Object.is(subs, this.subscribers.get(key))) {
					// 	this.subscribers.delete(key)
					// 	this.subscribeFuncs.delete(key)
					// }
				}
			}
			this.subscribers.set(key, subs)
			this.subscribeFuncs.set(key, subscribe)
		}
		return subscribe
	}

	notify(key: string) {
		const subs = this.subscribers.get(key)
		if (!subs) {
			return
		}
		for (const sub of subs) {
			sub()
		}
	}
}
