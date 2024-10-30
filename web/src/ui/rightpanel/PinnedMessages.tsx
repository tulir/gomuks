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
import { RoomStateStore, useRoomEvent, useRoomState } from "@/api/statestore"
import { EventID, PinnedEventsContent } from "@/api/types"
import reverseMap from "@/util/reversemap.ts"
import ClientContext from "../ClientContext.ts"
import { RoomContext } from "../roomview/roomcontext.ts"
import TimelineEvent from "../timeline/TimelineEvent.tsx"

interface PinnedMessageProps {
	evtID: EventID
	room: RoomStateStore
}

const PinnedMessage = ({ evtID, room }: PinnedMessageProps) => {
	const evt = useRoomEvent(room, evtID)
	if (!evt) {
		// This caches whether the event is requested or not, so it doesn't need to be wrapped in an effect.
		use(ClientContext)!.requestEvent(room, evtID)
		return <>Event {evtID} not found</>
	}
	return <TimelineEvent evt={evt} prevEvt={null} />
}

const PinnedMessages = () => {
	const roomCtx = use(RoomContext)
	const pins = useRoomState(roomCtx?.store, "m.room.pinned_events", "")?.content as PinnedEventsContent | undefined
	if (!roomCtx) {
		return null
	} else if (!Array.isArray(pins?.pinned) || pins.pinned.length === 0) {
		return <div className="empty">No pinned messages</div>
	}
	return <>
		{reverseMap(pins.pinned, evtID => typeof evtID === "string" ? <div className="pinned-event" key={evtID}>
			<PinnedMessage evtID={evtID} room={roomCtx.store} />
		</div> : null)}
	</>
}

export default PinnedMessages
