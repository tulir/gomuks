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
import toSearchableString from "@/util/searchablestring.ts"
import ClientContext from "../ClientContext.ts"
import Entry from "./Entry.tsx"
import "./RoomList.css"

interface RoomListProps {
	setActiveRoom: (room_id: RoomID) => void
	activeRoomID: RoomID | null
}

const RoomList = ({ setActiveRoom, activeRoomID }: RoomListProps) => {
	const roomList = useEventAsState(use(ClientContext)!.store.roomList)
	const roomFilterRef = useRef<HTMLInputElement>(null)
	const [roomFilter, setRoomFilter] = useState("")
	const [realRoomFilter, setRealRoomFilter] = useState("")
	const clickRoom = useCallback((evt: React.MouseEvent) => {
		const roomID = evt.currentTarget.getAttribute("data-room-id")
		if (roomID) {
			setActiveRoom(roomID)
		} else {
			console.warn("No room ID :(", evt.currentTarget)
		}
	}, [setActiveRoom])

	const updateRoomFilter = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setRoomFilter(evt.target.value)
		setRealRoomFilter(toSearchableString(evt.target.value))
	}, [])

	return <div className="room-list-wrapper">
		<input
			value={roomFilter}
			onChange={updateRoomFilter}
			className="room-search"
			type="text"
			placeholder="Search rooms"
			ref={roomFilterRef}
		/>
		<div className="room-list">
			{reverseMap(roomList, room =>
				<Entry
					key={room.room_id}
					isActive={room.room_id === activeRoomID}
					hidden={roomFilter ? !room.search_name.includes(realRoomFilter) : false}
					room={room}
					setActiveRoom={clickRoom}
				/>,
			)}
		</div>
	</div>
}

function reverseMap<T, O>(arg: T[], fn: (a: T) => O) {
	return arg.map((_, i, arr) => fn(arr[arr.length - i - 1]))
}

export default RoomList
