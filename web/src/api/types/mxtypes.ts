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
export type RoomVersion = "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" | "10" | "11"
export type RoomType = "" | "m.space"
export type RelationType = "m.annotation" | "m.reference" | "m.replace" | "m.thread"

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

export interface LazyLoadSummary {
	heroes?: UserID[]
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

export interface MemberEventContent {
	membership: "join" | "leave" | "ban" | "invite" | "knock"
	displayname?: string
	avatar_url?: ContentURI
	reason?: string
}
