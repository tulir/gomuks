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
import { useEventAsState } from "@/util/eventdispatcher.ts"
import ClientContext from "../ClientContext.ts"
import { ModalCloseContext, ModalContext } from "../modal"
import SettingsView from "../settings/SettingsView.tsx"
import DoorOpenIcon from "@/icons/door-open.svg?react"
import MarkReadIcon from "@/icons/mark-read.svg?react"
import MarkUnreadIcon from "@/icons/mark-unread.svg?react"
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
	const closeModal = use(ModalCloseContext)
	const roomRules = useAccountData(client.store, "m.push_rules")?.global?.room
	const pushRule = Array.isArray(roomRules) ? roomRules.find(rule => rule?.rule_id === roomID) : null
	const muted = pushRule?.enabled === true && !hasNotifyingActions(pushRule.actions)
	const toggleMute = () => {
		client.rpc.muteRoom(roomID, !muted).catch(err => {
			console.error("Failed to mute room", err)
			window.alert(`Failed to ${muted ? "unmute" : "mute"} room: ${err}`)
		})
		closeModal()
	}
	return <button onClick={toggleMute}>
		{muted ? <NotificationsIcon/> : <NotificationsOffIcon/>}
		{muted ? "Unmute" : "Mute"}
	</button>
}

const MarkReadButton = ({ room }: { room: RoomStateStore }) => {
	const meta = useEventAsState(room.meta)
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const read = !meta.marked_unread && meta.unread_messages === 0
	const markRead = () => {
		const evt = room.eventsByRowID.get(
			room.timeline[room.timeline.length-1]?.event_rowid ?? meta.preview_event_rowid,
		)
		if (!evt) {
			window.alert("Can't mark room as read: last event not found in cache")
			return
		}
		const rrType = room.preferences.send_read_receipts ? "m.read" : "m.read.private"
		client.rpc.markRead(room.roomID, evt.event_id, rrType).catch(err => {
			console.error("Failed to mark room as read", err)
			window.alert(`Failed to mark room as read: ${err}`)
		})
		closeModal()
	}
	const markUnread = () => {
		client.rpc.setAccountData("m.marked_unread", { unread: true }, room.roomID).catch(err => {
			console.error("Failed to mark room as unread", err)
			window.alert(`Failed to mark room as unread: ${err}`)
		})
		closeModal()
	}
	return <button onClick={read ? markUnread : markRead}>
		{read ? <MarkUnreadIcon/> : <MarkReadIcon/>}
		Mark {read ? "unread" : "read"}
	</button>
}

export const RoomMenu = ({ room, style }: RoomMenuProps) => {
	const openModal = use(ModalContext)
	const closeModal = use(ModalCloseContext)
	const client = use(ClientContext)!
	const openSettings = () => {
		openModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "settings-view",
			content: <SettingsView room={room} />,
		})
	}
	const leaveRoom = () => {
		if (!window.confirm(`Really leave ${room.meta.current.name}?`)) {
			return
		}
		client.rpc.leaveRoom(room.roomID).catch(err => {
			console.error("Failed to leave room", err)
			window.alert(`Failed to leave room: ${err}`)
		})
		closeModal()
	}
	return <div className="context-menu room-list-menu" style={style}>
		<MarkReadButton room={room} />
		<MuteButton roomID={room.roomID}/>
		<button onClick={openSettings}><SettingsIcon /> Settings</button>
		<button onClick={leaveRoom}><DoorOpenIcon /> Leave room</button>
	</div>
}
