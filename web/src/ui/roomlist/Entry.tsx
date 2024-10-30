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
import { getAvatarURL } from "@/api/media.ts"
import type { RoomListEntry } from "@/api/statestore"
import type { MemDBEvent, MemberEventContent } from "@/api/types"
import useContentVisibility from "@/util/contentvisibility.ts"
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
		let displayname = (senderMemberEvt?.content as MemberEventContent)?.displayname
		if (evt.sender === client.userID) {
			displayname = "You"
		} else if (!displayname) {
			displayname = evt.sender.slice(1).split(":")[0]
		}
		return [
			`${displayname}: ${evt.content.body}`,
			`${displayname.length > 16 ? displayname.slice(0, 12) + "…" : displayname}: ${evt.content.body}`,
		]
	}
	return ["", ""]
}

interface InnerProps {
	room: RoomListEntry
}

const EntryInner = ({ room }: InnerProps) => {
	const [previewText, croppedPreviewText] = usePreviewText(room.preview_event, room.preview_sender)
	return <>
		<div className="room-entry-left">
			<img
				loading="lazy"
				className="avatar room-avatar"
				src={getAvatarURL(room.dm_user_id ?? room.room_id, { avatar_url: room.avatar, displayname: room.name })}
				alt=""
			/>
		</div>
		<div className="room-entry-right">
			<div className="room-name">{room.name}</div>
			{previewText && <div className="message-preview" title={previewText}>{croppedPreviewText}</div>}
		</div>
		{room.unread_messages ? <div className="room-entry-unreads">
			<div className={`unread-count ${
				room.unread_notifications ? "notified" : ""} ${
				room.unread_highlights ? "highlighted" : ""}`}
			>
				{room.unread_messages || room.unread_notifications || room.unread_highlights}
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
