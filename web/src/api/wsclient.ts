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
import type { RPCCommand } from "./types"
import RPCClient from "./rpc.ts"

export default class WSClient extends RPCClient {
	#conn: WebSocket | null = null

	constructor(readonly addr: string) {
		super()
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

	get isConnected() {
		return this.#conn?.readyState === WebSocket.OPEN
	}

	send(data: string) {
		if (!this.#conn) {
			throw new Error("Websocket not connected")
		}
		this.#conn.send(data)
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
		this.onCommand(parsed)
	}

	#dispatchConnectionStatus(connected: boolean, error: Error | null) {
		this.connect.emit({ connected, error })
	}

	#onOpen = () => {
		console.info("Websocket opened")
		this.#dispatchConnectionStatus(true, null)
	}

	#clearPending = () => {
		for (const { reject } of this.pendingRequests.values()) {
			reject(new Error("Websocket closed"))
		}
		this.pendingRequests.clear()
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
