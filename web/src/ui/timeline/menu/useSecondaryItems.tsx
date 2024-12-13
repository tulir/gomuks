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
import Client from "@/api/client.ts"
import { useRoomState } from "@/api/statestore"
import { MemDBEvent } from "@/api/types"
import { ModalCloseContext, ModalContext } from "../../modal/Modal.tsx"
import { RoomContext, RoomContextData } from "../../roomview/roomcontext.ts"
import JSONView from "../../util/JSONView.tsx"
import ConfirmWithMessageModal from "./ConfirmWithMessageModal.tsx"
import { getPending, getPowerLevels } from "./util.ts"
import ViewSourceIcon from "@/icons/code.svg?react"
import DeleteIcon from "@/icons/delete.svg?react"
import PinIcon from "@/icons/pin.svg?react"
import ReportIcon from "@/icons/report.svg?react"
import UnpinIcon from "@/icons/unpin.svg?react"

export const useSecondaryItems = (
	client: Client,
	roomCtx: RoomContextData,
	evt: MemDBEvent,
) => {
	const closeModal = use(ModalCloseContext)
	const openModal = use(ModalContext)
	const onClickViewSource = useCallback(() => {
		openModal({
			dimmed: true,
			boxed: true,
			content: <JSONView data={evt} />,
		})
	}, [evt, openModal])
	const onClickReport = useCallback(() => {
		openModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "confirm-message-modal",
			content: <RoomContext value={roomCtx}>
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
	}, [evt, roomCtx, openModal, client])
	const onClickRedact = useCallback(() => {
		openModal({
			dimmed: true,
			boxed: true,
			innerBoxClass: "confirm-message-modal",
			content: <RoomContext value={roomCtx}>
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
	}, [evt, roomCtx, openModal, client])
	const onClickPin = useCallback(() => {
		closeModal()
		client.pinMessage(roomCtx.store, evt.event_id, true)
			.catch(err => window.alert(`Failed to pin message: ${err}`))
	}, [closeModal, client, roomCtx, evt.event_id])
	const onClickUnpin = useCallback(() => {
		closeModal()
		client.pinMessage(roomCtx.store, evt.event_id, false)
			.catch(err => window.alert(`Failed to unpin message: ${err}`))
	}, [closeModal, client, roomCtx, evt.event_id])

	const [isPending, pendingTitle] = getPending(evt)
	useRoomState(roomCtx.store, "m.room.power_levels", "")
	// We get pins from getPinnedEvents, but use the hook anyway to subscribe to changes
	useRoomState(roomCtx.store, "m.room.pinned_events", "")
	const [pls, ownPL] = getPowerLevels(roomCtx.store, client)
	const pins = roomCtx.store.getPinnedEvents()
	const pinPL = pls.events?.["m.room.pinned_events"] ?? pls.state_default ?? 50
	const redactEvtPL = pls.events?.["m.room.redaction"] ?? pls.events_default ?? 0
	const redactOtherPL = pls.redact ?? 50
	const canRedact = !evt.redacted_by
		&& ownPL >= redactEvtPL
		&& (evt.sender === client.userID || ownPL >= redactOtherPL)

	return <>
		<button onClick={onClickViewSource}><ViewSourceIcon/>View source</button>
		{ownPL >= pinPL && (pins.includes(evt.event_id)
			? <button onClick={onClickUnpin}><UnpinIcon/>Unpin message</button>
			: <button onClick={onClickPin} title={pendingTitle} disabled={isPending}><PinIcon/>Pin message</button>)}
		<button onClick={onClickReport} disabled={isPending} title={pendingTitle}><ReportIcon/>Report</button>
		{canRedact && <button
			onClick={onClickRedact}
			disabled={isPending}
			title={pendingTitle}
			className="redact-button"
		><DeleteIcon/>Remove</button>}
	</>
}