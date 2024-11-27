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
import type * as Wails from "@wailsio/runtime"
import { CancellablePromise } from "@/util/promise.ts"
import RPCClient, { ErrorResponse } from "./rpc.ts"

declare global {
	interface Window {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		wails: any
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		_wails: any
	}
}

// Wails uses Go naming conventions, so:
/* eslint-disable new-cap */

export default class WailsClient extends RPCClient {
	protected isConnected = true
	#wails?: typeof Wails

	async start() {
		this.#wails = await import("@wailsio/runtime")
		this.#wails.Events.On("hicli_event", (evt: Wails.Events.WailsEvent) => {
			this.event.emit(evt.data[0])
		})
		this.#wails.Call.ByName("main.CommandHandler.Init")
		this.connect.emit({ connected: true, error: null })
	}

	async stop() {}

	protected send() {
		throw new Error("Raw sends are not supported")
	}

	request<Req, Resp>(command: string, data: Req): CancellablePromise<Resp> {
		return new CancellablePromise((resolve, reject) => {
			if (!this.#wails) {
				reject(new Error("Wails not initialized"))
				return
			}
			this.#wails.Call.ByName("main.CommandHandler.HandleCommand", { command, data })
				.then((res: { command?: string, data: Resp }) => {
					if (typeof res !== "object" || !res) {
						reject(new Error("Unexpected response data from Wails"))
					} else if (res.command === "response") {
						resolve(res.data)
					} else if (res.command === "error") {
						reject(new ErrorResponse(res.data))
					} else {
						reject(new Error("Unexpected response data from Wails"))
					}
				}, reject)
		}, () => {})
	}
}
