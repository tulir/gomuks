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
import { RoomStateStore, useRoomEvent, useRoomState } from "@/api/statestore"
import type { EventID, MemDBEvent, MemberEventContent } from "@/api/types"
import { ClientContext } from "../ClientContext.ts"
import getBodyType, { ContentErrorBoundary } from "./content"
import CloseButton from "@/icons/close.svg?react"
import "./ReplyBody.css"

interface ReplyBodyProps {
	room: RoomStateStore
	event: MemDBEvent
	isThread: boolean
	isEditing?: boolean
	onClose?: (evt: React.MouseEvent) => void
}

interface ReplyIDBodyProps {
	room: RoomStateStore
	eventID: EventID
	isThread: boolean
}

export const ReplyIDBody = ({ room, eventID, isThread }: ReplyIDBodyProps) => {
	const event = useRoomEvent(room, eventID)
	if (!event) {
		// This caches whether the event is requested or not, so it doesn't need to be wrapped in an effect.
		use(ClientContext)!.requestEvent(room, eventID)
		return <blockquote className="reply-body">
			Reply to unknown event<br/><code>{eventID}</code>
		</blockquote>
	}
	return <ReplyBody room={room} event={event} isThread={isThread}/>
}

const onClickReply = (evt: React.MouseEvent) => {
	const targetEvt = document.querySelector(`div[data-event-id="${evt.currentTarget.getAttribute("data-reply-to")}"]`)
	if (targetEvt) {
		targetEvt.scrollIntoView({
			block: "center",
		})
		targetEvt.classList.add("jump-highlight")
		setTimeout(() => {
			targetEvt.classList.add("jump-highlight-fadeout")
			targetEvt.classList.remove("jump-highlight")
			setTimeout(() => {
				targetEvt.classList.remove("jump-highlight-fadeout")
			}, 1500)
		}, 3000)
	}
}

export const ReplyBody = ({ room, event, onClose, isThread, isEditing }: ReplyBodyProps) => {
	const memberEvt = useRoomState(room, "m.room.member", event.sender)
	const memberEvtContent = memberEvt?.content as MemberEventContent | undefined
	const BodyType = getBodyType(event, true)
	const classNames = ["reply-body"]
	if (onClose) {
		classNames.push("composer")
	}
	if (isThread) {
		classNames.push("thread")
	}
	if (isEditing) {
		classNames.push("editing")
	}
	return <blockquote data-reply-to={event.event_id} className={classNames.join(" ")} onClick={onClickReply}>
		<div className="reply-sender">
			<div className="sender-avatar" title={event.sender}>
				<img
					className="small avatar"
					loading="lazy"
					src={getAvatarURL(event.sender, memberEvtContent)}
					alt=""
				/>
			</div>
			<span className="event-sender">{memberEvtContent?.displayname || event.sender}</span>
			{onClose && <button className="close-reply" onClick={onClose}><CloseButton/></button>}
		</div>
		<ContentErrorBoundary>
			<BodyType room={room} event={event} sender={memberEvt}/>
		</ContentErrorBoundary>
	</blockquote>
}
