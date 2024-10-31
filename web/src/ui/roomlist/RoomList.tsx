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
import React, { use, useCallback, useRef, useState } from "react"
import type { RoomID } from "@/api/types"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import reverseMap from "@/util/reversemap.ts"
import toSearchableString from "@/util/searchablestring.ts"
import ClientContext from "../ClientContext.ts"
import Entry from "./Entry.tsx"
import "./RoomList.css"

interface RoomListProps {
	activeRoomID: RoomID | null
}

const RoomList = ({ activeRoomID }: RoomListProps) => {
	const client = use(ClientContext)!
	const roomList = useEventAsState(client.store.roomList)
	const roomFilterRef = useRef<HTMLInputElement>(null)
	const [roomFilter, setRoomFilter] = useState("")
	const [realRoomFilter, setRealRoomFilter] = useState("")

	const updateRoomFilter = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setRoomFilter(evt.target.value)
		client.store.currentRoomListFilter = toSearchableString(evt.target.value)
		setRealRoomFilter(client.store.currentRoomListFilter)
	}, [client])

	return <div className="room-list-wrapper">
		<input
			value={roomFilter}
			onChange={updateRoomFilter}
			className="room-search"
			type="text"
			placeholder="Search rooms"
			ref={roomFilterRef}
			id="room-search"
		/>
		<div className="room-list">
			{reverseMap(roomList, room =>
				<Entry
					key={room.room_id}
					isActive={room.room_id === activeRoomID}
					hidden={roomFilter ? !room.search_name.includes(realRoomFilter) : false}
					room={room}
				/>,
			)}
		</div>
	</div>
}

export default RoomList
