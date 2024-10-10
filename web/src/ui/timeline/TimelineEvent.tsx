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
import { RoomStateStore } from "../../api/statestore.ts"
import { getMediaURL } from "../../api/media.ts"
import { DBEvent, MemberEventContent } from "../../api/types"
import HiddenEvent from "./content/HiddenEvent.tsx"
import MessageBody from "./content/MessageBody.tsx"
import { EventContentProps } from "./content/props.ts"
import "./TimelineEvent.css"

export interface TimelineEventProps {
	room: RoomStateStore
	eventRowID: number
}

function getBodyType(evt: DBEvent): React.FunctionComponent<EventContentProps> {
	switch (evt.type) {
	case "m.room.message":
	case "m.sticker":
		return MessageBody
	}
	return HiddenEvent
}

const TimelineEvent = ({ room, eventRowID }: TimelineEventProps) => {
	const evt = room.eventsByRowID.get(eventRowID)
	if (!evt) {
		return null
	}
	const memberEvt = room.getStateEvent("m.room.member", evt.sender)
	const memberEvtContent = memberEvt?.content as MemberEventContent | undefined
	const BodyType = getBodyType(evt)
	// if (BodyType === HiddenEvent) {
	// 	return <div className="timeline-event">
	// 		<BodyType room={room} event={evt}/>
	// 	</div>
	// }
	return <div className="timeline-event">
		<div className="sender-avatar">
			<img loading="lazy" src={getMediaURL(memberEvtContent?.avatar_url)} alt="" />
		</div>
		<div className="sender-and-content">
			<div className="event-sender-and-time">
				<span className="event-sender">{memberEvtContent?.displayname ?? evt.sender}</span>
				<span className="event-time">{new Date(evt.timestamp).toLocaleTimeString()}</span>
			</div>
			<div className="event-content">
				<BodyType room={room} event={evt}/>
			</div>
		</div>
	</div>
}

export default TimelineEvent
