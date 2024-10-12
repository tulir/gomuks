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
import React, { use, useState } from "react"
import { getMediaURL } from "../api/media.ts"
import { RoomStateStore } from "../api/statestore.ts"
import { useNonNullEventAsState } from "../util/eventdispatcher.ts"
import { ClientContext } from "./ClientContext.ts"
import TimelineView from "./timeline/TimelineView.tsx"
import "./RoomView.css"

interface RoomViewProps {
	room: RoomStateStore
}

const RoomView = ({ room }: RoomViewProps) => {
	const [text, setText] = useState("")
	const client = use(ClientContext)!
	const roomMeta = useNonNullEventAsState(room.meta)
	const sendMessage = (evt: React.FormEvent) => {
		evt.preventDefault()
		setText("")
		client.rpc.sendMessage(room.roomID, text)
			.catch(err => window.alert("Failed to send message: " + err))
	}
	return <div className="room-view">
		<div className="room-header">
			<img
				className="avatar"
				loading="lazy"
				src={getMediaURL(roomMeta.avatar)}
				alt=""
			/>
			<span className="room-name">
				{roomMeta.name ?? roomMeta.room_id}
			</span>
		</div>
		<TimelineView room={room}/>
		<form className="message-composer" onSubmit={sendMessage}>
			<input
				autoFocus
				type="text"
				value={text}
				onChange={evt => setText(evt.target.value)}
				placeholder="Send a message"
			/>
			<button type="submit">Send</button>
		</form>
	</div>
}

export default RoomView
