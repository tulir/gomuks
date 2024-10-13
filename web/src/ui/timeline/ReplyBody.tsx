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
import { getAvatarURL } from "@/api/media.ts"
import type { RoomStateStore } from "@/api/statestore.ts"
import type { EventID, MemberEventContent } from "@/api/types"
import { TextMessageBody } from "./content/MessageBody.tsx"
import "./ReplyBody.css"

interface ReplyBodyProps {
	room: RoomStateStore
	eventID: EventID
}

const ReplyBody = ({ room, eventID }: ReplyBodyProps) => {
	const evt = room.eventsByID.get(eventID)
	if (!evt) {
		return <blockquote className="reply-body">
			Reply to {eventID}
		</blockquote>
	}
	const memberEvt = room.getStateEvent("m.room.member", evt.sender)
	const memberEvtContent = memberEvt?.content as MemberEventContent | undefined
	return <blockquote className="reply-body">
		<div className="reply-sender">
			<div className="sender-avatar" title={evt.sender}>
				<img
					className="avatar"
					loading="lazy"
					src={getAvatarURL(evt.sender, memberEvtContent?.avatar_url)}
					alt=""
				/>
			</div>
			<span className="event-sender">{memberEvtContent?.displayname ?? evt.sender}</span>
		</div>
		<TextMessageBody room={room} event={evt}/>
	</blockquote>
}

export default ReplyBody
