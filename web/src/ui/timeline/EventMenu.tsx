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
import { CSSProperties, use, useCallback, useRef } from "react"
import { useRoomState } from "@/api/statestore"
import { MemDBEvent, PowerLevelEventContent } from "@/api/types"
import { useNonNullEventAsState } from "@/util/eventdispatcher.ts"
import { ClientContext } from "../ClientContext.ts"
import EmojiPicker from "../emojipicker/EmojiPicker.tsx"
import { ModalContext } from "../modal/Modal.tsx"
import { useRoomContext } from "../roomcontext.ts"
import EditIcon from "../../icons/edit.svg?react"
import MoreIcon from "../../icons/more.svg?react"
import PinIcon from "../../icons/pin.svg?react"
import ReactIcon from "../../icons/react.svg?react"
import ReplyIcon from "../../icons/reply.svg?react"
import UnpinIcon from "../../icons/unpin.svg?react"
import "./EventMenu.css"

interface EventHoverMenuProps {
	evt: MemDBEvent
	setForceOpen: (forceOpen: boolean) => void
}

const EventMenu = ({ evt, setForceOpen }: EventHoverMenuProps) => {
	const client = use(ClientContext)!
	const userID = client.userID
	const roomCtx = useRoomContext()
	const openModal = use(ModalContext)
	const contextMenuRef = useRef<HTMLDivElement>(null)
	const onClickReply = useCallback(() => roomCtx.setReplyTo(evt.event_id), [roomCtx, evt.event_id])
	const onClickPin = useCallback(() => {
		client.pinMessage(roomCtx.store, evt.event_id, true)
			.catch(err => window.alert(`Failed to pin message: ${err}`))
	}, [client, roomCtx, evt.event_id])
	const onClickUnpin = useCallback(() => {
		client.pinMessage(roomCtx.store, evt.event_id, false)
			.catch(err => window.alert(`Failed to unpin message: ${err}`))
	}, [client, roomCtx, evt.event_id])
	const onClickReact = useCallback(() => {
		const reactionButton = contextMenuRef.current?.getElementsByClassName("reaction-button")?.[0]
		if (!reactionButton) {
			return
		}
		const rect = reactionButton.getBoundingClientRect()
		const style: CSSProperties = { right: window.innerWidth - rect.right }
		const emojiPickerHeight = 30 * 16
		if (rect.bottom + emojiPickerHeight > window.innerHeight) {
			style.bottom = window.innerHeight - rect.top
		} else {
			style.top = rect.bottom
		}
		setForceOpen(true)
		openModal({
			content: <EmojiPicker
				style={style}
				onSelect={emoji => {
					const content: Record<string, unknown> = {
						"m.relates_to": {
							rel_type: "m.annotation",
							event_id: evt.event_id,
							key: emoji.u,
						},
					}
					if (emoji.u?.startsWith("mxc://") && emoji.n) {
						content["com.beeper.emoji.shortcode"] = emoji.n
					}
					client.sendEvent(evt.room_id, "m.reaction", content)
						.catch(err => window.alert(`Failed to send reaction: ${err}`))
				}}
				closeOnSelect={true}
				allowFreeform={true}
			/>,
			onClose: () => setForceOpen(false),
		})
	}, [client, evt, setForceOpen, openModal])
	const onClickEdit = useCallback(() => {
		roomCtx.setEditing(evt)
	}, [roomCtx, evt])
	const onClickMore = useCallback(() => {
		window.alert("Nothing here yet :(")
	}, [])
	const isEditing = useNonNullEventAsState(roomCtx.isEditing)
	const plEvent = useRoomState(roomCtx.store, "m.room.power_levels", "")
	// We get pins from getPinnedEvents, but use the hook anyway to subscribe to changes
	useRoomState(roomCtx.store, "m.room.pinned_events", "")
	const pls = (plEvent?.content ?? {}) as PowerLevelEventContent
	const pins = roomCtx.store.getPinnedEvents()
	const ownPL = pls.users?.[userID] ?? pls.users_default ?? 0
	const pinPL = pls.events?.["m.room.pinned_events"] ?? pls.state_default ?? 50
	return <div className="context-menu" ref={contextMenuRef}>
		<button className="reaction-button" onClick={onClickReact}><ReactIcon/></button>
		<button
			className="reply-button"
			disabled={isEditing}
			title={isEditing ? "Can't reply to messages while editing a message" : undefined}
			onClick={onClickReply}
		><ReplyIcon/></button>
		{evt.sender === userID && evt.type === "m.room.message" && evt.relation_type !== "m.replace"
			&& <button className="edit-button" onClick={onClickEdit}><EditIcon/></button>}
		{ownPL >= pinPL && (pins.includes(evt.event_id)
			? <button className="unpin-button" onClick={onClickUnpin}><UnpinIcon/></button>
			: <button className="pin-button" onClick={onClickPin}><PinIcon/></button>)}
		<button className="more-button" onClick={onClickMore}><MoreIcon/></button>
	</div>
}

export default EventMenu
