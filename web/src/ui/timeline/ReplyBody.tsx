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
import { getAvatarURL } from "@/api/media.ts"
import { RoomStateStore, useRoomEvent } from "@/api/statestore"
import type { EventID, MemDBEvent, MemberEventContent } from "@/api/types"
import { ClientContext } from "../ClientContext.ts"
import { TextMessageBody } from "./content/MessageBody.tsx"
import CloseButton from "@/icons/close.svg?react"
import "./ReplyBody.css"

interface ReplyBodyProps {
	room: RoomStateStore
	event: MemDBEvent
	onClose?: () => void
}

interface ReplyIDBodyProps {
	room: RoomStateStore
	eventID: EventID
}

export const ReplyIDBody = ({ room, eventID }: ReplyIDBodyProps) => {
	const event = useRoomEvent(room, eventID)
	if (!event) {
		// This caches whether the event is requested or not, so it doesn't need to be wrapped in an effect.
		use(ClientContext)!.requestEvent(room, eventID)
		return <blockquote className="reply-body">
			Reply to unknown event<br/><code>{eventID}</code>
		</blockquote>
	}
	return <ReplyBody room={room} event={event}/>
}

export const ReplyBody = ({ room, event, onClose }: ReplyBodyProps) => {
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
