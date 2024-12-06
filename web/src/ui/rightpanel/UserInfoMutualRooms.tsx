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
import { useEffect, useReducer, useState } from "react"
import { ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { RoomListEntry } from "@/api/statestore"
import { UserID } from "@/api/types"
import ListEntry from "../roomlist/Entry.tsx"
import UserInfoError from "./UserInfoError.tsx"

interface MutualRoomsProps {
	client: Client
	userID: UserID
}

const MutualRooms = ({ client, userID }: MutualRoomsProps) => {
	const [rooms, setRooms] = useState<RoomListEntry[] | null>(null)
	const [errors, setErrors] = useState<string[] | null>(null)
	useEffect(() => {
		setRooms(null)
		setErrors(null)
		client.rpc.getMutualRooms(userID).then(
			rooms => setRooms(rooms.map((roomID): RoomListEntry | null => {
				const roomData = client.store.rooms.get(roomID)
				if (!roomData || roomData.hidden) {
					return null
				}
				return {
					room_id: roomID,
					dm_user_id: roomData.meta.current.lazy_load_summary?.heroes?.length === 1
						? roomData.meta.current.lazy_load_summary.heroes[0] : undefined,
					name: roomData.meta.current.name ?? "Unnamed room",
					avatar: roomData.meta.current.avatar,
					search_name: "",
					sorting_timestamp: 0,
					unread_messages: 0,
					unread_notifications: 0,
					unread_highlights: 0,
					marked_unread: false,
				}
			}).filter((data): data is RoomListEntry => !!data)),
			err => setErrors([`${err}`]),
		)
	}, [client, userID])
	const [maxCount, increaseMaxCount] = useReducer(count => count + 10, 3)
	if (!rooms) {
		return <div className="mutual-rooms">
			<h4>Shared rooms</h4>
			{rooms === undefined && <ScaleLoader className="user-info-loader" color="var(--primary-color)"/>}
			<UserInfoError errors={errors}/>
		</div>
	}
	return <div className="mutual-rooms">
		<h4>Shared rooms</h4>
		{rooms.slice(0, maxCount).map(room => <div key={room.room_id}>
			<ListEntry room={room} isActive={false} hidden={false}/>
		</div>)}
		{rooms.length > maxCount && <button className="show-more" onClick={increaseMaxCount}>
			Show {rooms.length - maxCount} more
		</button>}
		<UserInfoError errors={errors}/>
	</div>
}

export default MutualRooms
