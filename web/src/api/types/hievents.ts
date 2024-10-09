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
	DBEvent,
	DBRoom,
	EventRowID,
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
	event: DBEvent
	error: string | null
}

export interface SendCompleteEvent extends RPCCommand<SendCompleteData> {
	command: "send_complete"
}

export interface EventsDecryptedData {
	room_id: RoomID
	preview_event_rowid?: EventRowID
	events: DBEvent[]
}

export interface EventsDecryptedEvent extends RPCCommand<EventsDecryptedData> {
	command: "events_decrypted"
}

export interface SyncRoom {
	meta: DBRoom
	timeline: TimelineRowTuple[]
	events: DBEvent[]
	state: Record<EventType, Record<string, EventRowID>>
	reset: boolean
}

export interface SyncCompleteData {
	rooms: Record<RoomID, SyncRoom>
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

export type RPCEvent =
	ClientStateEvent |
	TypingEvent |
	SendCompleteEvent |
	EventsDecryptedEvent |
	SyncCompleteEvent
