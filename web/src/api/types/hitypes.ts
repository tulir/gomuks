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
	ContentURI,
	CreateEventContent,
	EncryptedEventContent,
	EncryptionEventContent,
	EventID,
	EventType,
	LazyLoadSummary,
	RelationType,
	RoomAlias,
	RoomID,
	TombstoneEventContent,
	UserID,
} from "./mxtypes.ts"

export type EventRowID = number
export type TimelineRowID = number

export interface TimelineRowTuple {
	timeline_rowid: TimelineRowID
	event_rowid: EventRowID
}

export enum RoomNameQuality {
	Nil = 0,
	Participants,
	CanonicalAlias,
	Explicit,
}

export interface DBRoom {
	room_id: RoomID
	creation_content?: CreateEventContent
	tombstone?: TombstoneEventContent

	name?: string
	name_quality: RoomNameQuality
	avatar?: ContentURI
	explicit_avatar: boolean
	topic?: string
	canonical_alias?: RoomAlias
	lazy_load_summary?: LazyLoadSummary

	encryption_event?: EncryptionEventContent
	has_member_list: boolean

	preview_event_rowid: EventRowID
	sorting_timestamp: number
	unread_highlights: number
	unread_notifications: number
	unread_messages: number

	prev_batch: string
}

//eslint-disable-next-line @typescript-eslint/no-explicit-any
export type UnknownEventContent = Record<string, any>

export enum UnreadType {
	None = 0b0000,
	Normal = 0b0001,
	Notify = 0b0010,
	Highlight = 0b0100,
	Sound = 0b1000,
}

export interface LocalContent {
	sanitized_html?: TrustedHTML
	html_version?: number
	was_plaintext?: boolean
	big_emoji?: boolean
}

export interface BaseDBEvent {
	rowid: EventRowID
	timeline_rowid: TimelineRowID

	room_id: RoomID
	event_id: EventID
	sender: UserID
	type: EventType
	state_key?: string
	timestamp: number

	content: UnknownEventContent
	unsigned: EventUnsigned
	local_content?: LocalContent

	transaction_id?: string

	redacted_by?: EventID
	relates_to?: EventID
	relation_type?: RelationType

	decryption_error?: string
	send_error?: string

	reactions?: Record<string, number>
	last_edit_rowid?: EventRowID
	unread_type: UnreadType
}

export interface RawDBEvent extends BaseDBEvent {
	decrypted?: UnknownEventContent
	decrypted_type?: EventType
}

export interface MemDBEvent extends BaseDBEvent {
	mem: true
	pending: boolean
	encrypted?: EncryptedEventContent
	orig_content?: UnknownEventContent
	last_edit?: MemDBEvent
}

export interface DBAccountData {
	user_id: UserID
	type: EventType
	content: UnknownEventContent
}

export interface DBRoomAccountData {
	user_id: UserID
	room_id: RoomID
	type: EventType
	content: UnknownEventContent
}

export interface PaginationResponse {
	events: RawDBEvent[]
	has_more: boolean
}

export interface ResolveAliasResponse {
	room_id: RoomID
	servers: string[]
}

export interface EventUnsigned {
	prev_content?: unknown
	prev_sender?: UserID
}

export interface ClientWellKnown {
	"m.homeserver": {
		base_url: string
	},
	"m.identity_server": {
		base_url: string
	}
}

export function roomStateGUIDToString(guid: RoomStateGUID): string {
	return `${encodeURIComponent(guid.room_id)}/${guid.type}/${encodeURIComponent(guid.state_key)}`
}

export function stringToRoomStateGUID(str?: string | null): RoomStateGUID | undefined {
	if (!str) {
		return
	}
	const [roomID, type, stateKey] = str.split("/")
	if (!roomID || !type || !stateKey) {
		return
	}
	return {
		room_id: decodeURIComponent(roomID) as RoomID,
		type: type as EventType,
		state_key: decodeURIComponent(stateKey),
	}
}

export interface RoomStateGUID {
	room_id: RoomID
	type: EventType
	state_key: string
}
