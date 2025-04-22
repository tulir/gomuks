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
import React, { use } from "react"
import { getRoomAvatarThumbnailURL, getRoomAvatarURL } from "@/api/media.ts"
import { RoomStateStore } from "@/api/statestore"
import { getModalStyleFromButton } from "@/ui/menu/util.ts"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { LightboxContext, NestableModalContext } from "../modal"
import RoomStateExplorer from "../settings/RoomStateExplorer.tsx"
import SettingsView from "../settings/SettingsView.tsx"
import BackIcon from "@/icons/back.svg?react"
import CodeIcon from "@/icons/code.svg?react"
import PeopleIcon from "@/icons/group.svg?react"
import MoreIcon from "@/icons/more.svg?react"
import PinIcon from "@/icons/pin.svg?react"
import SettingsIcon from "@/icons/settings.svg?react"
import WidgetIcon from "@/icons/widgets.svg?react"
import "./RoomViewHeader.css"

interface RoomViewHeaderProps {
	room: RoomStateStore
}

const RoomViewHeader = ({ room }: RoomViewHeaderProps) => {
	const roomMeta = useEventAsState(room.meta)
	const mainScreen = use(MainScreenContext)
	const openNestableModal = use(NestableModalContext)
	const openSettings = () => {
		openNestableModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "settings-view",
			content: <SettingsView room={room} />,
		})
	}
	const openRoomStateExplorer = () => {
		openNestableModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "room-state-explorer-box",
			content: <RoomStateExplorer room={room} />,
		})
	}
	const buttonCount = 5
	const makeButtons = (titles?: boolean)  => {
		let rightPanelOpener = mainScreen.clickRightPanelOpener
		if (titles) {
			rightPanelOpener = (evt: React.MouseEvent) => {
				window.closeNestableModal()
				mainScreen.clickRightPanelOpener(evt)
			}
		}
		return <>
			<button
				data-target-panel="pinned-messages"
				onClick={rightPanelOpener}
				title="Pinned Messages"
			><PinIcon/>{titles && "Pinned Messages"}</button>
			<button
				data-target-panel="members"
				onClick={rightPanelOpener}
				title="Room Members"
			><PeopleIcon/>{titles && "Room Members"}</button>
			<button
				data-target-panel="widgets"
				onClick={rightPanelOpener}
				title="Widgets in room"
			><WidgetIcon/>{titles && "Widgets in room"}</button>
			<button title="Explore room state" onClick={openRoomStateExplorer}>
				<CodeIcon/>{titles && "Explore room state"}
			</button>
			<button title="Room Settings" onClick={openSettings}>
				<SettingsIcon/>{titles && "Room Settings"}
			</button>
		</>
	}
	const openButtonContextMenu = (evt: React.MouseEvent<HTMLButtonElement>) => {
		openNestableModal({
			content: <div className="context-menu" style={getModalStyleFromButton(evt.currentTarget, buttonCount * 16)}>
				{makeButtons(true)}
			</div>,
		})
	}
	return <div className="room-header">
		<button className="back" onClick={mainScreen.clearActiveRoom}><BackIcon/></button>
		<img
			className="avatar"
			loading="lazy"
			src={getRoomAvatarThumbnailURL(roomMeta)}
			data-full-src={getRoomAvatarURL(roomMeta)}
			onClick={use(LightboxContext)}
			alt=""
		/>
		<div className="room-name-and-topic">
			<div className="room-name">
				{roomMeta.name ?? roomMeta.room_id}
			</div>
			{roomMeta.topic && <div className="room-topic">
				{roomMeta.topic}
			</div>}
		</div>
		<div className="right-buttons big-screen">{makeButtons()}</div>
		<div className="right-buttons small-screen">
			<button onClick={openButtonContextMenu}><MoreIcon/></button>
		</div>
	</div>
}

export default RoomViewHeader
