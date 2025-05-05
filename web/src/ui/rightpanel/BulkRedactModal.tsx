import React, { JSX, use, useState } from "react"
import { ModalCloseContext } from "@/ui/modal"
import Toggle from "@/ui/util/Toggle.tsx"
import { isMobileDevice } from "@/util/ismobile.ts"

interface BulkRedactProps {
	title: string
	description: string | JSX.Element
	placeholder: string
	confirmButton: string
	onConfirm: (preserveState: boolean, reason: string) => void
}

const BulkRedactModal = ({ title, description, placeholder, confirmButton, onConfirm }: BulkRedactProps) => {
	const [preserveState, setPreserveState] = useState(true)
	const [reason, setReason] = useState("")
	const closeModal = use(ModalCloseContext)
	const onConfirmWrapped = (evt: React.FormEvent) => {
		evt.preventDefault()
		closeModal()
		onConfirm(preserveState, reason)
	}

	return <form onSubmit={onConfirmWrapped}>
		<h3>{title}</h3>
		<div className="confirm-description">
			{description}
		</div>
		<input
			autoFocus={!isMobileDevice}
			value={reason}
			type="text"
			placeholder={placeholder}
			onChange={evt => setReason(evt.target.value)}
		/>
		<table>
			<tbody>
				<tr>
					<td>Preserve system messages</td>
					<td>
						<Toggle
							id="useMatrixTo"
							checked={preserveState}
							onChange={evt => setPreserveState(evt.target.checked)}
						/>
					</td>
				</tr>
				<tr>
					<td>This will keep events like their profile and membership</td>
				</tr>
			</tbody>
		</table>
		<div className="confirm-buttons">
			<button type="button" onClick={closeModal}>Cancel</button>
			<button type="submit">{confirmButton}</button>
		</div>
	</form>
}

export default BulkRedactModal
