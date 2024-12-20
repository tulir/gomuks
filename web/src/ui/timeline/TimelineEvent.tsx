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
import React, { use, useCallback, useState } from "react"
import { getAvatarURL, getMediaURL, getUserColorIndex } from "@/api/media.ts"
import { useRoomMember } from "@/api/statestore"
import { MemDBEvent, MemberEventContent, UnreadType } from "@/api/types"
import { isMobileDevice } from "@/util/ismobile.ts"
import { getDisplayname, isEventID } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { ModalContext } from "../modal/Modal.tsx"
import { useRoomContext } from "../roomview/roomcontext.ts"
import ReadReceipts from "./ReadReceipts.tsx"
import { ReplyIDBody } from "./ReplyBody.tsx"
import { ContentErrorBoundary, HiddenEvent, getBodyType, isSmallEvent } from "./content"
import { EventFullMenu, EventHoverMenu, getModalStyleFromMouse } from "./menu"
import ErrorIcon from "@/icons/error.svg?react"
import PendingIcon from "@/icons/pending.svg?react"
import SentIcon from "@/icons/sent.svg?react"
import "./TimelineEvent.css"

export interface TimelineEventProps {
	evt: MemDBEvent
	prevEvt: MemDBEvent | null
	disableMenu?: boolean
}

const fullTimeFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full", timeStyle: "medium" })
const dateFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full" })
const formatShortTime = (time: Date) =>
	`${time.getHours().toString().padStart(2, "0")}:${time.getMinutes().toString().padStart(2, "0")}`

const EventReactions = ({ reactions }: { reactions: Record<string, number> }) => {
	const reactionEntries = Object.entries(reactions).filter(([, count]) => count > 0).sort((a, b) => b[1] - a[1])
	if (reactionEntries.length === 0) {
		return null
	}
	return <div className="event-reactions">
		{reactionEntries.map(([reaction, count]) =>
			<div key={reaction} className="reaction" title={reaction}>
				{reaction.startsWith("mxc://")
					? <img className="reaction-emoji" src={getMediaURL(reaction)} alt=""/>
					: <span className="reaction-emoji">{reaction}</span>}
				<span className="reaction-count">{count}</span>
			</div>)}
	</div>
}

const EventSendStatus = ({ evt }: { evt: MemDBEvent }) => {
	if (evt.send_error && evt.send_error !== "not sent") {
		return <div className="event-send-status error" title={evt.send_error}><ErrorIcon/></div>
	} else if (evt.event_id.startsWith("~")) {
		return <div title="Waiting for /send to return" className="event-send-status sending"><PendingIcon/></div>
	} else if (evt.pending) {
		return <div title="Waiting to receive event in /sync" className="event-send-status sent"><SentIcon/></div>
	} else {
		return <div title="Event sent and remote echo received" className="event-send-status sent"><SentIcon/></div>
	}
}

const TimelineEvent = ({ evt, prevEvt, disableMenu }: TimelineEventProps) => {
	const roomCtx = useRoomContext()
	const client = use(ClientContext)!
	const mainScreen = use(MainScreenContext)
	const openModal = use(ModalContext)
	const [forceContextMenuOpen, setForceContextMenuOpen] = useState(false)
	const onContextMenu = useCallback((mouseEvt: React.MouseEvent) => {
		const targetElem = mouseEvt.target as HTMLElement
		if (
			!roomCtx.store.preferences.message_context_menu
			|| targetElem.tagName === "A"
			|| targetElem.tagName === "IMG"
			|| window.getSelection()?.type === "Range"
		) {
			return
		}
		mouseEvt.preventDefault()
		openModal({
			content: <EventFullMenu
				evt={evt}
				roomCtx={roomCtx}
				style={getModalStyleFromMouse(mouseEvt, 9 * 40)}
			/>,
		})
	}, [openModal, evt, roomCtx])
	const memberEvt = useRoomMember(client, roomCtx.store, evt.sender)
	const memberEvtContent = memberEvt?.content as MemberEventContent | undefined
	const BodyType = getBodyType(evt)
	const eventTS = new Date(evt.timestamp)
	const editEventTS = evt.last_edit ? new Date(evt.last_edit.timestamp) : null
	const wrapperClassNames = ["timeline-event"]
	if (evt.unread_type & UnreadType.Highlight) {
		wrapperClassNames.push("highlight")
	}
	if (evt.redacted_by) {
		wrapperClassNames.push("redacted-event")
	}
	if (evt.type === "m.room.member") {
		wrapperClassNames.push("membership-event")
	}
	if (BodyType === HiddenEvent) {
		wrapperClassNames.push("hidden-event")
	}
	if (evt.sender === client.userID) {
		wrapperClassNames.push("own-event")
	}
	let dateSeparator = null
	const prevEvtDate = prevEvt ? new Date(prevEvt.timestamp) : null
	if (prevEvtDate && (
		eventTS.getDate() !== prevEvtDate.getDate() ||
		eventTS.getMonth() !== prevEvtDate.getMonth() ||
		eventTS.getFullYear() !== prevEvtDate.getFullYear())) {
		dateSeparator = <div className="date-separator">
			<hr role="none"/>
			{dateFormatter.format(eventTS)}
			<hr role="none"/>
		</div>
	}
	let smallAvatar = false
	let renderAvatar = true
	let eventTimeOnly = false
	if (isSmallEvent(BodyType)) {
		wrapperClassNames.push("small-event")
		smallAvatar = true
		eventTimeOnly = true
	} else if (prevEvt?.sender === evt.sender &&
		prevEvt.timestamp + 15 * 60 * 1000 > evt.timestamp &&
		!isSmallEvent(getBodyType(prevEvt)) &&
		dateSeparator === null) {
		wrapperClassNames.push("same-sender")
		eventTimeOnly = true
		renderAvatar = false
	}
	const fullTime = fullTimeFormatter.format(eventTS)
	const shortTime = formatShortTime(eventTS)
	const editTime = editEventTS ? `Edited at ${fullTimeFormatter.format(editEventTS)}` : null
	const relatesTo = (evt.orig_content ?? evt.content)?.["m.relates_to"]
	const replyTo = relatesTo?.["m.in_reply_to"]?.event_id
	const mainEvent = <div
		data-event-id={evt.event_id}
		className={wrapperClassNames.join(" ")}
		onContextMenu={onContextMenu}
	>
		{!disableMenu && !isMobileDevice && <div
			className={`context-menu-container ${forceContextMenuOpen ? "force-open" : ""}`}
		>
			<EventHoverMenu evt={evt} setForceOpen={setForceContextMenuOpen}/>
		</div>}
		{renderAvatar && <div
			className="sender-avatar"
			title={evt.sender}
			data-target-panel="user"
			data-target-user={evt.sender}
			onClick={mainScreen.clickRightPanelOpener}
		>
			<img
				className={`${smallAvatar ? "small" : ""} avatar`}
				loading="lazy"
				src={getAvatarURL(evt.sender, memberEvtContent)}
				alt=""
			/>
		</div>}
		{!eventTimeOnly ? <div className="event-sender-and-time">
			<span
				className={`event-sender sender-color-${getUserColorIndex(evt.sender)}`}
				data-target-user={evt.sender}
				onClick={roomCtx.appendMentionToComposer}
			>
				{getDisplayname(evt.sender, memberEvtContent)}
			</span>
			<span className="event-time" title={fullTime}>{shortTime}</span>
			{(editEventTS && editTime) ? <span className="event-edited" title={editTime}>
				(edited at {formatShortTime(editEventTS)})
			</span> : null}
		</div> : <div className="event-time-only">
			<span className="event-time" title={editTime ? `${fullTime} - ${editTime}` : fullTime}>{shortTime}</span>
		</div>}
		<div className="event-content">
			{isEventID(replyTo) && BodyType !== HiddenEvent && !evt.redacted_by ? <ReplyIDBody
				room={roomCtx.store}
				eventID={replyTo}
				isThread={relatesTo?.rel_type === "m.thread"}
			/> : null}
			<ContentErrorBoundary>
				<BodyType room={roomCtx.store} sender={memberEvt} event={evt}/>
			</ContentErrorBoundary>
			{evt.reactions ? <EventReactions reactions={evt.reactions}/> : null}
		</div>
		{!evt.event_id.startsWith("~") && roomCtx.store.preferences.display_read_receipts &&
			<ReadReceipts room={roomCtx.store} eventID={evt.event_id} />}
		{evt.sender === client.userID && evt.transaction_id ? <EventSendStatus evt={evt}/> : null}
	</div>
	return <>
		{dateSeparator}
		{mainEvent}
	</>
}

export default React.memo(TimelineEvent)
