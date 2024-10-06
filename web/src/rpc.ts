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
import { RPCEvent } from "./hievents.ts"
import { EventDispatcher } from "./eventdispatcher.ts"

export class CancellablePromise<T> extends Promise<T> {
	constructor(
		executor: (resolve: (value: T) => void, reject: (reason?: Error) => void) => void,
		readonly cancel: (reason: string) => void,
	) {
		super(executor)
	}
}

export interface RPCClient {
	connect: EventDispatcher<ConnectionEvent>
	event: EventDispatcher<RPCEvent>
	start(): void
	stop(): void
	request<Req, Resp>(command: string, data: Req): CancellablePromise<Resp>
}

export interface ConnectionEvent {
	connected: boolean
	error: Error | null
}
