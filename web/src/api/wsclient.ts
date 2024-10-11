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
import RPCClient from "./rpc.ts"
import type { RPCCommand } from "./types"

const PING_INTERVAL = 15_000
const RECV_TIMEOUT = 4 * PING_INTERVAL

export default class WSClient extends RPCClient {
	#conn: WebSocket | null = null
	#lastMessage: number = 0
	#pingInterval: number | null = null

	constructor(readonly addr: string) {
		super()
	}

	start() {
		try {
			this.#lastMessage = Date.now()
			console.info("Connecting to websocket", this.addr)
			this.#conn = new WebSocket(this.addr)
			this.#conn.onmessage = this.#onMessage
			this.#conn.onopen = this.#onOpen
			this.#conn.onerror = this.#onError
			this.#conn.onclose = this.#onClose
			this.#pingInterval = setInterval(this.#pingLoop, PING_INTERVAL)
		} catch (err) {
			this.#dispatchConnectionStatus(false, err as Error)
		}
	}

	#pingLoop = () => {
		if (Date.now() - this.#lastMessage > RECV_TIMEOUT) {
			console.warn("Websocket ping timeout, last message at", this.#lastMessage)
			this.#conn?.close(4002, "Ping timeout")
			return
		}
		this.send(JSON.stringify({ command: "ping", request_id: this.nextRequestID }))
	}

	stop() {
		if (this.#pingInterval !== null) {
			clearInterval(this.#pingInterval)
		}
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
		this.#lastMessage = Date.now()
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
