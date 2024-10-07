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
import { RoomStateStore } from "../api/statestore.ts"
import { useNonNullEventAsState } from "../util/eventdispatcher.ts"
import "./RoomView.css"
import TimelineView from "./timeline/TimelineView.tsx"

interface RoomViewProps {
	room: RoomStateStore
}

const RoomView = ({ room }: RoomViewProps) => {
	const roomMeta = useNonNullEventAsState(room.meta)
	return <div className="room-view">
		{roomMeta.room_id}
		<TimelineView room={room} />
	</div>
}

export default RoomView
