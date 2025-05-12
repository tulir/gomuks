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
import React, { JSX, use, useCallback, useState } from "react"
import { createPortal } from "react-dom"
import { getAvatarThumbnailURL, getMediaURL, getUserColorIndex } from "@/api/media.ts"
import {
	RoomStateStore,
	applyPerMessageSender,
	maybeRedactMemberEvent,
	usePreference,
	useRoomMember,
} from "@/api/statestore"
import { MemDBEvent, URLPreview as URLPreviewType, UnreadType } from "@/api/types"
import { isMobileDevice } from "@/util/ismobile.ts"
import { getDisplayname, isEventID } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { EventFixedMenu, EventFullMenu, EventHoverMenu, getModalStyleFromMouse } from "../menu"
import { ModalContext, NestableModalContext } from "../modal"
import { useRoomContext } from "../roomview/roomcontext.ts"
import URLPreview from "../urlpreview/URLPreview.tsx"
import EventEditHistory from "./EventEditHistory.tsx"
import ReadReceipts from "./ReadReceipts.tsx"
import { ReplyIDBody } from "./ReplyBody.tsx"
import { ContentErrorBoundary, HiddenEvent, getBodyType, getPerMessageProfile, isSmallEvent } from "./content"
import ErrorIcon from "@/icons/error.svg?react"
import PendingIcon from "@/icons/pending.svg?react"
import SentIcon from "@/icons/sent.svg?react"
import "./TimelineEvent.css"

export interface TimelineEventProps {
	evt: MemDBEvent
	prevEvt: MemDBEvent | null
	disableMenu?: boolean
	smallReplies?: boolean
	isFocused?: boolean
	editHistoryView?: boolean
}

const fullTimeFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full", timeStyle: "medium" })
const dateFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full" })
const formatShortTime = (time: Date) =>
	`${time.getHours().toString().padStart(2, "0")}:${time.getMinutes().toString().padStart(2, "0")}`

interface EventReactionsProps {
	reactions: Record<string, number>
	onRereact: (mouseEvt: React.MouseEvent) => void
}

const EventReactions = ({ reactions, onRereact }: EventReactionsProps) => {
	const reactionEntries = Object.entries(reactions).filter(([, count]) => count > 0).sort((a, b) => b[1] - a[1])
	if (reactionEntries.length === 0) {
		return null
	}
	return <div className="event-reactions">
		{reactionEntries.map(([reaction, count]) =>
			<div key={reaction} className="reaction" title={reaction} onClick={onRereact}>
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

const EventURLPreviews = ({ event, room }: {
	room: RoomStateStore
	event: MemDBEvent
}) => {
	const client = use(ClientContext)!
	const renderPreviews = usePreference(client.store, room, "render_url_previews")
	if (event.redacted_by || !renderPreviews) {
		return null
	}

	const previews = (event.content["com.beeper.linkpreviews"] ?? event.content["m.url_previews"]) as URLPreviewType[]
	if (!previews) {
		return null
	}
	return <div className="url-previews">
		{previews.map((p, i) => <URLPreview key={i} url={p.matched_url} preview={p}/>)}
	</div>
}

const TimelineEvent = ({
	evt, prevEvt, disableMenu, smallReplies, isFocused, editHistoryView,
}: TimelineEventProps) => {
	const roomCtx = useRoomContext()
	const client = use(ClientContext)!
	const mainScreen = use(MainScreenContext)
	const openModal = use(ModalContext)
	const openNestableModal = use(NestableModalContext)
	const [forceContextMenuOpen, setForceContextMenuOpen] = useState(false)
	const onContextMenu = (mouseEvt: React.MouseEvent) => {
		const targetElem = mouseEvt.target as HTMLElement
		if (
			!roomCtx.store.preferences.message_context_menu
			|| targetElem.tagName === "A"
			|| targetElem.tagName === "IMG"
			|| targetElem.tagName === "VIDEO"
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
	}
	const onRereact = useCallback((mouseEvt: React.MouseEvent) => {
		client.sendEvent(evt.room_id, "m.reaction", {
			"m.relates_to": {
				rel_type: "m.annotation",
				event_id: evt.event_id,
				key: mouseEvt.currentTarget.getAttribute("title"),
			},
		}).catch(err => {
			console.error("Failed to send reaction", err)
			window.alert(`Failed to send reaction: ${err}`)
		})
	}, [client, evt])
	const onClick = (mouseEvt: React.MouseEvent) => {
		const targetElem = mouseEvt.target as HTMLElement
		if (
			targetElem.tagName === "A"
			|| targetElem.tagName === "IMG"
			|| targetElem.tagName === "VIDEO"
			|| targetElem.tagName === "SUMMARY"
		) {
			return
		}
		mouseEvt.preventDefault()
		mouseEvt.stopPropagation()
		roomCtx.setFocusedEventRowID(roomCtx.focusedEventRowID === evt.rowid ? null : evt.rowid)
	}
	const openEditHistory = () => {
		openNestableModal({
			content: <EventEditHistory evt={evt} roomCtx={roomCtx}/>,
			dimmed: true,
			boxed: true,
			boxClass: "full-screen-mobile event-edit-history-wrapper",
			innerBoxClass: "event-edit-history-modal",
		})
	}
	const BodyType = getBodyType(evt)
	const eventTS = new Date(evt.timestamp)
	const editEventTS = evt.last_edit ? new Date(evt.last_edit.timestamp) : null
	const wrapperClassNames = ["timeline-event"]
	if (evt.unread_type & UnreadType.Highlight) {
		wrapperClassNames.push("highlight")
	}
	const isRedacted = evt.redacted_by && !evt.viewing_redacted
	if (isRedacted) {
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
	if ((isMobileDevice && !editHistoryView) || disableMenu) {
		wrapperClassNames.push("no-hover")
	}
	if (isFocused) {
		wrapperClassNames.push("focused-event")
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
	const isSmallBodyType = isSmallEvent(BodyType)
	const relatesTo = (evt.orig_content ?? evt.content)?.["m.relates_to"]
	const replyTo = relatesTo?.["m.in_reply_to"]?.event_id
	let replyAboveMessage: JSX.Element | null = null
	let replyInMessage: JSX.Element | null = null
	if (isEventID(replyTo) && BodyType !== HiddenEvent && !isRedacted && !editHistoryView) {
		const replyElem = <ReplyIDBody
			room={roomCtx.store}
			eventID={replyTo}
			isThread={relatesTo?.rel_type === "m.thread"}
			small={!!smallReplies}
		/>
		if (smallReplies && !isSmallBodyType) {
			replyAboveMessage = replyElem
			wrapperClassNames.push("reply-above")
		} else {
			replyInMessage = replyElem
		}
	}
	const perMessageSender = getPerMessageProfile(evt)
	const prevPerMessageSender = getPerMessageProfile(prevEvt)
	const memberEvt = useRoomMember(client, roomCtx.store, evt.sender)
	const memberEvtContent = maybeRedactMemberEvent(memberEvt)
	const renderMemberEvtContent = applyPerMessageSender(memberEvtContent, perMessageSender)

	let smallAvatar = false
	let renderAvatar = true
	let eventTimeOnly = false
	if (isSmallBodyType) {
		wrapperClassNames.push("small-event")
		smallAvatar = true
		eventTimeOnly = true
	} else if (
		prevEvt?.sender === evt.sender
		&& prevEvt.timestamp + 15 * 60 * 1000 > evt.timestamp
		&& dateSeparator === null
		&& !replyAboveMessage
		&& !isSmallEvent(getBodyType(prevEvt))
		&& prevPerMessageSender?.id === perMessageSender?.id
	) {
		wrapperClassNames.push("same-sender")
		eventTimeOnly = true
		renderAvatar = false
	}
	if (editHistoryView) {
		wrapperClassNames.push("edit-history-event")
	}
	const fullTime = fullTimeFormatter.format(eventTS)
	const shortTime = formatShortTime(eventTS)
	const mainEvent = <div
		data-event-id={evt.event_id}
		className={wrapperClassNames.join(" ")}
		onContextMenu={onContextMenu}
		onClick={!disableMenu && !editHistoryView && isMobileDevice ? onClick : undefined}
	>
		{!disableMenu && (!isMobileDevice || editHistoryView) && <div
			className={`context-menu-container ${forceContextMenuOpen ? "force-open" : ""}`}
		>
			<EventHoverMenu evt={evt} roomCtx={roomCtx} setForceOpen={setForceContextMenuOpen}/>
		</div>}
		{isMobileDevice && isFocused && createPortal(
			<EventFixedMenu evt={evt} roomCtx={roomCtx} />,
			document.getElementById("mobile-event-menu-container")!,
		)}
		{replyAboveMessage}
		{renderAvatar && <div
			className="sender-avatar"
			title={perMessageSender ? `${perMessageSender.id} via ${evt.sender}` : evt.sender}
			data-target-panel="user"
			data-target-user={evt.sender}
			onClick={mainScreen.clickRightPanelOpener}
		>
			<img
				className={`${smallAvatar ? "small" : ""} avatar`}
				loading="lazy"
				src={getAvatarThumbnailURL(perMessageSender?.id ?? evt.sender, renderMemberEvtContent)}
				alt=""
			/>
		</div>}
		{!eventTimeOnly ? <div className="event-sender-and-time">
			<span
				className={`event-sender sender-color-${getUserColorIndex(perMessageSender?.id ?? evt.sender)}`}
				data-target-user={evt.sender}
				onClick={perMessageSender ? undefined : roomCtx.appendMentionToComposer}
				title={perMessageSender ? perMessageSender.id : evt.sender}
			>
				{getDisplayname(evt.sender, renderMemberEvtContent)}
			</span>
			{perMessageSender && <div className="per-message-event-sender">
				<span className="via">via</span>
				<span
					className={`event-sender sender-color-${getUserColorIndex(evt.sender)}`}
					data-target-user={evt.sender}
					onClick={roomCtx.appendMentionToComposer}
					title={evt.sender}
				>
					{getDisplayname(evt.sender, memberEvtContent)}
				</span>
			</div>}
			<span className="event-time" title={fullTime}>{shortTime}</span>
		</div> : <div className="event-time-only">
			<span className="event-time" title={fullTime}>{shortTime}</span>
		</div>}
		<div className="event-content">
			{replyInMessage}
			<ContentErrorBoundary>
				<BodyType room={roomCtx.store} sender={memberEvt} event={evt}/>
				{!isSmallBodyType && <EventURLPreviews room={roomCtx.store} event={evt}/>}
			</ContentErrorBoundary>
			{(!editHistoryView && editEventTS) ? <div
				className="event-edited"
				title={`Edited at ${fullTimeFormatter.format(editEventTS)}`}
				onClick={openEditHistory}
			>
				(edited at {formatShortTime(editEventTS)})
			</div> : null}
			{evt.reactions ? <EventReactions reactions={evt.reactions} onRereact={onRereact} /> : null}
		</div>
		{!evt.event_id.startsWith("~") && roomCtx.store.preferences.display_read_receipts && !editHistoryView &&
			<ReadReceipts room={roomCtx.store} eventID={evt.event_id} />}
		{evt.sender === client.userID && evt.transaction_id && !editHistoryView ? <EventSendStatus evt={evt}/> : null}
	</div>
	return <>
		{dateSeparator}
		{mainEvent}
	</>
}

export default React.memo(TimelineEvent)
