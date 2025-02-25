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
import { CSSProperties, use } from "react"
import { RoomListEntry, RoomStateStore, useAccountData } from "@/api/statestore"
import { RoomID } from "@/api/types"
import ClientContext from "../../ClientContext.ts"
import { ModalContext } from "../../modal"
import SettingsView from "../../settings/SettingsView.tsx"
import NotificationsOffIcon from "@/icons/notifications-off.svg?react"
import NotificationsIcon from "@/icons/notifications.svg?react"
import SettingsIcon from "@/icons/settings.svg?react"
import "./RoomMenu.css"

interface RoomMenuProps {
	room: RoomStateStore
	entry: RoomListEntry
	style: CSSProperties
}

const hasNotifyingActions = (actions: unknown) => {
	return Array.isArray(actions) && actions.length > 0 && actions.includes("notify")
}

const MuteButton = ({ roomID }: { roomID: RoomID }) => {
	const client = use(ClientContext)!
	const roomRules = useAccountData(client.store, "m.push_rules")?.global?.room
	const pushRule = Array.isArray(roomRules) ? roomRules.find(rule => rule?.rule_id === roomID) : null
	const muted = pushRule?.enabled === true && !hasNotifyingActions(pushRule.actions)
	const toggleMute = () => {
		client.rpc.muteRoom(roomID, !muted).catch(err => {
			console.error("Failed to mute room", err)
			window.alert(`Failed to ${muted ? "unmute" : "mute"} room: ${err}`)
		})
	}
	return <button onClick={toggleMute}>
		{muted ? <NotificationsIcon/> : <NotificationsOffIcon/>}
		{muted ? "Unmute" : "Mute"}
	</button>
}

export const RoomMenu = ({ room, style }: RoomMenuProps) => {
	const openModal = use(ModalContext)
	const openSettings = () => {
		openModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "settings-view",
			content: <SettingsView room={room} />,
		})
	}
	return <div className="context-menu room-list-menu" style={style}>
		<MuteButton roomID={room.roomID}/>
		<button onClick={openSettings}><SettingsIcon /> Settings</button>
	</div>
}
