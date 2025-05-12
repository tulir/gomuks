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
	DeviceID,
	EncryptedEventContent,
	EncryptionEventContent,
	EventID,
	EventType,
	LazyLoadSummary,
	ReceiptType,
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
	dm_user_id?: UserID
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
	marked_unread: boolean

	prev_batch: string
}

export interface DBSpaceEdge {
	// space_id: RoomID
	child_id: RoomID

	child_event_rowid?: EventRowID
	order?: string
	suggested?: true

	parent_event_rowid?: EventRowID
	canonical?: true
}

//eslint-disable-next-line @typescript-eslint/no-explicit-any
export type UnknownEventContent = Record<string, any>

export interface StrippedStateEvent {
	type: EventType
	sender: UserID
	state_key: string
	content: UnknownEventContent
}

export interface DBInvitedRoom {
	room_id: RoomID
	created_at: number
	invite_state: StrippedStateEvent[]
}

export enum UnreadType {
	None = 0b0000,
	Normal = 0b0001,
	Notify = 0b0010,
	Highlight = 0b0100,
	Sound = 0b1000,
}

export interface LocalContent {
	sanitized_html?: TrustedHTML
	edit_source?: string
	html_version?: number
	was_plaintext?: boolean
	big_emoji?: boolean
	has_math?: boolean
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
	orig_local_content?: LocalContent
	last_edit?: MemDBEvent
	viewing_redacted?: boolean
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

export interface DBReceipt {
	user_id: UserID
	receipt_type: ReceiptType
	thread_id?: EventID | "main"
	event_id: EventID
	timestamp: number
}

export interface MemReceipt extends DBReceipt {
	event_rowid: EventRowID
	timeline_rowid: TimelineRowID
}

export interface PaginationResponse {
	events: RawDBEvent[]
	receipts: Record<EventID, DBReceipt[]>
	related_events: RawDBEvent[]
	has_more: boolean
}

export interface ResolveAliasResponse {
	room_id: RoomID
	servers: string[]
}

export interface LoginFlowsResponse {
	flows: {
		type: string
	}[]
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

export interface PasswordLoginRequest {
	type: "m.login.password"
	identifier: {
		type: "m.id.user"
		user: string
	}
	password: string
}

export interface SSOLoginRequest {
	type: "m.login.token"
	token: string
}

export interface JWTLoginRequest {
	type: "org.matrix.login.jwt"
	token: string
}

export type LoginRequest = PasswordLoginRequest | SSOLoginRequest | JWTLoginRequest

export type TrustState = "blacklisted" | "unverified" | "verified"
	| "cross-signed-untrusted" | "cross-signed-tofu" | "cross-signed-verified"
	| "unknown-device" | "forwarded" | "invalid"

export interface ProfileDevice {
	device_id: DeviceID
	name: string
	identity_key: string
	signing_key: string
	fingerprint: string
	trust_state: TrustState
}

export interface ProfileEncryptionInfo {
	devices_tracked: boolean
	devices: ProfileDevice[]
	master_key: string
	first_master_key: string
	user_trusted: boolean
	errors: string[]
}

export interface DBPushRegistration {
	device_id: string
	type: "fcm"
	data: unknown
	encryption: { key: string }
	expiration?: number
}

export interface MediaEncodingOptions {
	encode_to?: string
	quality?: number
	resize_width?: number
	resize_height?: number
	resize_percent?: number
}

export type MembershipAction = "invite" | "kick" | "ban" | "unban"

export interface KeyRestoreProgress  {
	current_room_id: RoomID
	stage: "fetching" | "decrypting" | "saving" | "postprocessing" | "done"
	decrypted: number
	decryption_failed: number
	import_failed: number
	saved: number
	post_processed: number
	total: number
}
