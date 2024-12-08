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
import { memo, use } from "react"
import { getRoomAvatarURL } from "@/api/media.ts"
import type { RoomListEntry } from "@/api/statestore"
import type { MemDBEvent, MemberEventContent } from "@/api/types"
import useContentVisibility from "@/util/contentvisibility.ts"
import { getDisplayname } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"

export interface RoomListEntryProps {
	room: RoomListEntry
	isActive: boolean
	hidden: boolean
}

function usePreviewText(evt?: MemDBEvent, senderMemberEvt?: MemDBEvent | null): [string, string] {
	if (!evt) {
		return ["", ""]
	}
	if ((evt.type === "m.room.message" || evt.type === "m.sticker") && typeof evt.content.body === "string") {
		const client = use(ClientContext)!
		const displayname = evt.sender === client.userID
			? "You"
			: getDisplayname(evt.sender, senderMemberEvt?.content as MemberEventContent)
		let previewText = evt.content.body
		if (evt.content.formatted_body?.includes?.("data-mx-spoiler")) {
			previewText = "<message contains spoilers>"
		}
		return [
			`${displayname}: ${evt.content.body}`,
			`${displayname.length > 16 ? displayname.slice(0, 12) + "…" : displayname}: ${previewText}`,
		]
	}
	return ["", ""]
}

interface InnerProps {
	room: RoomListEntry
}

const EntryInner = ({ room }: InnerProps) => {
	const [previewText, croppedPreviewText] = usePreviewText(room.preview_event, room.preview_sender)
	const unreadCount = room.unread_messages || room.unread_notifications || room.unread_highlights
	const countIsBig = Boolean(room.unread_notifications || room.unread_highlights)
	let unreadCountDisplay = unreadCount.toString()
	if (unreadCount > 999 && countIsBig) {
		unreadCountDisplay = "99+"
	} else if (unreadCount > 9999 && countIsBig) {
		unreadCountDisplay = "999+"
	}

	return <>
		<div className="room-entry-left">
			<img
				loading="lazy"
				className="avatar room-avatar"
				src={getRoomAvatarURL(room)}
				alt=""
			/>
		</div>
		<div className="room-entry-right">
			<div className="room-name">{room.name}</div>
			{previewText && <div className="message-preview" title={previewText}>{croppedPreviewText}</div>}
		</div>
		{(room.unread_messages || room.marked_unread) ? <div className="room-entry-unreads">
			<div title={unreadCount.toString()} className={`unread-count ${
				room.marked_unread ? "marked-unread" : ""} ${
				room.unread_notifications ? "notified" : ""} ${
				room.unread_highlights ? "highlighted" : ""}`}
			>
				{unreadCountDisplay}
			</div>
		</div> : null}
	</>
}

const Entry = ({ room, isActive, hidden }: RoomListEntryProps) => {
	const [isVisible, divRef] = useContentVisibility<HTMLDivElement>()
	return <div
		ref={divRef}
		className={`room-entry ${isActive ? "active" : ""} ${hidden ? "hidden" : ""}`}
		onClick={use(MainScreenContext).clickRoom}
		data-room-id={room.room_id}
	>
		{isVisible ? <EntryInner room={room}/> : null}
	</div>
}

export default memo(Entry)
