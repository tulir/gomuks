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
import { JSX, useRef } from "react"
import { RoomStateStore } from "@/api/statestore"
import MessageComposer from "../composer/MessageComposer.tsx"
import RightPanel, { RightPanelProps } from "../rightpanel/RightPanel.tsx"
import TimelineView from "../timeline/TimelineView.tsx"
import RoomViewHeader from "./RoomViewHeader.tsx"
import { RoomContext, RoomContextData } from "./roomcontext.ts"
import "./RoomView.css"

interface RoomViewProps {
	room: RoomStateStore
	rightPanel: RightPanelProps | null
	rightPanelResizeHandle: JSX.Element
}

const onKeyDownRoomView = (evt: React.KeyboardEvent) => {
	if (evt.target === evt.currentTarget && (!evt.ctrlKey || evt.key === "v" || evt.key === "a") && !evt.altKey) {
		document.getElementById("message-composer")?.focus()
	}
}

const RoomView = ({ room, rightPanelResizeHandle, rightPanel }: RoomViewProps) => {
	const roomContextDataRef = useRef<RoomContextData | undefined>(undefined)
	if (roomContextDataRef.current === undefined) {
		roomContextDataRef.current = new RoomContextData(room)
	}
	return <RoomContext value={roomContextDataRef.current}>
		<div className="room-view" onKeyDown={onKeyDownRoomView} tabIndex={-1}>
			<RoomViewHeader room={room}/>
			<TimelineView/>
			<MessageComposer/>
		</div>
		{rightPanelResizeHandle}
		{rightPanel && <RightPanel {...rightPanel}/>}
	</RoomContext>
}

export default RoomView
