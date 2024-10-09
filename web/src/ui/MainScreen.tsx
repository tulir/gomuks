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
import { use, useState } from "react"
import type { RoomID } from "../api/types"
import { ClientContext } from "./ClientContext.ts"
import RoomView from "./RoomView.tsx"
import RoomList from "./roomlist/RoomList.tsx"
import "./MainScreen.css"

const MainScreen = () => {
	const [activeRoomID, setActiveRoomID] = useState<RoomID | null>(null)
	const activeRoom = activeRoomID && use(ClientContext)!.store.rooms.get(activeRoomID)
	return <main className="matrix-main">
		<RoomList setActiveRoom={setActiveRoomID} />
		{activeRoom && <RoomView key={activeRoomID} room={activeRoom} />}
	</main>
}

export default MainScreen
