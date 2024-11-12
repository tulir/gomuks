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
import { MemDBEvent, PowerLevelEventContent } from "@/api/types"
import { emojiToReactionContent } from "@/util/emoji"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import ClientContext from "../../ClientContext.ts"
import EmojiPicker from "../../emojipicker/EmojiPicker.tsx"
import { ModalContext } from "../../modal/Modal.tsx"
import { useRoomContext } from "../../roomview/roomcontext.ts"
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
	const isPending = evt.event_id.startsWith("~")
	const pendingTitle = isPending ? "Can't action messages that haven't been sent yet" : undefined
	// TODO should these subscribe to the store?
	const plEvent = roomCtx.store.getStateEvent("m.room.power_levels", "")
	const encryptionEvent = roomCtx.store.getStateEvent("m.room.encryption", "")
	const isEncrypted = encryptionEvent?.content?.algorithm === "m.megolm.v1.aes-sha2"
	const pls = (plEvent?.content ?? {}) as PowerLevelEventContent
	const ownPL = pls.users?.[userID] ?? pls.users_default ?? 0
	const reactPL = pls.events?.["m.reaction"] ?? pls.events_default ?? 0
	const evtSendType = isEncrypted ? "m.room.encrypted" : evt.type === "m.sticker" ? "m.sticker" : "m.room.message"
	const messageSendPL = pls.events?.[evtSendType] ?? pls.events_default ?? 0

	const canSend = ownPL >= messageSendPL
	const canEdit = canSend
		&& evt.sender === userID
		&& evt.type === "m.room.message"
		&& evt.relation_type !== "m.replace"
		&& !evt.redacted_by
	const canReact = ownPL >= reactPL

	return <div className="event-hover-menu" ref={contextMenuRef}>
		{canReact && <button disabled={isPending} title={pendingTitle} onClick={onClickReact}><ReactIcon/></button>}
		{canSend && <button
			disabled={isEditing || isPending}
			title={isEditing ? "Can't reply to messages while editing a message" : pendingTitle}
			onClick={onClickReply}
		><ReplyIcon/></button>}
		{canEdit && <button onClick={onClickEdit} disabled={isPending} title={pendingTitle}><EditIcon/></button>}
		<button onClick={onClickMore}><MoreIcon/></button>
	</div>
}

export default EventMenu
