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
import {
	DBAccountData,
	DBRoom,
	DBRoomAccountData,
	EventRowID,
	RawDBEvent,
	TimelineRowTuple,
} from "./hitypes.ts"
import {
	DeviceID,
	EventType,
	RoomID,
	UserID,
} from "./mxtypes.ts"

interface BaseRPCCommand<T> {
	command: string
	request_id: number
	data: T
}

export interface TypingEventData {
	room_id: RoomID
	user_ids: UserID[]
}

export interface TypingEvent extends BaseRPCCommand<TypingEventData> {
	command: "typing"
}

export interface SendCompleteData {
	event: RawDBEvent
	error: string | null
}

export interface SendCompleteEvent extends BaseRPCCommand<SendCompleteData> {
	command: "send_complete"
}

export interface EventsDecryptedData {
	room_id: RoomID
	preview_event_rowid?: EventRowID
	events: RawDBEvent[]
}

export interface EventsDecryptedEvent extends BaseRPCCommand<EventsDecryptedData> {
	command: "events_decrypted"
}

export interface ImageAuthTokenEvent extends BaseRPCCommand<string> {
	command: "image_auth_token"
}

export interface SyncRoom {
	meta: DBRoom
	timeline: TimelineRowTuple[]
	events: RawDBEvent[]
	state: Record<EventType, Record<string, EventRowID>>
	reset: boolean
	notifications: SyncNotification[]
	account_data: Record<EventType, DBRoomAccountData>
}

export interface SyncNotification {
	event_rowid: EventRowID
	sound: boolean
}

export interface SyncCompleteData {
	rooms: Record<RoomID, SyncRoom>
	left_rooms: RoomID[]
	account_data: Record<EventType, DBAccountData>
	since?: string
	clear_state?: boolean
}

export interface SyncCompleteEvent extends BaseRPCCommand<SyncCompleteData> {
	command: "sync_complete"
}


export type ClientState = {
	is_logged_in: false
	is_verified: false
} | {
	is_logged_in: true
	is_verified: boolean
	user_id: UserID
	device_id: DeviceID
	homeserver_url: string
}

export interface ClientStateEvent extends BaseRPCCommand<ClientState> {
	command: "client_state"
}

export interface SyncStatus {
	type: "ok" | "waiting" | "errored"
	error?: string
	error_count: number
	last_sync?: number
}

export interface SyncStatusEvent extends BaseRPCCommand<SyncStatus> {
	command: "sync_status"
}

export interface InitCompleteEvent extends BaseRPCCommand<void> {
	command: "init_complete"
}

export interface RunData {
	run_id: string
	etag: string
}

export interface RunIDEvent extends BaseRPCCommand<RunData> {
	command: "run_id"
}

export interface ResponseCommand extends BaseRPCCommand<unknown> {
	command: "response"
}

export interface ErrorCommand extends BaseRPCCommand<unknown> {
	command: "error"
}

export type RPCEvent =
	ClientStateEvent |
	SyncStatusEvent |
	TypingEvent |
	SendCompleteEvent |
	EventsDecryptedEvent |
	SyncCompleteEvent |
	ImageAuthTokenEvent |
	InitCompleteEvent |
	RunIDEvent

export type RPCCommand = RPCEvent | ResponseCommand | ErrorCommand
