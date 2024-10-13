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
import { use } from "react"
import { getMediaURL } from "../../api/media.ts"
import type { RoomListEntry } from "../../api/statestore.ts"
import type { MemDBEvent, MemberEventContent } from "../../api/types"
import { ClientContext } from "../ClientContext.ts"

export interface RoomListEntryProps {
	room: RoomListEntry
	setActiveRoom: (evt: React.MouseEvent) => void
	isActive: boolean
	hidden: boolean
}

function usePreviewText(evt?: MemDBEvent, senderMemberEvt?: MemDBEvent): [string, string] {
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

const Entry = ({ room, setActiveRoom, isActive, hidden }: RoomListEntryProps) => {
	const [previewText, croppedPreviewText] = usePreviewText(room.preview_event, room.preview_sender)
	return <div
		className={`room-entry ${isActive ? "active" : ""} ${hidden ? "hidden" : ""}`}
		onClick={setActiveRoom}
		data-room-id={room.room_id}
	>
		<div className="room-entry-left">
			<img loading="lazy" className="avatar room-avatar" src={getMediaURL(room.avatar)} alt=""/>
		</div>
		<div className="room-entry-right">
			<div className="room-name">{room.name}</div>
			{previewText && <div className="message-preview" title={previewText}>{croppedPreviewText}</div>}
		</div>
	</div>
}

export default Entry
