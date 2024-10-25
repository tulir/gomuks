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
import { use, useCallback } from "react"
import { useRoomState } from "@/api/statestore"
import { MemDBEvent, PowerLevelEventContent } from "@/api/types"
import { useNonNullEventAsState } from "@/util/eventdispatcher.ts"
import { ClientContext } from "../ClientContext.ts"
import { useRoomContext } from "../roomcontext.ts"
import EditIcon from "../../icons/edit.svg?react"
import MoreIcon from "../../icons/more.svg?react"
import PinIcon from "../../icons/pin.svg?react"
import ReactIcon from "../../icons/react.svg?react"
import ReplyIcon from "../../icons/reply.svg?react"
import UnpinIcon from "../../icons/unpin.svg?react"
import "./EventMenu.css"

interface EventHoverMenuProps {
	evt: MemDBEvent
}

const EventMenu = ({ evt }: EventHoverMenuProps) => {
	const client = use(ClientContext)!
	const userID = client.userID
	const roomCtx = useRoomContext()
	const onClickReply = useCallback(() => roomCtx.setReplyTo(evt.event_id), [roomCtx, evt.event_id])
	const onClickPin = useCallback(() => {
		client.pinMessage(roomCtx.store, evt.event_id, true)
			.catch(err => window.alert(`Failed to pin message: ${err}`))
	}, [client, roomCtx, evt.event_id])
	const onClickUnpin = useCallback(() => {
		client.pinMessage(roomCtx.store, evt.event_id, false)
			.catch(err => window.alert(`Failed to unpin message: ${err}`))
	}, [client, roomCtx, evt.event_id])
	const onClickReact = useCallback(() => {
		window.alert("No reactions yet :(")
	}, [])
	const onClickEdit = useCallback(() => {
		roomCtx.setEditing(evt)
	}, [roomCtx, evt])
	const onClickMore = useCallback(() => {
		window.alert("Nothing here yet :(")
	}, [])
	const isEditing = useNonNullEventAsState(roomCtx.isEditing)
	const plEvent = useRoomState(roomCtx.store, "m.room.power_levels", "")
	// We get pins from getPinnedEvents, but use the hook anyway to subscribe to changes
	useRoomState(roomCtx.store, "m.room.pinned_events", "")
	const pls = (plEvent?.content ?? {}) as PowerLevelEventContent
	const pins = roomCtx.store.getPinnedEvents()
	const ownPL = pls.users?.[userID] ?? pls.users_default ?? 0
	const pinPL = pls.events?.["m.room.pinned_events"] ?? pls.state_default ?? 50
	return <div className="context-menu">
		<button onClick={onClickReact}><ReactIcon/></button>
		<button
			disabled={isEditing}
			title={isEditing ? "Can't reply to messages while editing a message" : undefined}
			onClick={onClickReply}
		><ReplyIcon/></button>
		{evt.sender === userID && evt.type === "m.room.message" && evt.relation_type !== "m.replace"
			&& <button onClick={onClickEdit}><EditIcon/></button>}
		{ownPL >= pinPL && (pins.includes(evt.event_id)
			? <button onClick={onClickUnpin}><UnpinIcon/></button>
			: <button onClick={onClickPin}><PinIcon/></button>)}
		<button onClick={onClickMore}><MoreIcon/></button>
	</div>
}

export default EventMenu
