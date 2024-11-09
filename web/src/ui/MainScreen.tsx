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
import { use, useEffect, useInsertionEffect, useMemo, useReducer, useState } from "react"
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import type { RoomID } from "@/api/types"
import ClientContext from "./ClientContext.ts"
import MainScreenContext, { MainScreenContextFields } from "./MainScreenContext.ts"
import Keybindings from "./keybindings.ts"
import RightPanel, { RightPanelProps } from "./rightpanel/RightPanel.tsx"
import RoomList from "./roomlist/RoomList.tsx"
import RoomView from "./roomview/RoomView.tsx"
import { useResizeHandle } from "./util/useResizeHandle.tsx"
import "./MainScreen.css"

const rpReducer = (prevState: RightPanelProps | null, newState: RightPanelProps | null) => {
	if (prevState?.type === newState?.type) {
		return null
	}
	return newState
}

class ContextFields implements MainScreenContextFields {
	public keybindings: Keybindings

	constructor(
		public setRightPanel: (props: RightPanelProps | null) => void,
		private directSetActiveRoom: (room: RoomStateStore | null) => void,
		private client: Client,
	) {
		this.keybindings = new Keybindings(client.store, this)
		client.store.switchRoom = this.setActiveRoom
		window.mainScreenContext = this
	}

	setActiveRoom = (roomID: RoomID | null) => {
		console.log("Switching to room", roomID)
		const room = (roomID && this.client.store.rooms.get(roomID)) || null
		this.directSetActiveRoom(room)
		this.setRightPanel(null)
		if (room?.stateLoaded === false) {
			this.client.loadRoomState(room.roomID)
				.catch(err => console.error("Failed to load room state", err))
		}
		this.client.store.activeRoomID = room?.roomID
		this.keybindings.activeRoom = room
		if (roomID) {
			document.querySelector(`div.room-entry[data-room-id="${CSS.escape(roomID)}"]`)
				?.scrollIntoView({ block: "nearest" })
		}
	}

	clickRoom = (evt: React.MouseEvent) => {
		const roomID = evt.currentTarget.getAttribute("data-room-id")
		if (roomID) {
			this.setActiveRoom(roomID)
		} else {
			console.warn("No room ID :(", evt.currentTarget)
		}
	}

	clickRightPanelOpener = (evt: React.MouseEvent) => {
		const type = evt.currentTarget.getAttribute("data-target-panel")
		if (type === "pinned-messages" || type === "members") {
			this.setRightPanel({ type })
		} else {
			throw new Error(`Invalid right panel type ${type}`)
		}
	}

	clearActiveRoom = () => this.setActiveRoom(null)
	closeRightPanel = () => this.setRightPanel(null)
}

const MainScreen = () => {
	const [activeRoom, directSetActiveRoom] = useState<RoomStateStore | null>(null)
	const [rightPanel, setRightPanel] = useReducer(rpReducer, null)
	const client = use(ClientContext)!
	const context = useMemo(
		() => new ContextFields(setRightPanel, directSetActiveRoom, client),
		[client],
	)
	useEffect(() => context.keybindings.listen(), [context])
	useInsertionEffect(() => {
		const styleTags = document.createElement("style")
		styleTags.textContent = `
			div.html-body > a.hicli-matrix-uri-user[href="matrix:u/${client.userID.slice(1).replaceAll(`"`, `\\"`)}"] {
				background-color: var(--highlight-pill-background-color);
				color: var(--highlight-pill-text-color);
			}
		`
		document.head.appendChild(styleTags)
		return () => {
			document.head.removeChild(styleTags)
		}
	}, [client.userID])
	const [roomListWidth, resizeHandle1] = useResizeHandle(
		300, 48, 900, "roomListWidth", { className: "room-list-resizer" },
	)
	const [rightPanelWidth, resizeHandle2] = useResizeHandle(
		300, 100, 900, "rightPanelWidth", { className: "right-panel-resizer", inverted: true },
	)
	const extraStyle = {
		["--room-list-width" as string]: `${roomListWidth}px`,
		["--right-panel-width" as string]: `${rightPanelWidth}px`,
	}
	const classNames = ["matrix-main"]
	if (activeRoom) {
		classNames.push("room-selected")
	}
	if (rightPanel) {
		classNames.push("right-panel-open")
	}
	return <main className={classNames.join(" ")} style={extraStyle}>
		<MainScreenContext value={context}>
			<RoomList activeRoomID={activeRoom?.roomID ?? null}/>
			{resizeHandle1}
			{activeRoom
				? <RoomView
					key={activeRoom.roomID}
					room={activeRoom}
					rightPanel={rightPanel}
					rightPanelResizeHandle={resizeHandle2}
				/>
				: rightPanel && <>
					{resizeHandle2}
					{rightPanel && <RightPanel {...rightPanel}/>}
				</>}
		</MainScreenContext>
	</main>
}

export default MainScreen
