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
import { use } from "react"
import { getRoomAvatarThumbnailURL, getRoomAvatarURL } from "@/api/media.ts"
import { RoomStateStore } from "@/api/statestore"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { LightboxContext } from "../modal"
import { ModalContext } from "../modal"
import SettingsView from "../settings/SettingsView.tsx"
import BackIcon from "@/icons/back.svg?react"
import PeopleIcon from "@/icons/group.svg?react"
import PinIcon from "@/icons/pin.svg?react"
import SettingsIcon from "@/icons/settings.svg?react"
import "./RoomViewHeader.css"

interface RoomViewHeaderProps {
	room: RoomStateStore
}

const RoomViewHeader = ({ room }: RoomViewHeaderProps) => {
	const roomMeta = useEventAsState(room.meta)
	const mainScreen = use(MainScreenContext)
	const openModal = use(ModalContext)
	const openSettings = () => {
		openModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "settings-view",
			content: <SettingsView room={room} />,
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
		<div className="right-buttons">
			<button
				data-target-panel="pinned-messages"
				onClick={mainScreen.clickRightPanelOpener}
				title="Pinned Messages"
			><PinIcon/></button>
			<button
				data-target-panel="members"
				onClick={mainScreen.clickRightPanelOpener}
				title="Room Members"
			><PeopleIcon/></button>
			<button title="Room Settings" onClick={openSettings}><SettingsIcon/></button>
		</div>
	</div>
}

export default RoomViewHeader
