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
import toSearchableString from "@/util/searchablestring.ts"
import { ensureString, getDisplayname } from "@/util/validation.ts"
import type {
	ContentURI,
	DBInvitedRoom, JoinRule,
	MemberEventContent, Membership,
	RoomAlias,
	RoomID,
	RoomSummary,
	RoomType,
	RoomVersion,
	StrippedStateEvent,
	UserID,
} from "../types"
import type { RoomListEntry, StateStore } from "./main.ts"

export class InvitedRoomStore implements RoomListEntry, RoomSummary {
	readonly room_id: RoomID
	readonly sorting_timestamp: number
	readonly date: string
	readonly name: string = ""
	readonly search_name: string
	readonly dm_user_id?: UserID
	readonly canonical_alias?: RoomAlias
	readonly topic?: string
	readonly avatar?: ContentURI
	readonly encryption?: "m.megolm.v1.aes-sha2"
	readonly room_version?: RoomVersion
	readonly join_rule?: JoinRule
	readonly invited_by?: UserID
	readonly inviter_profile?: MemberEventContent
	readonly is_direct: boolean
	readonly is_invite = true

	constructor(public readonly meta: DBInvitedRoom, parent: StateStore) {
		this.room_id = meta.room_id
		this.sorting_timestamp = 1000000000000000 + meta.created_at
		this.date = new Date(meta.created_at - new Date().getTimezoneOffset() * 60000)
			.toISOString().replace("T", " ").replace("Z", "")
		const members = new Map<UserID, StrippedStateEvent>()
		for (const state of this.meta.invite_state) {
			if (state.type === "m.room.name") {
				this.name = ensureString(state.content.name)
			} else if (state.type === "m.room.canonical_alias") {
				this.canonical_alias = ensureString(state.content.alias)
			} else if (state.type === "m.room.topic") {
				this.topic = ensureString(state.content.topic)
			} else if (state.type === "m.room.avatar") {
				this.avatar = ensureString(state.content.url)
			} else if (state.type === "m.room.encryption") {
				this.encryption = state.content.algorithm as "m.megolm.v1.aes-sha2"
			} else if (state.type === "m.room.create") {
				this.room_version = ensureString(state.content.version) as RoomVersion
			} else if (state.type === "m.room.member") {
				members.set(state.state_key, state)
			} else if (state.type === "m.room.join_rules") {
				this.join_rule = ensureString(state.content.join_rule) as JoinRule
			}
		}
		this.search_name = toSearchableString(this.name ?? "")
		const ownMemberEvt = members.get(parent.userID)
		if (ownMemberEvt) {
			this.invited_by = ownMemberEvt.sender
			this.inviter_profile = members.get(ownMemberEvt.sender)?.content as MemberEventContent
		}
		this.is_direct = Boolean(ownMemberEvt?.content.is_direct)
		if (
			!this.name
			&& !this.avatar
			&& !this.topic
			&& !this.canonical_alias
			&& this.join_rule === "invite"
			&& this.invited_by
			&& this.is_direct
		) {
			this.dm_user_id = this.invited_by
			this.name = getDisplayname(this.invited_by, this.inviter_profile)
			this.avatar = this.inviter_profile?.avatar_url
		}
	}

	get membership(): Membership {
		return "invite"
	}

	get avatar_url(): ContentURI | undefined {
		return this.avatar
	}

	get num_joined_members(): number {
		return 0
	}

	get room_type(): RoomType {
		return ""
	}

	get world_readable(): boolean {
		return false
	}

	get guest_can_join(): boolean {
		return false
	}

	get unread_messages(): number {
		return 0
	}

	get unread_notifications(): number {
		return 0
	}

	get unread_highlights(): number {
		return 1
	}

	get marked_unread(): boolean {
		return true
	}
}
