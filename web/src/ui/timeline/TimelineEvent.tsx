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
import { DBEvent, MemberEventContent } from "../../api/types"
import { EventContentProps } from "./content/props.ts"
import HiddenEvent from "./content/HiddenEvent.tsx"
import "./TimelineEvent.css"
import MessageBody from "./content/MessageBody.tsx"

export interface TimelineEventProps {
	room: RoomStateStore
	eventRowID: number
}

function getBodyType(evt: DBEvent): React.FunctionComponent<EventContentProps> {
	switch (evt.type) {
	case "m.room.message":
		return MessageBody
	case "m.sticker":
	}
	return HiddenEvent
}

const TimelineEvent = ({ room, eventRowID }: TimelineEventProps) => {
	const evt = room.eventsByRowID.get(eventRowID)
	if (!evt) {
		return null
	}
	const BodyType = getBodyType(evt)
	if (BodyType === HiddenEvent) {
		return <div className="timeline-event">
			<BodyType room={room} event={evt}/>
		</div>
	}
	return <div className="timeline-event">
		<div className="event-sender">
			{evt.sender}
		</div>
		<BodyType room={room} event={evt}/>
	</div>
}

export default TimelineEvent
