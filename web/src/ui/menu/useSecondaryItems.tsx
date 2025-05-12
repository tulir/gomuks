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
import Client from "@/api/client.ts"
import { useRoomState } from "@/api/statestore"
import { MemDBEvent } from "@/api/types"
import { ConfirmWithMessageModal, ModalCloseContext, ModalContext, ShareModal } from "../modal"
import { RoomContext, RoomContextData } from "../roomview/roomcontext.ts"
import JSONView from "../util/JSONView.tsx"
import { getPending, getPowerLevels } from "./util.ts"
import ViewSourceIcon from "@/icons/code.svg?react"
import DeleteIcon from "@/icons/delete.svg?react"
import PinIcon from "@/icons/pin.svg?react"
import ReportIcon from "@/icons/report.svg?react"
import RestoreTrashIcon from "@/icons/restore-trash.svg?react"
import ShareIcon from "@/icons/share.svg?react"
import UnpinIcon from "@/icons/unpin.svg?react"

export const useSecondaryItems = (
	client: Client,
	roomCtx: RoomContextData,
	evt: MemDBEvent,
	names = true,
) => {
	const closeModal = use(ModalCloseContext)
	const openModal = use(ModalContext)
	const onClickViewSource = () => {
		openModal({
			dimmed: true,
			boxed: true,
			content: <JSONView data={evt}/>,
		})
	}
	const onClickReport = () => {
		openModal({
			dimmed: true,
			boxed: true,
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
	}
	const onClickRedact = () => {
		openModal({
			dimmed: true,
			boxed: true,
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
	}
	const onClickHideUnredacted = () => {
		closeModal()
		roomCtx.store.setViewingRedacted(evt, false)
	}
	const onClickUnredact = () => {
		closeModal()
		if (Object.entries(evt.content).length > 0) {
			roomCtx.store.setViewingRedacted(evt, true)
		} else {
			client.requestEvent(roomCtx.store, evt.event_id, true)
		}
	}
	const onClickPin = (pin: boolean) => () => {
		closeModal()
		client.pinMessage(roomCtx.store, evt.event_id, pin)
			.catch(err => window.alert(`Failed to ${pin ? "pin" : "unpin"} message: ${err}`))
	}

	const onClickShareEvent = () => {
		const generateLink = (useMatrixTo: boolean, includeEvent: boolean) => {
			const isRoomIDLink = true
			let generatedURL = useMatrixTo ? "https://matrix.to/#/" : "matrix:roomid/"
			if (useMatrixTo) {
				generatedURL += encodeURIComponent(evt.room_id)
			} else {
				generatedURL += encodeURIComponent(`${evt.room_id.slice(1)}`)
			}
			if (includeEvent) {
				if (useMatrixTo) {
					generatedURL += `/${encodeURIComponent(evt.event_id)}`
				} else {
					generatedURL += `/e/${encodeURIComponent(evt.event_id.slice(1))}`
				}
			}
			if (isRoomIDLink) {
				generatedURL += "?" + new URLSearchParams(
					roomCtx.store.getViaServers().map(server => ["via", server]),
				).toString()
			}
			return generatedURL
		}
		openModal({
			dimmed: true,
			boxed: true,
			content: <RoomContext value={roomCtx}>
				<ShareModal
					evt={evt}
					onConfirm={(useMatrixTo: boolean, includeEvent: boolean) => {
						navigator.clipboard.writeText(generateLink(useMatrixTo, includeEvent)).catch(
							err => window.alert(`Failed to copy link: ${err}`),
						)
					}}
					generateLink={generateLink}
				/>
			</RoomContext>,
		})
	}

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
	// TODO check server admin status and room PLs
	const canUnredact = Boolean(evt.redacted_by)

	return <>
		<button onClick={onClickViewSource}><ViewSourceIcon/>{names && "View source"}</button>
		<button onClick={onClickShareEvent}><ShareIcon/>{names && "Share"}</button>
		{ownPL >= pinPL && (pins.includes(evt.event_id)
			? <button onClick={onClickPin(false)}>
				<UnpinIcon/>{names && "Unpin message"}
			</button>
			: <button onClick={onClickPin(true)} title={pendingTitle} disabled={isPending}>
				<PinIcon/>{names && "Pin message"}
			</button>)}
		<button onClick={onClickReport} disabled={isPending} title={pendingTitle}>
			<ReportIcon/>{names && "Report"}
		</button>
		{canRedact && <button
			onClick={onClickRedact}
			disabled={isPending}
			title={pendingTitle}
			className="redact-button"
		><DeleteIcon/>{names && "Remove"}</button>}
		{canUnredact && (evt.viewing_redacted ? <button onClick={onClickHideUnredacted}>
			<DeleteIcon/>{names && "Hide content"}
		</button> : <button onClick={onClickUnredact}>
			<RestoreTrashIcon/>{names && "View content"}
		</button>)}
	</>
}
