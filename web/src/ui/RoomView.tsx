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
import { use, useRef } from "react"
import { getMediaURL } from "@/api/media.ts"
import { RoomStateStore } from "@/api/statestore"
import { EventID } from "@/api/types"
import { useNonNullEventAsState } from "@/util/eventdispatcher.ts"
import { LightboxContext } from "./Lightbox.tsx"
import MessageComposer from "./MessageComposer.tsx"
import TimelineView from "./timeline/TimelineView.tsx"
import BackIcon from "@/icons/back.svg?react"
import "./RoomView.css"

interface RoomViewProps {
	room: RoomStateStore
	clearActiveRoom: () => void
}

const RoomHeader = ({ room, clearActiveRoom }: RoomViewProps) => {
	const roomMeta = useNonNullEventAsState(room.meta)
	return <div className="room-header">
		<button className="back" onClick={clearActiveRoom}><BackIcon/></button>
		<img
			className="avatar"
			loading="lazy"
			src={getMediaURL(roomMeta.avatar)}
			onClick={use(LightboxContext)!}
			alt=""
		/>
		<span className="room-name">
			{roomMeta.name ?? roomMeta.room_id}
		</span>
	</div>
}

const onKeyDownRoomView = (evt: React.KeyboardEvent) => {
	if (evt.target === evt.currentTarget && (!evt.ctrlKey || evt.key === "v") && !evt.altKey) {
		document.getElementById("message-composer")?.focus()
	}
}

const RoomView = ({ room, clearActiveRoom }: RoomViewProps) => {
	const scrollToBottomRef = useRef<() => void>(() => {})
	const setReplyToRef = useRef<(evt: EventID | null) => void>(() => {})
	return <div className="room-view" onKeyDown={onKeyDownRoomView} tabIndex={-1}>
		<RoomHeader room={room} clearActiveRoom={clearActiveRoom}/>
		<TimelineView room={room} scrollToBottomRef={scrollToBottomRef} setReplyToRef={setReplyToRef}/>
		<MessageComposer room={room} scrollToBottomRef={scrollToBottomRef} setReplyToRef={setReplyToRef}/>
	</div>
}

export default RoomView
