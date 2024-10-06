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
import { RPCCommand, RPCEvent } from "./types/hievents.ts"
import { CachedEventDispatcher, EventDispatcher } from "../util/eventdispatcher.ts"
import { ConnectionEvent, RPCClient } from "./rpc.ts"
import { CancellablePromise } from "../util/promise.ts"

export class ErrorResponse extends Error {
	constructor(public data: unknown) {
		super(`${data}`)
	}
}

export default class WSClient implements RPCClient {
	#conn: WebSocket | null = null
	readonly connect: CachedEventDispatcher<ConnectionEvent> = new CachedEventDispatcher()
	readonly event: EventDispatcher<RPCEvent> = new EventDispatcher()
	readonly #pendingRequests: Map<number, {
		resolve: (data: unknown) => void,
		reject: (err: Error) => void
	}> = new Map()
	#nextRequestID: number = 1

	constructor(readonly addr: string) {

	}

	start() {
		try {
			console.info("Connecting to websocket", this.addr)
			this.#conn = new WebSocket(this.addr)
			this.#conn.onmessage = this.#onMessage
			this.#conn.onopen = this.#onOpen
			this.#conn.onerror = this.#onError
			this.#conn.onclose = this.#onClose
		} catch (err) {
			this.#dispatchConnectionStatus(false, err as Error)
		}
	}

	stop() {
		this.#conn?.close(1000, "Client closed")
	}

	#cancelRequest(request_id: number, reason: string) {
		if (!this.#pendingRequests.has(request_id)) {
			console.debug("Tried to cancel unknown request", request_id)
			return
		}
		this.request("cancel", { request_id, reason }).then(
			() => console.debug("Cancelled request", request_id, "for", reason),
			err => console.debug("Failed to cancel request", request_id, "for", reason, err),
		)
	}

	request<Req, Resp>(command: string, data: Req): CancellablePromise<Resp> {
		if (!this.#conn) {
			return new CancellablePromise((_resolve, reject) => {
				reject(new Error("Websocket not connected"))
			}, () => {
			})
		}
		const request_id = this.#nextRequestID++
		return new CancellablePromise((resolve, reject) => {
			if (!this.#conn) {
				reject(new Error("Websocket not connected"))
				return
			}
			this.#pendingRequests.set(request_id, { resolve: resolve as ((value: unknown) => void), reject })
			this.#conn.send(JSON.stringify({
				command,
				request_id,
				data,
			}))
		}, this.#cancelRequest.bind(this, request_id))
	}

	#onMessage = (ev: MessageEvent) => {
		let parsed: RPCCommand<unknown>
		try {
			parsed = JSON.parse(ev.data)
			if (!parsed.command) {
				throw new Error("Missing 'command' field in JSON message")
			}
		} catch (err) {
			console.error("Malformed JSON in websocket:", err)
			console.error("Message:", ev.data)
			this.#conn?.close(1003, "Malformed JSON")
			return
		}
		if (parsed.command === "response" || parsed.command === "error") {
			const target = this.#pendingRequests.get(parsed.request_id)
			if (!target) {
				console.error("Received response for unknown request:", parsed)
				return
			}
			this.#pendingRequests.delete(parsed.request_id)
			if (parsed.command === "response") {
				target.resolve(parsed.data)
			} else {
				target.reject(new ErrorResponse(parsed.data))
			}
		} else {
			this.event.emit(parsed as RPCEvent)
		}
	}

	#dispatchConnectionStatus(connected: boolean, error: Error | null) {
		this.connect.emit({ connected, error })
	}

	#onOpen = () => {
		console.info("Websocket opened")
		this.#dispatchConnectionStatus(true, null)
	}

	#clearPending = () => {
		for (const { reject } of this.#pendingRequests.values()) {
			reject(new Error("Websocket closed"))
		}
		this.#pendingRequests.clear()
	}

	#onError = (ev: Event) => {
		console.error("Websocket error:", ev)
		this.#dispatchConnectionStatus(false, new Error("Websocket error"))
		this.#clearPending()
	}

	#onClose = (ev: CloseEvent) => {
		console.warn("Websocket closed:", ev)
		this.#dispatchConnectionStatus(false, new Error(`Websocket closed: ${ev.code} ${ev.reason}`))
		this.#clearPending()
	}
}
