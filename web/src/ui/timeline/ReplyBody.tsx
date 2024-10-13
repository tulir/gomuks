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
import type { EventID, MemDBEvent, MemberEventContent } from "@/api/types"
import { TextMessageBody } from "./content/MessageBody.tsx"
import CloseButton from "@/icons/close.svg?react"
import "./ReplyBody.css"

interface BaseReplyBodyProps {
	room: RoomStateStore
	eventID?: EventID
	event?: MemDBEvent
	onClose?: () => void
}

type ReplyBodyProps = BaseReplyBodyProps & ({eventID: EventID } | {event: MemDBEvent })

const ReplyBody = ({ room, eventID, event, onClose }: ReplyBodyProps) => {
	if (!event) {
		event = room.eventsByID.get(eventID!)
		if (!event) {
			return <blockquote className="reply-body">
				Reply to {eventID}
			</blockquote>
		}
	}
	const memberEvt = room.getStateEvent("m.room.member", event.sender)
	const memberEvtContent = memberEvt?.content as MemberEventContent | undefined
	return <blockquote className={`reply-body ${onClose ? "composer" : ""}`}>
		<div className="reply-sender">
			<div className="sender-avatar" title={event.sender}>
				<img
					className="small avatar"
					loading="lazy"
					src={getAvatarURL(event.sender, memberEvtContent?.avatar_url)}
					alt=""
				/>
			</div>
			<span className="event-sender">{memberEvtContent?.displayname ?? event.sender}</span>
			{onClose && <button className="close-reply" onClick={onClose}><CloseButton/></button>}
		</div>
		<TextMessageBody room={room} event={event}/>
	</blockquote>
}

export default ReplyBody
