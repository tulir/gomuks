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
import React, { useMemo } from "react"
import Client from "../api/client.ts"
import { DBEvent, RoomID } from "../api/types/hitypes.ts"
import { useNonNullEventAsState } from "../util/eventdispatcher.ts"
import { RoomListEntry } from "../api/statestore.ts"
import "./RoomList.css"

export interface RoomListProps {
	client: Client
	setActiveRoom: (room_id: RoomID) => void
}

const RoomList = ({ client, setActiveRoom }: RoomListProps) => {
	const roomList = useNonNullEventAsState(client.store.roomList)
	const clickRoom = useMemo(() => (evt: React.MouseEvent) => {
		const roomID = evt.currentTarget.getAttribute("data-room-id")
		if (roomID) {
			setActiveRoom(roomID)
		} else {
			console.warn("No room ID :(", evt.currentTarget)
		}
	}, [setActiveRoom])

	return <div className="room-list">
		{reverseMap(roomList, room =>
			<RoomEntry
				key={room.room_id}
				client={client}
				room={room}
				setActiveRoom={clickRoom}
			/>,
		)}
	</div>
}

function reverseMap<T, O>(arg: T[], fn: (a: T) => O) {
	return arg.map((_, i, arr) => fn(arr[arr.length - i - 1]))
}

export interface RoomListEntryProps {
	client: Client
	room: RoomListEntry
	setActiveRoom: (evt: React.MouseEvent) => void
}

function makePreviewText(evt?: DBEvent): string {
	if (!evt) {
		return ""
	}
	if (evt.type === "m.room.message") {
		// @ts-expect-error TODO add content types
		return evt.content.body
	} else if (evt.decrypted_type === "m.room.message") {
		// @ts-expect-error TODO add content types
		return evt.decrypted.body
	}
	return ""
}

const avatarRegex = /^mxc:\/\/([a-zA-Z0-9.:-]+)\/([a-zA-Z0-9_-]+)$/

const getAvatarURL = (avatar?: string): string | undefined => {
	if (!avatar) {
		return undefined
	}
	const match = avatar.match(avatarRegex)
	if (!match) {
		return undefined
	}
	return `_gomuks/media/${match[1]}/${match[2]}`
}

const RoomEntry = ({ room, setActiveRoom }: RoomListEntryProps) => {
	const previewText = makePreviewText(room.preview_event)
	return <div className="room-entry" onClick={setActiveRoom} data-room-id={room.room_id}>
		<div className="room-entry-left">
			<img className="room-avatar" src={getAvatarURL(room.avatar)} alt=""/>
		</div>
		<div className="room-entry-right">
			<div className="room-name">{room.name}</div>
			{previewText && <div className="message-preview" title={previewText}>{previewText}</div>}
		</div>
	</div>
}

export default RoomList
