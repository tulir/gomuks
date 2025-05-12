// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Nexus Nicholson
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
import { useMemo, useState } from "react"
import { UserID } from "@/api/types"
import { isMobileDevice } from "@/util/ismobile.ts"
import Toggle from "../util/Toggle.tsx"
import ConfirmModal from "./ConfirmModal.tsx"

export type BulkRedactConfirmArgs = readonly [boolean, boolean, string]

export interface BulkRedactProps {
	userID: UserID
	isBanModal: boolean
	evtCount: number
	nonStateEvtCount: number
	onConfirm: (doRedact: boolean, preserveState: boolean, reason: string) => void
}

const BulkRedactModal = ({
	userID,
	onConfirm,
	isBanModal,
	evtCount,
	nonStateEvtCount,
}: BulkRedactProps) => {
	const [doRedact, setDoRedact] = useState(!isBanModal)
	const [preserveState, setPreserveState] = useState(true)
	const [reason, setReason] = useState("")
	const confirmArgs = useMemo(() => [doRedact, preserveState, reason] as const, [doRedact, preserveState, reason])

	const targetEvtCount = preserveState ? nonStateEvtCount : evtCount
	const redactConfirmation = `Redact ${targetEvtCount} events`
	return <ConfirmModal<BulkRedactConfirmArgs>
		title={isBanModal ? "Ban user" : "Redact recent messages"}
		description={isBanModal
			? <>Are you sure you want to ban <code>{userID}</code>?</>
			: <>
				Are you sure you want to redact all currently loaded timeline events
				of <code>{userID}</code>? This will remove {targetEvtCount} events.
			</>}
		confirmButton={
			isBanModal
				? (doRedact ? `Ban and ${redactConfirmation.toLowerCase()}` : "Ban")
				: redactConfirmation
		}
		onConfirm={onConfirm}
		confirmArgs={confirmArgs}
	>
		<input
			autoFocus={!isMobileDevice}
			value={reason}
			type="text"
			placeholder="Reason (optional)"
			onChange={evt => setReason(evt.target.value)}
		/>
		<div className="toggle-sheet">
			{isBanModal ? <>
				<label htmlFor="redact-recent-messages">Redact recent messages</label>
				<Toggle
					id="redact-recent-messages"
					checked={doRedact}
					onChange={evt => setDoRedact(evt.target.checked)}
				/>
			</> : null}
			{doRedact ? <>
				<label htmlFor="preserve-system-messages">Preserve system messages</label>
				<Toggle
					id="preserve-system-messages"
					checked={preserveState}
					onChange={evt => setPreserveState(evt.target.checked)}
				/>
			</> : null}
		</div>
	</ConfirmModal>
}

export default BulkRedactModal
