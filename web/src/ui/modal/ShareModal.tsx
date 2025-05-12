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
import { useMemo, useState } from "react"
import { MemDBEvent } from "@/api/types"
import Toggle from "../util/Toggle.tsx"
import ConfirmModal from "./ConfirmModal.tsx"

type ShareConfirmArgs = readonly [boolean, boolean]

export interface ShareModalProps {
	evt: MemDBEvent
	onConfirm: (useMatrixTo: boolean, includeEvent: boolean) => void
	generateLink: (useMatrixTo: boolean, includeEvent: boolean) => string
}

const ShareModal = ({ evt, onConfirm, generateLink }: ShareModalProps) => {
	const [useMatrixTo, setUseMatrixTo] = useState(false)
	const [includeEvent, setIncludeEvent] = useState(true)
	const confirmArgs = useMemo(() => [useMatrixTo, includeEvent] as const, [useMatrixTo, includeEvent])
	const link = generateLink(useMatrixTo, includeEvent)

	return <ConfirmModal<ShareConfirmArgs>
		evt={evt}
		title="Share Message"
		confirmButton="Copy to clipboard"
		onConfirm={onConfirm}
		confirmArgs={confirmArgs}
	>
		<div className="toggle-sheet">
			<label htmlFor="use-matrix-to">Use matrix.to link</label>
			<Toggle
				id="use-matrix-to"
				checked={useMatrixTo}
				onChange={evt => setUseMatrixTo(evt.target.checked)}
			/>
			<label htmlFor="share-event">Link to this specific event</label>
			<Toggle
				id="share-event"
				checked={includeEvent}
				onChange={evt => setIncludeEvent(evt.target.checked)}
			/>
		</div>
		<div className="output-preview">
			<span className="no-select">Preview: </span><code>{link}</code>
		</div>
	</ConfirmModal>
}

export default ShareModal
