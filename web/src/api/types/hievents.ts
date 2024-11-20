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

export interface RPCCommand<T> {
	command: string
	request_id: number
	data: T
}

export interface TypingEventData {
	room_id: RoomID
	user_ids: UserID[]
}

export interface TypingEvent extends RPCCommand<TypingEventData> {
	command: "typing"
}

export interface SendCompleteData {
	event: RawDBEvent
	error: string | null
}

export interface SendCompleteEvent extends RPCCommand<SendCompleteData> {
	command: "send_complete"
}

export interface EventsDecryptedData {
	room_id: RoomID
	preview_event_rowid?: EventRowID
	events: RawDBEvent[]
}

export interface EventsDecryptedEvent extends RPCCommand<EventsDecryptedData> {
	command: "events_decrypted"
}

export interface ImageAuthTokenEvent extends RPCCommand<string> {
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
}

export interface SyncCompleteEvent extends RPCCommand<SyncCompleteData> {
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

export interface ClientStateEvent extends RPCCommand<ClientState> {
	command: "client_state"
}

export interface SyncStatus {
	type: "ok" | "waiting" | "errored"
	error?: string
	error_count: number
	last_sync?: number
}

export interface SyncStatusEvent extends RPCCommand<SyncStatus> {
	command: "sync_status"
}

export type RPCEvent =
	ClientStateEvent |
	SyncStatusEvent |
	TypingEvent |
	SendCompleteEvent |
	EventsDecryptedEvent |
	SyncCompleteEvent |
	ImageAuthTokenEvent
