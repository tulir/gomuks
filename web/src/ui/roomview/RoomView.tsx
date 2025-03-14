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
import { JSX, useEffect, useState } from "react"
import { RoomStateStore } from "@/api/statestore"
import MessageComposer from "../composer/MessageComposer.tsx"
import TypingNotifications from "../composer/TypingNotifications.tsx"
import RightPanel, { RightPanelProps } from "../rightpanel/RightPanel.tsx"
import TimelineView from "../timeline/TimelineView.tsx"
import ErrorBoundary from "../util/ErrorBoundary.tsx"
import RoomViewHeader from "./RoomViewHeader.tsx"
import { RoomContext, RoomContextData } from "./roomcontext.ts"
import "./RoomView.css"

interface RoomViewProps {
	room: RoomStateStore
	rightPanel: RightPanelProps | null
	rightPanelResizeHandle: JSX.Element
}

const RoomView = ({ room, rightPanelResizeHandle, rightPanel }: RoomViewProps) => {
	const [roomContextData] = useState(() => new RoomContextData(room))
	useEffect(() => {
		window.activeRoomContext = roomContextData
		window.addEventListener("resize", roomContextData.scrollToBottom)
		return () => {
			window.removeEventListener("resize", roomContextData.scrollToBottom)
			if (window.activeRoomContext === roomContextData) {
				window.activeRoomContext = undefined
			}
		}
	}, [roomContextData])
	const onClick = (evt: React.MouseEvent<HTMLDivElement>) => {
		if (roomContextData.focusedEventRowID) {
			roomContextData.setFocusedEventRowID(null)
			evt.stopPropagation()
		}
	}
	return <RoomContext value={roomContextData}>
		<div className="room-view" onClick={onClick}>
			<ErrorBoundary thing="room view" wrapperClassName="room-view-error">
				<div id="mobile-event-menu-container"/>
				<RoomViewHeader room={room}/>
				<TimelineView/>
				<MessageComposer/>
				<TypingNotifications/>
			</ErrorBoundary>
		</div>
		{rightPanelResizeHandle}
		{rightPanel && <RightPanel {...rightPanel}/>}
	</RoomContext>
}

export default RoomView
