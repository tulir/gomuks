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
import Client from "../api/client.ts"
import { RoomStateStore } from "../api/statestore.ts"
import { useNonNullEventAsState } from "../util/eventdispatcher.ts"
import "./RoomView.css"
import TimelineEvent from "./timeline/TimelineEvent.tsx"

export interface RoomViewProps {
	client: Client
	room: RoomStateStore
}

const RoomView = ({ client, room }: RoomViewProps) => {
	const roomMeta = useNonNullEventAsState(room.meta)
	const timeline = useNonNullEventAsState(room.timeline)
	return <div className="room-view">
		{roomMeta.room_id}
		<button onClick={() => client.loadMoreHistory(roomMeta.room_id)}>Load history</button>
		{timeline.map(entry => <TimelineEvent
			key={entry.event_rowid} client={client} room={room} eventRowID={entry.event_rowid}
		/>)}
	</div>
}

export default RoomView
