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
import React, { use, useMemo } from "react"
import type { RoomID } from "../../api/types"
import { useNonNullEventAsState } from "../../util/eventdispatcher.ts"
import { ClientContext } from "../ClientContext.ts"
import Entry from "./Entry.tsx"
import "./RoomList.css"

interface RoomListProps {
	setActiveRoom: (room_id: RoomID) => void
}

const RoomList = ({ setActiveRoom }: RoomListProps) => {
	const roomList = useNonNullEventAsState(use(ClientContext)!.store.roomList)
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
			<Entry key={room.room_id} room={room} setActiveRoom={clickRoom}/>,
		)}
	</div>
}

function reverseMap<T, O>(arg: T[], fn: (a: T) => O) {
	return arg.map((_, i, arr) => fn(arr[arr.length - i - 1]))
}

export default RoomList
