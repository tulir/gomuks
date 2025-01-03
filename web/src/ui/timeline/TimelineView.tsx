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
import { Virtualizer, useVirtualizer } from "@tanstack/react-virtual"
import { use, useCallback, useEffect, useLayoutEffect, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { usePreference, useRoomTimeline } from "@/api/statestore"
import { EventRowID, MemDBEvent } from "@/api/types"
import useFocus from "@/util/focus.ts"
import ClientContext from "../ClientContext.ts"
import { useRoomContext } from "../roomview/roomcontext.ts"
import TimelineEvent from "./TimelineEvent.tsx"
import { getBodyType, isSmallEvent } from "./content/index.ts"
import "./TimelineView.css"

// This is necessary to take into account margin, which the default measurement
// (using getBoundingClientRect) doesn't by default
const measureElement = (
	element: Element, entry: ResizeObserverEntry | undefined, instance: Virtualizer<HTMLDivElement, Element>,
) => {
	const horizontal = instance.options.horizontal
	const style = window.getComputedStyle(element)
	if (entry == null ? void 0 : entry.borderBoxSize) {
		const box = entry?.borderBoxSize[0]
		if (box) {
			const size = Math.round(
				box[horizontal ? "inlineSize" : "blockSize"],
			)
			return size
				+ parseFloat(style[horizontal ? "marginInlineStart" : "marginBlockStart"])
				+ parseFloat(style[horizontal ? "marginInlineEnd" : "marginBlockEnd"])
		}
	}
	return Math.round(
		element.getBoundingClientRect()[instance.options.horizontal ? "width" : "height"]
		+ parseFloat(style[horizontal ? "marginLeft" : "marginTop"])
		+ parseFloat(style[horizontal ? "marginRight" : "marginBottom"]),
	)
}

const estimateEventHeight = (event: MemDBEvent) => isSmallEvent(getBodyType(event)) ?
	(event.reactions ? 26 : 0)
	+ (event.content.body ? (event.local_content?.big_emoji ? 92 : 44) : 0)
	+ (event.content.info?.h || 0)
	: 26

const TimelineView = () => {
	const roomCtx = useRoomContext()
	const room = roomCtx.store
	const timeline = useRoomTimeline(room)
	const client = use(ClientContext)!
	const [isLoadingHistory, setLoadingHistory] = useState(false)
	const [focusedEventRowID, directSetFocusedEventRowID] = useState<EventRowID | null>(null)
	const bottomRef = roomCtx.timelineBottomRef
	const timelineViewRef = useRef<HTMLDivElement>(null)
	const focused = useFocus()
	const smallReplies = usePreference(client.store, room, "small_replies")

	const virtualListRef = useRef<HTMLDivElement>(null)

	const virtualizer = useVirtualizer({
		count: timeline.length,
		getScrollElement: () => timelineViewRef.current,
		estimateSize: (index) => timeline[index] ? estimateEventHeight(timeline[index]) : 0,
		getItemKey: (index) => timeline[index]?.rowid || index,
		overscan: 6,
		measureElement,
	})

	const items = virtualizer.getVirtualItems()

	const loadHistory = useCallback(() => {
		setLoadingHistory(true)
		client.loadMoreHistory(room.roomID)
			.catch(err => console.error("Failed to load history", err))
			.then((loadedEventCount) => {
				// Prevent scroll getting stuck loading more history
				if (loadedEventCount &&
					timelineViewRef.current &&
					timelineViewRef.current.scrollTop <= (virtualListRef.current?.offsetTop ?? 0)) {
					// FIXME: This seems to run before the events are measured,
					// resulting in a jump in the timeline of the difference in
					// height when scrolling very fast
					virtualizer.scrollToIndex(loadedEventCount, { align: "end" })
				}
			})
			.finally(() => {
				setLoadingHistory(false)
			})
	}, [client, room, virtualizer])

	useLayoutEffect(() => {
		if (roomCtx.scrolledToBottom) {
			// timelineViewRef.current && (timelineViewRef.current.scrollTop = timelineViewRef.current.scrollHeight)
			bottomRef.current?.scrollIntoView()
		}
	}, [roomCtx, timeline, virtualizer.getTotalSize(), bottomRef])

	// When the user scrolls the timeline manually, remember if they were at the bottom,
	// so that we can keep them at the bottom when new events are added.
	const handleScroll = () => {
		if (!timelineViewRef.current) {
			return
		}
		const timelineView = timelineViewRef.current
		roomCtx.scrolledToBottom = timelineView.scrollTop + timelineView.clientHeight + 1 >= timelineView.scrollHeight
	}

	useEffect(() => {
		roomCtx.directSetFocusedEventRowID = directSetFocusedEventRowID
	}, [roomCtx])

	useEffect(() => {
		const newestEvent = timeline[timeline.length - 1]
		if (
			roomCtx.scrolledToBottom
			&& focused
			&& newestEvent
			&& newestEvent.timeline_rowid > 0
			&& (room.meta.current.marked_unread
				|| (room.readUpToRow < newestEvent.timeline_rowid
					&& newestEvent.sender !== client.userID))
		) {
			room.readUpToRow = newestEvent.timeline_rowid
			room.meta.current.marked_unread = false
			const receiptType = roomCtx.store.preferences.send_read_receipts ? "m.read" : "m.read.private"
			client.rpc.markRead(room.roomID, newestEvent.event_id, receiptType).then(
				() => console.log("Marked read up to", newestEvent.event_id, newestEvent.timeline_rowid),
				err => {
					console.error(`Failed to send read receipt for ${newestEvent.event_id}:`, err)
					room.readUpToRow = -1
				},
			)
		}
	}, [focused, client, roomCtx, room, timeline])

	const firstItem = items[0]

	useEffect(() => {
		if (!room.hasMoreHistory || room.paginating) {
			return
		}


		// Load history if there is none
		if (!firstItem) {
			loadHistory()
			return
		}

		// Load more history when the virtualizer loads the last item
		if (firstItem.index == 0) {
			console.log("Loading more history...")
			loadHistory()
			return
		}
	}, [
		room.hasMoreHistory, loadHistory,
		room.paginating,
		firstItem,
	])

	return <div className="timeline-view" onScroll={handleScroll} ref={timelineViewRef}>
		<div className="timeline-beginning">
			{room.hasMoreHistory ? <button onClick={loadHistory} disabled={isLoadingHistory}>
				{isLoadingHistory
					? <><ScaleLoader color="var(--primary-color)"/> Loading history...</>
					: "Load more history"}
			</button> : "No more history available in this room"}
		</div>
		<div
			style={{
				height: virtualizer.getTotalSize(),
				width: "100%",
				position: "relative",
			}}
			className="timeline-list"
			ref={virtualListRef}
		>
			<div
				style={{
					position: "absolute",
					top: 0,
					left: 0,
					width: "100%",
					transform: `translateY(${items[0]?.start ?? 0}px)`,
				}}
				className="timeline-virtual-items"
			>

				{items.map((virtualRow) => {
					const entry = timeline[virtualRow.index]
					if (!entry) {
						return null
					}
					const thisEvt = <TimelineEvent
						evt={entry}
						prevEvt={timeline[virtualRow.index - 1] ?? null}
						smallReplies={smallReplies}
						isFocused={focusedEventRowID === entry.rowid}

						key={virtualRow.key}
						virtualIndex={virtualRow.index}
						ref={virtualizer.measureElement}
					/>

					return thisEvt
				})}
			</div>
		</div>
		<div className="timeline-bottom-ref" ref={bottomRef}/>
	</div>
}

export default TimelineView
