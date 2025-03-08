// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import {
	IGetMediaConfigResult,
	IOpenIDCredentials,
	IOpenIDUpdate,
	IRoomAccountData,
	IRoomEvent,
	ISendDelayedEventDetails,
	ISendEventDetails,
	ITurnServer,
	OpenIDRequestState,
	SimpleObservable,
	Symbols,
	UpdateDelayedEventAction,
	WidgetDriver,
} from "matrix-widget-api"
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import { EventRowID, RoomID } from "@/api/types"
import { filterEvent, isRecord, iterRoomTimeline, memDBEventToIRoomEvent, notNull } from "./util"

class GomuksWidgetDriver extends WidgetDriver {
	private openIDToken: IOpenIDCredentials | null = null
	private openIDExpiry: number | null = null

	constructor(
		private client: Client,
		private room: RoomStateStore,
		private openPermissionPrompt: (requested: Set<string>) => Promise<Set<string>>,
	) {
		super()
	}

	async validateCapabilities(requested: Set<string>): Promise<Set<string>> {
		return this.openPermissionPrompt(requested)
	}

	async sendEvent(
		eventType: string,
		content: unknown,
		stateKey: string | null = null,
		roomID: string | null = null,
	): Promise<ISendEventDetails> {
		if (!isRecord(content)) {
			throw new Error("Content must be an object")
		}
		roomID = roomID ?? this.room.roomID
		if (stateKey) {
			const eventID = await this.client.rpc.setState(roomID, eventType, stateKey, content)
			return { eventId: eventID, roomId: roomID }
		} else {
			const rawDBEvt = await this.client.rpc.sendEvent(roomID, eventType, content, false, true)
			return { eventId: rawDBEvt.event_id, roomId: rawDBEvt.room_id }
		}
	}

	async sendDelayedEvent(
		delay: number | null,
		parentDelayID: string | null,
		eventType: string,
		content: unknown,
		stateKey: string | null = null,
		roomID: string | null = null,
	): Promise<ISendDelayedEventDetails> {
		if (!isRecord(content)) {
			throw new Error("Content must be an object")
		} else if (stateKey === null) {
			throw new Error("Non-state delayed events are not supported")
		} else if (parentDelayID !== null) {
			throw new Error("Parent delayed events are not supported")
		} else if (!delay) {
			throw new Error("Delay must be a number")
		}
		roomID = roomID ?? this.room.roomID
		const delayID = await this.client.rpc.setState(roomID, eventType, stateKey, content, { delay_ms: delay })
		return { delayId: delayID, roomId: roomID }
	}

	async updateDelayedEvent(delayID: string, action: UpdateDelayedEventAction): Promise<void> {
		await this.client.rpc.updateDelayedEvent(delayID, action)
	}

	async sendToDevice(
		eventType: string,
		encrypted: boolean,
		content: { [userId: string]: { [deviceId: string]: object } },
	): Promise<void> {
		await this.client.rpc.sendToDevice(eventType, content, encrypted)
	}

	private readRoomData<T>(
		roomIDs: RoomID[] | null,
		reader: (room: RoomStateStore) => T | null,
	): T[] {
		if (roomIDs === null || (roomIDs.length === 1 && roomIDs[0] === this.room.roomID)) {
			const val = reader(this.room)
			return val ? [val] : []
		} else if (roomIDs.includes(Symbols.AnyRoom)) {
			return Array.from(this.client.store.rooms.values().map(reader).filter(notNull))
		} else {
			return roomIDs.map(roomID => {
				const room = this.client.store.rooms.get(roomID)
				if (!room) {
					return null
				}
				return reader(room)
			}).filter(notNull)
		}
	}

	async readRoomTimeline(
		roomID: string,
		eventType: string,
		msgtype: string | undefined,
		stateKey: string | undefined,
		limit: number,
		since: string | undefined,
	): Promise<IRoomEvent[]> {
		const room = this.client.store.rooms.get(roomID)
		if (!room) {
			return []
		}
		if (room.timeline.length === 0) {
			await this.client.loadMoreHistory(roomID)
		}
		return iterRoomTimeline(room, since)
			.filter(filterEvent(eventType, msgtype, stateKey))
			.take(limit)
			.map(memDBEventToIRoomEvent)
			.toArray()
	}

	async readRoomState(roomID: string, eventType: string, stateKey?: string): Promise<IRoomEvent[]> {
		const room = this.client.store.rooms.get(roomID)
		if (!room) {
			return []
		}
		if (
			stateKey === undefined
			&& eventType === "m.room.member"
			&& !room.fullMembersLoaded
			&& !room.membersRequested
		) {
			room.membersRequested = true
			this.client.loadRoomState(room.roomID, { omitMembers: false, refetch: false })
		}
		const stateEvts = room.state.get(eventType)
		if (!stateEvts) {
			return []
		}
		let stateRowIDs: EventRowID[] = []
		if (stateKey !== undefined) {
			const stateEvtID = stateEvts.get(stateKey)
			if (!stateEvtID) {
				return []
			}
			stateRowIDs = [stateEvtID]
		} else {
			stateRowIDs = Array.from(stateEvts.values())
		}
		return stateRowIDs.map(rowID => {
			const evt = room.eventsByRowID.get(rowID)
			if (!evt) {
				return null
			}
			return memDBEventToIRoomEvent(evt)
		}).filter(notNull)
	}

	async readStateEvents(
		eventType: string,
		stateKey: string | undefined,
		limit: number,
		roomIDs: RoomID[] | null = null,
	): Promise<IRoomEvent[]> {
		console.warn(`Deprecated call to readStateEvents(${eventType}, ${stateKey}, ${limit}, ${roomIDs})`)
		return (await Promise.all(
			this.readRoomData(roomIDs, room => this.readRoomState(room.roomID, eventType, stateKey)),
		)).flatMap(evts => evts)
	}

	async readRoomAccountData(type: string, roomIDs: string[] | null = null): Promise<IRoomAccountData[]> {
		return this.readRoomData(roomIDs, room => {
			const content = room.accountData.get(type)
			if (!content) {
				return null
			}
			return {
				type,
				room_id: room.roomID,
				content,
			}
		})
	}

	async askOpenID(observer: SimpleObservable<IOpenIDUpdate>): Promise<void> {
		if (!this.openIDToken || (this.openIDExpiry ?? 0) < Date.now()) {
			const openID = await this.client.rpc.requestOpenIDToken()
			if (!openID) {
				return
			}
			this.openIDToken = openID
			this.openIDExpiry = Date.now() + (openID.expires_in / 2) * 1000
		}
		observer.update({
			state: OpenIDRequestState.Allowed,
			token: this.openIDToken,
		})
	}

	async uploadFile(file: XMLHttpRequestBodyInit): Promise<{ contentUri: string }> {
		const res = await fetch("_gomuks/upload?encrypt=false", {
			method: "POST",
			body: file,
		})
		const json = await res.json()
		if (!res.ok) {
			throw new Error(json.error)
		}
		return { contentUri: json.url }
	}

	async downloadFile(url: string): Promise<{ file: XMLHttpRequestBodyInit }> {
		const res = await fetch(url)
		if (!res.ok) {
			throw new Error(res.statusText)
		}
		return { file: await res.blob() }
	}

	async getMediaConfig(): Promise<IGetMediaConfigResult> {
		return await this.client.rpc.getMediaConfig()
	}

	getKnownRooms(): string[] {
		return Array.from(this.client.store.rooms.keys())
	}

	async navigate(uri: string): Promise<void> {
		if (uri.startsWith("https://matrix.to/")) {
			const parsedURL = new URL(uri)
			const parts = parsedURL.hash.split("/")
			if (parts[1][0] === "#") {
				uri = `matrix:r/${parts[1].slice(1)}`
			} else if (parts[1][0] === "!") {
				if (parts.length >= 4 && parts[3][0] === "$") {
					uri = `matrix:roomid/${parts[1].slice(1)}/e/${parts[4].slice(1)}`
				} else {
					uri = `matrix:roomid/${parts[1].slice(1)}`
				}
			} else if (parts[1][0] === "@") {
				uri = `matrix:u/${parts[1].slice(1)}`
			}
		}
		if (uri.startsWith("matrix:")) {
			window.location.hash = `#/uri/${encodeURIComponent(uri)}`
		} else {
			throw new Error("Unsupported URI: " + uri)
		}
	}

	async * getTurnServers(): AsyncGenerator<ITurnServer> {
		const res = await this.client.rpc.getTurnServers()
		yield res
	}

	// TODO: searchUserDirectory, readEventRelations
}

export default GomuksWidgetDriver
