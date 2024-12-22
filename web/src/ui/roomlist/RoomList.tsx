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
import React, { use, useRef, useState } from "react"
import type { RoomID } from "@/api/types"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import reverseMap from "@/util/reversemap.ts"
import toSearchableString from "@/util/searchablestring.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { keyToString } from "../keybindings.ts"
import Entry from "./Entry.tsx"
import CloseIcon from "@/icons/close.svg?react"
import SearchIcon from "@/icons/search.svg?react"
import "./RoomList.css"

interface RoomListProps {
	activeRoomID: RoomID | null
}

const RoomList = ({ activeRoomID }: RoomListProps) => {
	const client = use(ClientContext)!
	const mainScreen = use(MainScreenContext)
	const roomList = useEventAsState(client.store.roomList)
	const roomFilterRef = useRef<HTMLInputElement>(null)
	const [roomFilter, setRoomFilter] = useState("")
	const [realRoomFilter, setRealRoomFilter] = useState("")

	const updateRoomFilter = (evt: React.ChangeEvent<HTMLInputElement>) => {
		setRoomFilter(evt.target.value)
		client.store.currentRoomListFilter = toSearchableString(evt.target.value)
		setRealRoomFilter(client.store.currentRoomListFilter)
	}
	const clearQuery = () => {
		setRoomFilter("")
		client.store.currentRoomListFilter = ""
		setRealRoomFilter("")
		roomFilterRef.current?.focus()
	}
	const onKeyDown = (evt: React.KeyboardEvent<HTMLInputElement>) => {
		const key = keyToString(evt)
		if (key === "Escape") {
			clearQuery()
			evt.stopPropagation()
			evt.preventDefault()
		} else if (key === "Enter") {
			const roomList = client.store.getFilteredRoomList()
			mainScreen.setActiveRoom(roomList[roomList.length-1]?.room_id)
			clearQuery()
			evt.stopPropagation()
			evt.preventDefault()
		}
	}

	return <div className="room-list-wrapper">
		<div className="room-search-wrapper">
			<input
				value={roomFilter}
				onChange={updateRoomFilter}
				onKeyDown={onKeyDown}
				className="room-search"
				type="text"
				placeholder="Search rooms"
				ref={roomFilterRef}
				id="room-search"
			/>
			<button onClick={clearQuery} disabled={roomFilter === ""}>
				{roomFilter !== "" ? <CloseIcon/> : <SearchIcon/>}
			</button>
		</div>
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
