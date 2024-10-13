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
import { getMediaURL } from "../api/media.ts"
import { RoomStateStore } from "../api/statestore.ts"
import { useNonNullEventAsState } from "../util/eventdispatcher.ts"
import MessageComposer from "./MessageComposer.tsx"
import TimelineView from "./timeline/TimelineView.tsx"
import "./RoomView.css"

interface RoomViewProps {
	room: RoomStateStore
}

const RoomHeader = ({ room }: RoomViewProps) => {
	const roomMeta = useNonNullEventAsState(room.meta)
	return <div className="room-header">
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
}

const onKeyDownRoomView = (evt: React.KeyboardEvent) => {
	if (evt.target === evt.currentTarget) {
		document.getElementById("message-composer")?.focus()
	}
}

const RoomView = ({ room }: RoomViewProps) => {
	return <div className="room-view" onKeyDown={onKeyDownRoomView} tabIndex={-1}>
		<RoomHeader room={room}/>
		<TimelineView room={room}/>
		<MessageComposer room={room}/>
	</div>
}

export default RoomView
