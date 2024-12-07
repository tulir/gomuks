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

function checkUpdate(etag: string) {
	if (!import.meta.env.PROD) {
		return
	} else if (!etag) {
		console.log("Not checking for update, frontend etag not found in websocket init")
		return
	}
	const currentETag = (
		document.querySelector("meta[name=gomuks-frontend-etag]") as HTMLMetaElement
	)?.content
	if (!currentETag) {
		console.log("Not checking for update, frontend etag not found in head")
	} else if (currentETag === etag) {
		console.log("Frontend is up to date")
	} else if (localStorage.lastUpdateTo === etag) {
		console.warn(
			`Frontend etag mismatch ${currentETag} !== ${etag}, `,
			"but localstorage says an update was already attempted",
		)
	} else {
		console.info(`Frontend etag mismatch ${currentETag} !== ${etag}, reloading`)
		localStorage.lastUpdateTo = etag
		location.search = "?" + new URLSearchParams({
			updateTo: etag,
			state: JSON.stringify(history.state),
		})
	}
}

export default class WSClient extends RPCClient {
	#conn: WebSocket | null = null
	#lastMessage: number = 0
	#pingInterval: number | null = null
	#lastReceivedEvt: number = 0
	#resumeRunID: string = ""
	#stopped = false
	#reconnectTimeout: number | null = null
	#connectFailures: number = 0

	constructor(readonly addr: string) {
		super()
	}

	start() {
		try {
			this.#stopped = false
			this.#lastMessage = Date.now()
			const params = new URLSearchParams({
				run_id: this.#resumeRunID,
				last_received_event: this.#lastReceivedEvt.toString(),
			}).toString()
			const addr = this.#lastReceivedEvt && this.#resumeRunID ? `${this.addr}?${params}` : this.addr
			console.info("Connecting to websocket", addr)
			this.#conn = new WebSocket(addr)
			this.#conn.onmessage = this.#onMessage
			this.#conn.onopen = this.#onOpen
			this.#conn.onerror = this.#onError
			this.#conn.onclose = this.#onClose
		} catch (err) {
			this.#dispatchConnectionStatus(false, false, `Failed to create websocket: ${err}`)
		}
	}

	#pingLoop = () => {
		if (Date.now() - this.#lastMessage > RECV_TIMEOUT) {
			console.warn("Websocket ping timeout, last message at", this.#lastMessage)
			this.#conn?.close(4002, "Ping timeout")
			return
		}
		this.send(JSON.stringify({
			command: "ping",
			data: {
				last_received_id: this.#lastReceivedEvt,
			},
			request_id: this.nextRequestID,
		}))
	}

	stop() {
		this.#stopped = true
		if (this.#pingInterval !== null) {
			clearInterval(this.#pingInterval)
			this.#pingInterval = null
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
		let parsed: RPCCommand
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
		if (parsed.request_id < 0) {
			this.#lastReceivedEvt = parsed.request_id
		} else if (parsed.command === "run_id") {
			console.log("Received run ID", parsed.data)
			this.#resumeRunID = parsed.data.run_id
			checkUpdate(parsed.data.etag)
		}
		this.onCommand(parsed)
	}

	#dispatchConnectionStatus(connected: boolean, reconnecting: boolean, error: string | null, nextAttempt?: number) {
		this.connect.emit({
			connected,
			reconnecting,
			error,
			nextAttempt: nextAttempt ? new Date(nextAttempt).toLocaleTimeString() : undefined,
		})
	}

	#onOpen = () => {
		console.info("Websocket opened")
		this.#dispatchConnectionStatus(true, false, null)
		this.#connectFailures = 0
		this.#pingInterval = setInterval(this.#pingLoop, PING_INTERVAL)
	}

	#clearPending = () => {
		for (const { reject } of this.pendingRequests.values()) {
			reject(new Error("Websocket closed"))
		}
		this.pendingRequests.clear()
	}

	#onError = (ev: Event) => {
		console.error("Websocket error:", ev)
	}

	#onClose = (ev: CloseEvent) => {
		this.#connectFailures++
		console.warn("Websocket closed:", ev)
		this.#clearPending()
		if (this.#pingInterval !== null) {
			clearInterval(this.#pingInterval)
			this.#pingInterval = null
		}
		const willReconnect = !this.#stopped && !this.#reconnectTimeout
		const backoff = Math.min(2 ** (this.#connectFailures - 4), 10) * 1000
		this.#dispatchConnectionStatus(
			false,
			willReconnect,
			`Websocket closed: ${ev.code} ${ev.reason}`,
			Date.now() + backoff,
		)
		if (willReconnect) {
			console.log("Attempting to reconnect in", backoff, "ms")
			this.#reconnectTimeout = setTimeout(() => {
				console.log("Reconnecting now")
				this.#reconnectTimeout = null
				this.start()
			}, backoff)
		} else {
			console.log(`Not reconnecting (stopped=${this.#stopped}, reconnectTimeout=${this.#reconnectTimeout})`)
		}
	}
}
