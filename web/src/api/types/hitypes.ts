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
	creation_content: CreateEventContent

	name?: string
	name_quality: RoomNameQuality
	avatar?: ContentURI
	topic?: string
	canonical_alias?: RoomAlias
	lazy_load_summary?: LazyLoadSummary

	encryption_event?: EncryptionEventContent
	has_member_list: boolean

	preview_event_rowid: EventRowID
	sorting_timestamp: number

	prev_batch: string
}

export interface DBEvent {
	rowid: EventRowID
	timeline_rowid: TimelineRowID

	room_id: RoomID
	event_id: EventID
	sender: UserID
	type: EventType
	state_key?: string
	timestamp: number

	content: unknown
	decrypted?: unknown
	decrypted_type?: EventType
	encrypted?: EncryptedEventContent
	unsigned: EventUnsigned

	transaction_id?: string

	redacted_by?: EventID
	relates_to?: EventID
	relation_type?: RelationType

	decryption_error?: string

	reactions?: Record<string, number>
	last_edit_rowid?: EventRowID
}

export interface DBAccountData {
	user_id: UserID
	room_id?: RoomID
	type: EventType
	content: unknown
}

export interface PaginationResponse {
	events: DBEvent[]
	has_more: boolean
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
