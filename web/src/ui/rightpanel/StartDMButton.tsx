// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import { use, useMemo, useState } from "react"
import Client from "@/api/client.ts"
import { UserID } from "@/api/types"
import MainScreenContext from "../MainScreenContext.ts"
import ChatIcon from "@/icons/chat.svg?react"

const StartDMButton = ({ userID, client }: { userID: UserID; client: Client }) => {
	const mainScreen = use(MainScreenContext)!
	const [isCreating, setIsCreating] = useState(false)

	const findExistingRoom = () => {
		for (const room of client.store.rooms.values()) {
			if (room.meta.current.dm_user_id === userID) {
				return room.roomID
			}
		}
	}
	const existingRoom = useMemo(findExistingRoom, [userID, client])

	const startDM = async () => {
		if (existingRoom) {
			mainScreen.setActiveRoom(existingRoom)
			return
		}
		if (!window.confirm(`Are you sure you want to start a chat with ${userID}?`)) {
			return
		}
		const existingRoomRelookup = findExistingRoom()
		if (existingRoomRelookup) {
			mainScreen.setActiveRoom(existingRoomRelookup)
			return
		}

		try {
			setIsCreating(true)

			let shouldEncrypt = false
			const initialState = []

			try {
				shouldEncrypt = (await client.rpc.trackUserDevices(userID)).devices.length > 0

				if (shouldEncrypt) {
					console.log("User has encryption devices, creating encrypted room")
					initialState.push({
						type: "m.room.encryption",
						content: {
							algorithm: "m.megolm.v1.aes-sha2",
						},
					})
				}
			} catch (err) {
				console.warn("Failed to check user encryption status:", err)
			}

			// Create the room with encryption if needed
			const response = await client.rpc.createRoom({
				is_direct: true,
				preset: "trusted_private_chat",
				invite: [userID],
				initial_state: initialState,
			})
			console.log("Created DM room:", response.room_id)

			// FIXME this is a hacky way to work around the room taking time to come down /sync
			setTimeout(() => {
				mainScreen.setActiveRoom(response.room_id)
			}, 1000)
		} catch (err) {
			console.error("Failed to create DM room:", err)
			window.alert(`Failed to create DM room: ${err}`)
		} finally {
			setIsCreating(false)
		}
	}

	return <button
		className="moderation-action positive"
		onClick={startDM}
		disabled={isCreating}
	>
		<ChatIcon />
		<span>{existingRoom ? "Go to DM" : isCreating ? "Creating..." : "Create DM"}</span>
	</button>
}

export default StartDMButton
