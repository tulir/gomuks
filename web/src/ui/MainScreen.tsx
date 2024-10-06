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
import { useState } from "react"
import type Client from "../api/client.ts"
import type { RoomID } from "../api/types/hitypes.ts"
import RoomList from "./RoomList.tsx"
import RoomView from "./RoomView.tsx"
import "./MainScreen.css"

export interface MainScreenProps {
	client: Client
}

const MainScreen = ({ client }: MainScreenProps) => {
	const [activeRoomID, setActiveRoomID] = useState<RoomID | null>(null)
	const activeRoom = activeRoomID && client.store.rooms.get(activeRoomID)
	return <main className="matrix-main">
		<RoomList client={client} setActiveRoom={setActiveRoomID} />
		{activeRoom && <RoomView client={client} room={activeRoom} />}
	</main>
}

export default MainScreen
