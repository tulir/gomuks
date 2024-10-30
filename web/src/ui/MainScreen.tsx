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
import { use, useCallback, useLayoutEffect, useMemo, useState } from "react"
import type { RoomID } from "@/api/types"
import ClientContext from "./ClientContext.ts"
import MainScreenContext, { MainScreenContextFields } from "./MainScreenContext.ts"
import RoomView from "./RoomView.tsx"
import RightPanel, { RightPanelProps } from "./rightpanel/RightPanel.tsx"
import RoomList from "./roomlist/RoomList.tsx"
import { useResizeHandle } from "./useResizeHandle.tsx"
import "./MainScreen.css"

const MainScreen = () => {
	const [activeRoomID, setActiveRoomID] = useState<RoomID | null>(null)
	const [rightPanel, setRightPanel] = useState<RightPanelProps | null>(null)
	const client = use(ClientContext)!
	const activeRoom = activeRoomID && client.store.rooms.get(activeRoomID)
	const setActiveRoom = useCallback((roomID: RoomID) => {
		console.log("Switching to room", roomID)
		setActiveRoomID(roomID)
		setRightPanel(null)
		if (client.store.rooms.get(roomID)?.stateLoaded === false) {
			client.loadRoomState(roomID)
				.catch(err => console.error("Failed to load room state", err))
		}
	}, [client])
	const context: MainScreenContextFields = useMemo(() => ({
		setActiveRoom,
		clickRoom: (evt: React.MouseEvent) => {
			const roomID = evt.currentTarget.getAttribute("data-room-id")
			if (roomID) {
				setActiveRoom(roomID)
			} else {
				console.warn("No room ID :(", evt.currentTarget)
			}
		},
		clearActiveRoom: () => setActiveRoomID(null),
		setRightPanel,
	}), [setRightPanel, setActiveRoom])
	useLayoutEffect(() => {
		client.store.switchRoom = setActiveRoom
	}, [client, setActiveRoom])
	const [roomListWidth, resizeHandle1] = useResizeHandle(
		300, 48, 900, "roomListWidth", { className: "room-list-resizer" },
	)
	const [rightPanelWidth, resizeHandle2] = useResizeHandle(
		300, 100, 900, "rightPanelWidth", { className: "right-panel-resizer" },
	)
	const extraStyle = {
		["--room-list-width" as string]: `${roomListWidth}px`,
		["--right-panel-width" as string]: `${rightPanelWidth}px`,
	}
	return <main className={`matrix-main ${activeRoom ? "room-selected" : ""}`} style={extraStyle}>
		<MainScreenContext value={context}>
			<RoomList activeRoomID={activeRoomID}/>
			{resizeHandle1}
			{activeRoom && <RoomView key={activeRoomID} room={activeRoom}/>}
			{resizeHandle2}
			{rightPanel && <RightPanel {...rightPanel}/>}
		</MainScreenContext>
	</main>
}

export default MainScreen
