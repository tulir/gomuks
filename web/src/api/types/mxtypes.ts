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
export type RoomID = string
export type EventID = string
export type UserID = string
export type DeviceID = string
export type EventType = string
export type ContentURI = string
export type RoomAlias = string
export type ReceiptType = "m.read" | "m.read.private"
export type RoomVersion = "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" | "10" | "11"
export type RoomType = "" | "m.space" | "support.feline.policy.lists.msc.v1" | "org.matrix.msc3417.call"
export type RelationType = "m.annotation" | "m.reference" | "m.replace" | "m.thread"

export type JSONValue =
	| string
	| number
	| boolean
	| null
	| JSONValue[]
	| {[key: string]: JSONValue}

export interface RoomPredecessor {
	room_id: RoomID
	event_id: EventID
}

export interface CreateEventContent {
	type: RoomType
	"m.federate": boolean
	room_version: RoomVersion
	predecessor: RoomPredecessor
}

export interface TombstoneEventContent {
	body: string
	replacement_room: RoomID
}

export interface LazyLoadSummary {
	"m.heroes"?: UserID[]
	"m.joined_member_count"?: number
	"m.invited_member_count"?: number
}

export interface EncryptionEventContent {
	algorithm: string
	rotation_period_ms?: number
	rotation_period_msgs?: number
}

export interface EncryptedEventContent {
	algorithm: "m.megolm.v1.aes-sha2"
	ciphertext: string
	session_id: string
	sender_key?: string
	device_id?: DeviceID
}

export interface UserProfile {
	displayname?: string
	avatar_url?: ContentURI
	avatar_file?: EncryptedFile
	[custom: string]: unknown
}

export interface PronounSet {
	subject?: string
	object?: string
	possessive_determiner?: string
	possessive_pronoun?: string
	reflexive?: string
	summary: string
	language: string
}

export type Membership = "join" | "leave" | "ban" | "invite" | "knock"

export interface MemberEventContent extends UserProfile {
	membership: Membership
	reason?: string
}

export interface RoomAvatarEventContent {
	url?: ContentURI
}

export interface RoomNameEventContent {
	name?: string
}

export interface RoomCanonicalAliasEventContent {
	alias?: RoomAlias | null
	alt_aliases?: RoomAlias[]
}

export interface RoomTopicEventContent {
	topic?: string
}

export interface ACLEventContent {
	allow?: string[]
	allow_ip_literals?: boolean
	deny?: string[]
}

export interface PolicyRuleContent {
	entity: string
	reason: string
	recommendation: string
	"org.matrix.msc4205.hashes"?: {
		sha256: string
	}
}

export interface PowerLevelEventContent {
	users?: Record<UserID, number>
	users_default?: number
	events?: Record<EventType, number>
	events_default?: number
	state_default?: number
	notifications?: {
		room?: number
	}
	ban?: number
	redact?: number
	invite?: number
	kick?: number
}

export interface PinnedEventsContent {
	pinned?: EventID[]
}

export interface Mentions {
	user_ids: UserID[]
	room: boolean
}

export interface RelatesTo {
	rel_type?: RelationType
	event_id?: EventID
	key?: string

	is_falling_back?: boolean
	"m.in_reply_to"?: {
		event_id?: EventID
	}
}

export enum ContentWarningType {
	Spoiler = "town.robin.msc3725.spoiler",
	NSFW = "town.robin.msc3725.nsfw",
	Graphic = "town.robin.msc3725.graphic",
	Medical = "town.robin.msc3725.medical",
}

export interface ContentWarning {
	type: ContentWarningType
	description?: string
}

export interface URLPreview {
	matched_url: string
	"beeper:image:encryption"?: EncryptedFile
	"matrix:image:size": number
	"og:image"?: ContentURI
	"og:url": string
	"og:image:width"?: number
	"og:image:height"?: number
	"og:image:type"?: string
	"og:title"?: string
	"og:description"?: string
}

export interface BeeperPerMessageProfile extends UserProfile {
	id: string
}

export interface BaseMessageEventContent {
	msgtype: string
	body: string
	formatted_body?: string
	format?: "org.matrix.custom.html"
	"m.mentions"?: Mentions
	"m.relates_to"?: RelatesTo
	"town.robin.msc3725.content_warning"?: ContentWarning
	"page.codeberg.everypizza.msc4193.spoiler"?: boolean
	"page.codeberg.everypizza.msc4193.spoiler.reason"?: string
	"m.url_previews"?: URLPreview[]
	"com.beeper.linkpreviews"?: URLPreview[]
	"com.beeper.per_message_profile"?: BeeperPerMessageProfile
}

export interface TextMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.text" | "m.notice" | "m.emote"
}

export interface MediaMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.sticker" | "m.image" | "m.file" | "m.audio" | "m.video"
	filename?: string
	url?: ContentURI
	file?: EncryptedFile
	info?: MediaInfo
}

export interface ReactionEventContent {
	"m.relates_to": {
		rel_type: "m.annotation"
		event_id: EventID
		key: string
	}
	"com.beeper.reaction.shortcode"?: string
}

export interface IgnoredUsersEventContent {
	ignored_users: Record<string, unknown>
}

export interface EncryptedFile {
	url: ContentURI
	k: string
	v: "v2"
	ext: true
	alg: "A256CTR"
	key_ops: string[]
	kty: "oct"
}

export interface MediaInfo {
	mimetype?: string
	size?: number
	w?: number
	h?: number
	duration?: number
	thumbnail_url?: ContentURI
	thumbnail_file?: EncryptedFile
	thumbnail_info?: MediaInfo

	"fi.mau.hide_controls"?: boolean
	"fi.mau.loop"?: boolean
	"fi.mau.autoplay"?: boolean
	"fi.mau.no_audio"?: boolean
	"xyz.amorgan.blurhash"?: string
}

export interface LocationMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.location"
	geo_uri: string
	"org.matrix.msc3488.asset"?: {
		type?: "m.pin"
	}
	"org.matrix.msc3488.location"?: {
		uri: string
		description?: string
	}
}

export type MessageEventContent = TextMessageEventContent | MediaMessageEventContent | LocationMessageEventContent

export type ImagePackUsage = "emoticon" | "sticker"

export interface ImagePackEntry {
	url: ContentURI
	body?: string
	info?: MediaInfo
	usage?: ImagePackUsage[]
}

export interface ImagePack {
	images: Record<string, ImagePackEntry>
	pack: {
		display_name?: string
		avatar_url?: ContentURI
		usage?: ImagePackUsage[]
	}
}

export interface ImagePackRooms {
	rooms: Record<RoomID, Record<string, Record<string, never>>>
}

export interface ElementRecentEmoji {
	recent_emoji: [string, number][]
}

export type JoinRule = "public" | "knock" | "restricted" | "knock_restricted" | "invite" | "private"

export interface RoomSummary {
	room_id: RoomID
	membership?: Membership

	room_version?: RoomVersion
	"im.nheko.summary.room_version"?: RoomVersion
	"im.nheko.summary.version"?: RoomVersion
	encryption?: "m.megolm.v1.aes-sha2"
	"im.nheko.summary.encryption"?: "m.megolm.v1.aes-sha2"

	avatar_url?: ContentURI
	canonical_alias?: RoomAlias
	guest_can_join: boolean
	join_rule?: JoinRule
	name?: string
	num_joined_members: number
	room_type: RoomType
	topic?: string
	world_readable: boolean
	allowed_room_ids?: RoomID[]
}

export interface RespRoomJoin {
	room_id: RoomID
}

export interface RespOpenIDToken {
	access_token: string
	expires_in: number
	matrix_server_name: string
	token_type: "Bearer"
}

export type RoomVisibility = "public" | "private"
export type RoomPreset = "private_chat" | "public_chat" | "trusted_private_chat"

export interface CreateRoomInitialState {
	type: EventType
	state_key?: string
	content: Record<string, unknown>
}

export interface ReqCreateRoom {
	visibility?: RoomVisibility
	room_alias_name?: string
	name?: string
	topic?: string
	invite?: UserID[]
	preset?: RoomPreset
	is_direct?: boolean
	initial_state?: CreateRoomInitialState[]
	room_version?: string
	creation_content?: Record<string, unknown>
	power_level_content_override?: Record<string, unknown>
	"fi.mau.room_id"?: RoomID
}

export interface RespCreateRoom {
	room_id: RoomID
}

export interface RespTurnServer {
	username: string
	password: string
	ttl: number
	uris: string[]
}

export interface RespMediaConfig {
	"m.upload.size": number
	[key: string]: unknown
}
