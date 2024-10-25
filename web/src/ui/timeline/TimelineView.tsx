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
import { use, useCallback, useEffect, useLayoutEffect, useRef } from "react"
import { useRoomTimeline } from "@/api/statestore"
import { MemDBEvent } from "@/api/types"
import useFocus from "@/util/focus.ts"
import { ClientContext } from "../ClientContext.ts"
import { useRoomContext } from "../roomcontext.ts"
import TimelineEvent from "./TimelineEvent.tsx"
import "./TimelineView.css"

const TimelineView = () => {
	const roomCtx = useRoomContext()
	const room = roomCtx.store
	const timeline = useRoomTimeline(room)
	const client = use(ClientContext)!
	const loadHistory = useCallback(() => {
		client.loadMoreHistory(room.roomID)
			.catch(err => console.error("Failed to load history", err))
	}, [client, room])
	const bottomRef = roomCtx.timelineBottomRef
	const topRef = useRef<HTMLDivElement>(null)
	const timelineViewRef = useRef<HTMLDivElement>(null)
	const prevOldestTimelineRow = useRef(0)
	const oldScrollHeight = useRef(0)
	const focused = useFocus()

	// When the user scrolls the timeline manually, remember if they were at the bottom,
	// so that we can keep them at the bottom when new events are added.
	const handleScroll = useCallback(() => {
		if (!timelineViewRef.current) {
			return
		}
		const timelineView = timelineViewRef.current
		roomCtx.scrolledToBottom = timelineView.scrollTop + timelineView.clientHeight + 1 >= timelineView.scrollHeight
	}, [roomCtx])
	// Save the scroll height prior to updating the timeline, so that we can adjust the scroll position if needed.
	if (timelineViewRef.current) {
		oldScrollHeight.current = timelineViewRef.current.scrollHeight
	}
	useLayoutEffect(() => {
		const bottomRef = roomCtx.timelineBottomRef
		if (bottomRef.current && roomCtx.scrolledToBottom) {
			// For any timeline changes, if we were at the bottom, scroll to the new bottom
			bottomRef.current.scrollIntoView()
		} else if (timelineViewRef.current && prevOldestTimelineRow.current > (timeline[0]?.timeline_rowid ?? 0)) {
			// When new entries are added to the top of the timeline, scroll down to keep the same position
			timelineViewRef.current.scrollTop += timelineViewRef.current.scrollHeight - oldScrollHeight.current
		}
		prevOldestTimelineRow.current = timeline[0]?.timeline_rowid ?? 0
		roomCtx.ownMessages = timeline
			.filter(evt => evt !== null
				&& evt.sender === client.userID
				&& evt.type === "m.room.message"
				&& evt.relation_type !== "m.replace")
			.map(evt => evt!.rowid)
	}, [client.userID, roomCtx, timeline])
	useEffect(() => {
		const newestEvent = timeline[timeline.length - 1]
		if (
			roomCtx.scrolledToBottom
			&& focused
			&& newestEvent
			&& newestEvent.timeline_rowid > 0
			&& room.readUpToRow < newestEvent.timeline_rowid
			&& newestEvent.sender !== client.userID
		) {
			room.readUpToRow = newestEvent.timeline_rowid
			client.rpc.markRead(room.roomID, newestEvent.event_id, "m.read").then(
				() => console.log("Marked read up to", newestEvent.event_id, newestEvent.timeline_rowid),
				err => console.error(`Failed to send read receipt for ${newestEvent.event_id}:`, err),
			)
		}
	}, [focused, client, roomCtx, room, timeline])
	useEffect(() => {
		const topElem = topRef.current
		if (!topElem) {
			return
		}
		const observer = new IntersectionObserver(entries => {
			if (entries[0]?.isIntersecting && room.paginationRequestedForRow !== prevOldestTimelineRow.current) {
				room.paginationRequestedForRow = prevOldestTimelineRow.current
				loadHistory()
			}
		}, {
			root: topElem.parentElement!.parentElement,
			rootMargin: "0px",
			threshold: 1.0,
		})
		observer.observe(topElem)
		return () => observer.unobserve(topElem)
	}, [room, loadHistory])

	let prevEvt: MemDBEvent | null = null
	return <div className="timeline-view" onScroll={handleScroll} ref={timelineViewRef}>
		<div className="timeline-beginning">
			<button onClick={loadHistory}>Load history</button>
		</div>
		<div className="timeline-list">
			<div className="timeline-top-ref" ref={topRef}/>
			{timeline.map(entry => {
				if (!entry) {
					return null
				}
				const thisEvt = <TimelineEvent
					key={entry.rowid} evt={entry} prevEvt={prevEvt}
				/>
				prevEvt = entry
				return thisEvt
			})}
			<div className="timeline-bottom-ref" ref={bottomRef}/>
		</div>
	</div>
}

export default TimelineView
