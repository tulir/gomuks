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
import { MemDBEvent } from "@/api/types"
import { emojiToReactionContent } from "@/util/emoji"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import ClientContext from "../../ClientContext.ts"
import EmojiPicker from "../../emojipicker/EmojiPicker.tsx"
import { ModalContext } from "../../modal/Modal.tsx"
import { useRoomContext } from "../../roomcontext.ts"
import EventExtraMenu from "./EventExtraMenu.tsx"
import EditIcon from "@/icons/edit.svg?react"
import MoreIcon from "@/icons/more.svg?react"
import ReactIcon from "@/icons/react.svg?react"
import ReplyIcon from "@/icons/reply.svg?react"
import "./index.css"

interface EventHoverMenuProps {
	evt: MemDBEvent
	setForceOpen: (forceOpen: boolean) => void
}

function getModalStyle(button: HTMLButtonElement, modalHeight: number): CSSProperties {
	const rect = button.getBoundingClientRect()
	const style: CSSProperties = { right: window.innerWidth - rect.right }
	if (rect.bottom + modalHeight > window.innerHeight) {
		style.bottom = window.innerHeight - rect.top
	} else {
		style.top = rect.bottom
	}
	return style
}

const EventMenu = ({ evt, setForceOpen }: EventHoverMenuProps) => {
	const client = use(ClientContext)!
	const userID = client.userID
	const roomCtx = useRoomContext()
	const openModal = use(ModalContext)
	const contextMenuRef = useRef<HTMLDivElement>(null)
	const onClickReply = useCallback(() => roomCtx.setReplyTo(evt.event_id), [roomCtx, evt.event_id])
	const onClickReact = useCallback((mevt: React.MouseEvent<HTMLButtonElement>) => {
		const emojiPickerHeight = 34 * 16
		setForceOpen(true)
		openModal({
			content: <EmojiPicker
				style={getModalStyle(mevt.currentTarget, emojiPickerHeight)}
				onSelect={emoji => {
					client.sendEvent(evt.room_id, "m.reaction", emojiToReactionContent(emoji, evt.event_id))
						.catch(err => window.alert(`Failed to send reaction: ${err}`))
				}}
				room={roomCtx.store}
				closeOnSelect={true}
				allowFreeform={true}
			/>,
			onClose: () => setForceOpen(false),
		})
	}, [client, roomCtx, evt, setForceOpen, openModal])
	const onClickEdit = useCallback(() => {
		roomCtx.setEditing(evt)
	}, [roomCtx, evt])
	const onClickMore = useCallback((mevt: React.MouseEvent<HTMLButtonElement>) => {
		const moreMenuHeight = 10 * 16
		setForceOpen(true)
		openModal({
			content: <EventExtraMenu
				evt={evt}
				room={roomCtx.store}
				style={getModalStyle(mevt.currentTarget, moreMenuHeight)}
			/>,
			onClose: () => setForceOpen(false),
		})
	}, [evt, roomCtx, setForceOpen, openModal])
	const isEditing = useEventAsState(roomCtx.isEditing)
	return <div className="event-hover-menu" ref={contextMenuRef}>
		<button onClick={onClickReact}><ReactIcon/></button>
		<button
			disabled={isEditing}
			title={isEditing ? "Can't reply to messages while editing a message" : undefined}
			onClick={onClickReply}
		><ReplyIcon/></button>
		{evt.sender === userID && evt.type === "m.room.message" && evt.relation_type !== "m.replace" && !evt.redacted_by
			&& <button onClick={onClickEdit}><EditIcon/></button>}
		<button onClick={onClickMore}><MoreIcon/></button>
	</div>
}

export default EventMenu
