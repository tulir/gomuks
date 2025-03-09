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
import React, { JSX, memo, use } from "react"
import { getRoomAvatarThumbnailURL } from "@/api/media.ts"
import type { RoomListEntry } from "@/api/statestore"
import type { MemDBEvent, MemberEventContent } from "@/api/types"
import useContentVisibility from "@/util/contentvisibility.ts"
import { getDisplayname } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { RoomMenu, getModalStyleFromMouse } from "../menu"
import { ModalContext } from "../modal"
import UnreadCount from "./UnreadCount.tsx"

export interface RoomListEntryProps {
	room: RoomListEntry
	isActive: boolean
	hidden: boolean
	hideAvatar?: boolean
}

function getPreviewText(evt?: MemDBEvent, senderMemberEvt?: MemDBEvent | null): [string, JSX.Element | null] {
	if (!evt) {
		return ["", null]
	}
	if ((evt.type === "m.room.message" || evt.type === "m.sticker") && typeof evt.content.body === "string") {
		// eslint-disable-next-line react-hooks/rules-of-hooks
		const client = use(ClientContext)!
		const displayname = evt.sender === client.userID
			? "You"
			: getDisplayname(evt.sender, senderMemberEvt?.content as MemberEventContent)
		let previewText = evt.content.body
		if (evt.content.formatted_body?.includes?.("data-mx-spoiler")) {
			previewText = "<message contains spoilers>"
		}
		return [
			`${displayname}: ${evt.content.body}`,
			<>
				<span style={{ unicodeBidi: "isolate" }}>
					{displayname.length > 16 ? displayname.slice(0, 12) + "…" : displayname}
				</span>: {previewText}
			</>,
		]
	}
	return ["", null]
}

function renderEntry(room: RoomListEntry, hideAvatar: boolean | undefined) {
	const [previewText, croppedPreviewText] = getPreviewText(room.preview_event, room.preview_sender)

	return <>
		<div className="room-entry-left">
			<img
				loading="lazy"
				className="avatar room-avatar"
				src={getRoomAvatarThumbnailURL(room, undefined, hideAvatar)}
				alt=""
			/>
		</div>
		<div className="room-entry-right">
			<div className="room-name">{room.name}</div>
			{previewText && <div className="message-preview" title={previewText}>{croppedPreviewText}</div>}
		</div>
		<UnreadCount counts={room} />
	</>
}

const Entry = ({ room, isActive, hidden, hideAvatar }: RoomListEntryProps) => {
	const [isVisible, divRef] = useContentVisibility<HTMLDivElement>()
	const openModal = use(ModalContext)
	const mainScreen = use(MainScreenContext)
	const client = use(ClientContext)!
	const onContextMenu = (evt: React.MouseEvent<HTMLDivElement>) => {
		const realRoom = client.store.rooms.get(room.room_id)
		if (!realRoom) {
			// TODO implement separate menu for invite rooms
			console.error("Room state store not found for", room.room_id)
			return
		}
		openModal({
			content: <RoomMenu
				room={realRoom}
				entry={room}
				style={getModalStyleFromMouse(evt, 6 * 40)}
			/>,
		})
		evt.preventDefault()
	}
	return <div
		ref={divRef}
		className={`room-entry ${isActive ? "active" : ""} ${hidden ? "hidden" : ""}`}
		onClick={mainScreen.clickRoom}
		onContextMenu={onContextMenu}
		data-room-id={room.room_id}
	>
		{isVisible ? renderEntry(room, hideAvatar) : null}
	</div>
}

export default memo(Entry)
