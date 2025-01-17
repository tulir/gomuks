import React, { use, useState } from "react"
import { MemDBEvent } from "@/api/types"
import { ModalCloseContext } from "@/ui/modal"
import TimelineEvent from "@/ui/timeline/TimelineEvent.tsx"
import Toggle from "@/ui/util/Toggle.tsx"

interface ConfirmWithMessageProps {
	evt: MemDBEvent
	title: string
	confirmButton: string
	onConfirm: (useMatrixTo: boolean, includeEvent: boolean) => void
	generateLink: (useMatrixTo: boolean, includeEvent: boolean) => string
}

const ShareModal = ({ evt, title, confirmButton, onConfirm, generateLink }: ConfirmWithMessageProps) => {
	const [useMatrixTo, setUseMatrixTo] = useState(false)
	const [includeEvent, setIncludeEvent] = useState(true)
	const closeModal = use(ModalCloseContext)
	const onConfirmWrapped = (evt: React.FormEvent) => {
		evt.preventDefault()
		closeModal()
		onConfirm(useMatrixTo, includeEvent)
	}

	const link = generateLink(useMatrixTo, includeEvent)
	return <form onSubmit={onConfirmWrapped}>
		<h3>{title}</h3>
		<div className="timeline-event-container">
			<TimelineEvent evt={evt} prevEvt={null} disableMenu={true}/>
		</div>
		<table>
			<tbody>
				<tr>
					<td>Use matrix.to link</td>
					<td>
						<Toggle
							id="useMatrixTo"
							checked={useMatrixTo}
							onChange={evt => setUseMatrixTo(evt.target.checked)}
						/>
					</td>
				</tr>
				<tr>
					<td>Link to this specific event</td>
					<td>
						<Toggle
							id="shareEvent"
							checked={includeEvent}
							onChange={evt => setIncludeEvent(evt.target.checked)}
						/>
					</td>
				</tr>
			</tbody>
		</table>
		<div className="description">
			Share: <a href={link} target="_blank" rel="noreferrer">{link}</a>
		</div>
		<div className="confirm-buttons">
			<button type="button" onClick={closeModal}>Cancel</button>
			<button type="submit">{confirmButton}</button>
		</div>
	</form>
}

export default ShareModal
