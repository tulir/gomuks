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
import React from "react"
import { getAvatarURL } from "../../api/media.ts"
import { RoomStateStore } from "../../api/statestore.ts"
import { MemDBEvent, MemberEventContent } from "../../api/types"
import EncryptedBody from "./content/EncryptedBody.tsx"
import HiddenEvent from "./content/HiddenEvent.tsx"
import MessageBody from "./content/MessageBody.tsx"
import RedactedBody from "./content/RedactedBody.tsx"
import { EventContentProps } from "./content/props.ts"
import "./TimelineEvent.css"

export interface TimelineEventProps {
	room: RoomStateStore
	evt: MemDBEvent
	prevEvt: MemDBEvent | null
}

function getBodyType(evt: MemDBEvent): React.FunctionComponent<EventContentProps> {
	if (evt.relation_type === "m.replace") {
		return HiddenEvent
	}
	switch (evt.type) {
	case "m.room.message":
	case "m.sticker":
		if (evt.redacted_by) {
			return RedactedBody
		}
		return MessageBody
	case "m.room.encrypted":
		if (evt.redacted_by) {
			return RedactedBody
		}
		return EncryptedBody
	}
	return HiddenEvent
}

const fullTimeFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full", timeStyle: "medium" })
const formatShortTime = (time: Date) =>
	`${time.getHours().toString().padStart(2, "0")}:${time.getMinutes().toString().padStart(2, "0")}`

const EventReactions = ({ reactions }: { reactions: Record<string, number> }) => {
	return <div className="event-reactions">
		{Object.entries(reactions).map(([reaction, count]) => <span key={reaction} className="reaction">
			{reaction} {count}
		</span>)}
	</div>
}

const TimelineEvent = ({ room, evt, prevEvt }: TimelineEventProps) => {
	const memberEvt = room.getStateEvent("m.room.member", evt.sender)
	const memberEvtContent = memberEvt?.content as MemberEventContent | undefined
	const BodyType = getBodyType(evt)
	const eventTS = new Date(evt.timestamp)
	const editEventTS = evt.last_edit ? new Date(evt.last_edit.timestamp) : null
	const wrapperClassNames = ["timeline-event"]
	if (BodyType === HiddenEvent) {
		wrapperClassNames.push("hidden-event")
	} else if (prevEvt?.sender === evt.sender && getBodyType(prevEvt) !== HiddenEvent) {
		wrapperClassNames.push("same-sender")
	}
	const fullTime = fullTimeFormatter.format(eventTS)
	const shortTime = formatShortTime(eventTS)
	const editTime = editEventTS ? `Edited at ${fullTimeFormatter.format(editEventTS)}` : null
	return <div className={wrapperClassNames.join(" ")}>
		<div className="sender-avatar" title={evt.sender}>
			<img
				className="avatar"
				loading="lazy"
				src={getAvatarURL(evt.sender, memberEvtContent?.avatar_url)}
				alt=""
			/>
		</div>
		<div className="event-sender-and-time">
			<span className="event-sender">{memberEvtContent?.displayname ?? evt.sender}</span>
			<span className="event-time" title={fullTime}>{shortTime}</span>
			{(editEventTS && editTime) ? <span className="event-edited" title={editTime}>
				(edited at {formatShortTime(editEventTS)})
			</span> : null}
		</div>
		<div className="event-time-only">
			<span className="event-time" title={editTime ? `${fullTime} - ${editTime}` : fullTime}>{shortTime}</span>
		</div>
		<div className="event-content">
			<BodyType room={room} event={evt}/>
			{evt.reactions ? <EventReactions reactions={evt.reactions}/> : null}
		</div>
	</div>
}

export default TimelineEvent
