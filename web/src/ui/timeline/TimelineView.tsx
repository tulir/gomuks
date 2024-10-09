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
	const timelineViewRef = useRef<HTMLDivElement>(null)
	const prevOldestTimelineRow = useRef(0)
	const oldScrollHeight = useRef(0)
	const scrolledToBottom = useRef(true)

	// When the user scrolls the timeline manually, remember if they were at the bottom,
	// so that we can keep them at the bottom when new events are added.
	const handleScroll = useMemo(() => () => {
		if (!timelineViewRef.current) {
			return
		}
		const timelineView = timelineViewRef.current
		scrolledToBottom.current = timelineView.scrollTop + timelineView.clientHeight + 1 >= timelineView.scrollHeight
	}, [])
	// Save the scroll height prior to updating the timeline, so that we can adjust the scroll position if needed.
	if (timelineViewRef.current) {
		oldScrollHeight.current = timelineViewRef.current.scrollHeight
	}
	useEffect(() => {
		if (bottomRef.current && scrolledToBottom.current) {
			// For any timeline changes, if we were at the bottom, scroll to the new bottom
			bottomRef.current.scrollIntoView()
		} else if (timelineViewRef.current && prevOldestTimelineRow.current > timeline[0]?.timeline_rowid) {
			// When new entries are added to the top of the timeline, scroll down to keep the same position
			timelineViewRef.current.scrollTop += timelineViewRef.current.scrollHeight - oldScrollHeight.current
		}
		prevOldestTimelineRow.current = timeline[0]?.timeline_rowid ?? 0
	}, [timeline])

	return <div className="timeline-view" onScroll={handleScroll} ref={timelineViewRef}>
		<div className="timeline-beginning">
			<button onClick={loadHistory}>Load history</button>
		</div>
		<div className="timeline-list">
			{timeline.map(entry => <TimelineEvent
				key={entry.event_rowid} room={room} eventRowID={entry.event_rowid}
			/>)}
			<div className="timeline-bottom-ref" ref={bottomRef}/>
		</div>
	</div>
}

export default TimelineView
