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
import { use, useEffect, useMemo, useRef } from "react"
import { RoomStateStore } from "../../api/statestore.ts"
import { useNonNullEventAsState } from "../../util/eventdispatcher.ts"
import { ClientContext } from "../ClientContext.ts"
import TimelineEvent from "./TimelineEvent.tsx"
import "./TimelineView.css"

interface TimelineViewProps {
	room: RoomStateStore
}

const TimelineView = ({ room }: TimelineViewProps) => {
	const timeline = useNonNullEventAsState(room.timeline)
	const client = use(ClientContext)!
	const loadHistory = useMemo(() =>  () => {
		client.loadMoreHistory(room.roomID)
			.catch(err => console.error("Failed to load history", err))
	}, [client, room.roomID])
	const bottomRef = useRef<HTMLDivElement>(null)
	const isAtBottom = useRef(true)
	const handleScroll = useMemo(() => () => {
		if (!bottomRef.current?.parentElement) {
			return
		}
		const timelineView = bottomRef.current.parentElement
		isAtBottom.current = timelineView.scrollTop + timelineView.clientHeight + 1 >= timelineView.scrollHeight
	}, [])
	useEffect(() => {
		if (bottomRef.current && isAtBottom.current) {
			bottomRef.current.scrollIntoView()
		}
	}, [timeline])
	return <div className="timeline-view" onScroll={handleScroll}>
		<button onClick={loadHistory}>Load history</button>
		{timeline.map(entry => <TimelineEvent
			key={entry.event_rowid} room={room} eventRowID={entry.event_rowid}
		/>)}
		<div ref={bottomRef}/>
	</div>
}

export default TimelineView
