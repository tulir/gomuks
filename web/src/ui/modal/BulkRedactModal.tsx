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

export type BulkRedactConfirmArgs = readonly [boolean, string]

export interface BulkRedactProps {
	userID: UserID
	evtCount: number
	nonStateEvtCount: number
	onConfirm: (preserveState: boolean, reason: string) => void
}

const BulkRedactModal = ({
	userID,
	onConfirm,
	evtCount,
	nonStateEvtCount,
}: BulkRedactProps) => {
	const [preserveState, setPreserveState] = useState(true)
	const [reason, setReason] = useState("")
	const confirmArgs = useMemo(() => [preserveState, reason] as const, [preserveState, reason])

	const targetEvtCount = preserveState ? nonStateEvtCount : evtCount
	return <ConfirmModal<BulkRedactConfirmArgs>
		title="Redact recent messages"
		description={<>
			Are you sure you want to redact all currently loaded timeline events
			of <code>{userID}</code>? This will remove {targetEvtCount} events.
		</>}
		confirmButton={`Redact ${targetEvtCount} events`}
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
		<table>
			<tbody>
				<tr>
					<td><label htmlFor="preserve-system-messages">Preserve system messages</label></td>
					<td>
						<Toggle
							id="preserve-system-messages"
							checked={preserveState}
							onChange={evt => setPreserveState(evt.target.checked)}
						/>
					</td>
				</tr>
			</tbody>
		</table>
	</ConfirmModal>
}

export default BulkRedactModal
