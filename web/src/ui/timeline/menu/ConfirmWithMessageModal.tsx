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
import React, { use, useState } from "react"
import { MemDBEvent } from "@/api/types"
import { isMobileDevice } from "@/util/ismobile.ts"
import { ModalCloseContext } from "../../modal"
import TimelineEvent from "../TimelineEvent.tsx"

interface ConfirmWithMessageProps {
	evt?: MemDBEvent
	title: string
	description: string
	placeholder: string
	confirmButton: string
	onConfirm: (reason: string) => void
}

const ConfirmWithMessageModal = ({
	evt, title, description, placeholder, confirmButton, onConfirm,
}: ConfirmWithMessageProps) => {
	const [reason, setReason] = useState("")
	const closeModal = use(ModalCloseContext)
	const onConfirmWrapped = (evt: React.FormEvent) => {
		evt.preventDefault()
		closeModal()
		onConfirm(reason)
	}
	return <form onSubmit={onConfirmWrapped}>
		<h3>{title}</h3>
		{evt && <div className="timeline-event-container">
			<TimelineEvent evt={evt} prevEvt={null} disableMenu={true} />
		</div>}
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
		<div className="confirm-buttons">
			<button type="button" onClick={closeModal}>Cancel</button>
			<button type="submit">{confirmButton}</button>
		</div>
	</form>
}

export default ConfirmWithMessageModal
