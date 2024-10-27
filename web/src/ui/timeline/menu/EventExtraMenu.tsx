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
import { CSSProperties, use, useCallback } from "react"
import { RoomStateStore, useRoomState } from "@/api/statestore"
import { MemDBEvent, PowerLevelEventContent } from "@/api/types"
import ClientContext from "../../ClientContext.ts"
import { ModalCloseContext, ModalContext } from "../../modal/Modal.tsx"
import { RoomContext, RoomContextData } from "../../roomcontext.ts"
import ConfirmWithMessageModal from "./ConfirmWithMessageModal.tsx"
import ViewSourceModal from "./ViewSourceModal.tsx"
import ViewSourceIcon from "@/icons/code.svg?react"
import DeleteIcon from "@/icons/delete.svg?react"
import PinIcon from "@/icons/pin.svg?react"
import ReportIcon from "@/icons/report.svg?react"
import UnpinIcon from "@/icons/unpin.svg?react"

interface EventExtraMenuProps {
	evt: MemDBEvent
	room: RoomStateStore
	style: CSSProperties
}

const EventExtraMenu = ({ evt, room, style }: EventExtraMenuProps) => {
	const client = use(ClientContext)!
	const userID = client.userID
	const closeModal = use(ModalCloseContext)
	const openModal = use(ModalContext)
	const onClickViewSource = useCallback(() => {
		openModal({ dimmed: true, content: <ViewSourceModal evt={evt}/> })
	}, [evt, openModal])
	const onClickReport = useCallback(() => {
		openModal({
			dimmed: true,
			content: <RoomContext value={new RoomContextData(room)}>
				<ConfirmWithMessageModal
					evt={evt}
					title="Report Message"
					description="Report this message to your homeserver administrator?"
					placeholder="Reason for report"
					confirmButton="Send report"
					onConfirm={reason => {
						client.rpc.reportEvent(evt.room_id, evt.event_id, reason)
							.catch(err => window.alert(`Failed to report message: ${err}`))
					}}
				/>
			</RoomContext>,
		})
	}, [evt, room, openModal, client])
	const onClickRedact = useCallback(() => {
		openModal({
			dimmed: true,
			content: <RoomContext value={new RoomContextData(room)}>
				<ConfirmWithMessageModal
					evt={evt}
					title="Remove Message"
					description="Permanently remove the content of this event?"
					placeholder="Reason for removal"
					confirmButton="Remove"
					onConfirm={reason => {
						client.rpc.redactEvent(evt.room_id, evt.event_id, reason)
							.catch(err => window.alert(`Failed to redact message: ${err}`))
					}}
				/>
			</RoomContext>,
		})
	}, [evt, room, openModal, client])
	const onClickPin = useCallback(() => {
		closeModal()
		client.pinMessage(room, evt.event_id, true)
			.catch(err => window.alert(`Failed to pin message: ${err}`))
	}, [closeModal, client, room, evt.event_id])
	const onClickUnpin = useCallback(() => {
		closeModal()
		client.pinMessage(room, evt.event_id, false)
			.catch(err => window.alert(`Failed to unpin message: ${err}`))
	}, [closeModal, client, room, evt.event_id])

	const plEvent = useRoomState(room, "m.room.power_levels", "")
	// We get pins from getPinnedEvents, but use the hook anyway to subscribe to changes
	useRoomState(room, "m.room.pinned_events", "")
	const pls = (plEvent?.content ?? {}) as PowerLevelEventContent
	const pins = room.getPinnedEvents()
	const ownPL = pls.users?.[userID] ?? pls.users_default ?? 0
	const pinPL = pls.events?.["m.room.pinned_events"] ?? pls.state_default ?? 50
	const redactEvtPL = pls.events?.["m.room.redaction"] ?? pls.events_default ?? 0
	const redactOtherPL = pls.redact ?? 50
	const canRedact = !evt.redacted_by && ownPL >= redactEvtPL && (evt.sender === userID || ownPL >= redactOtherPL)

	return <div style={style} className="event-context-menu-extra">
		<button onClick={onClickViewSource}><ViewSourceIcon/>View source</button>
		{ownPL >= pinPL && (pins.includes(evt.event_id)
			? <button onClick={onClickUnpin}><UnpinIcon/>Unpin message</button>
			: <button onClick={onClickPin}><PinIcon/>Pin message</button>)}
		<button onClick={onClickReport}><ReportIcon/>Report</button>
		{canRedact && <button onClick={onClickRedact} className="redact-button"><DeleteIcon/>Remove</button>}
	</div>
}

export default EventExtraMenu
