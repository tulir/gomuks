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
import { use, useCallback, useState } from "react"
import { MemDBEvent } from "@/api/types"
import useEvent from "@/util/useEvent.ts"
import { ModalCloseContext } from "../../modal/Modal.tsx"
import TimelineEvent from "../TimelineEvent.tsx"

interface ConfirmWithMessageProps {
	evt: MemDBEvent
	title: string
	description: string
	placeholder: string
	confirmButton: string
	onConfirm: (reason: string) => void
}

const ConfirmWithMessageProps = ({
	evt, title, description, placeholder, confirmButton, onConfirm,
}: ConfirmWithMessageProps) => {
	const [reason, setReason] = useState("")
	const closeModal = use(ModalCloseContext)
	const onConfirmWrapped = useEvent(() => {
		closeModal()
		onConfirm(reason)
	})
	const onChangeReason = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setReason(evt.target.value)
	}, [])
	return <div className="confirm-message-modal">
		<h3>{title}</h3>
		<div className="timeline-event-container">
			<TimelineEvent evt={evt} prevEvt={null} disableMenu={true} />
		</div>
		<div className="confirm-description">
			{description}
		</div>
		<input value={reason} type="text" placeholder={placeholder} onChange={onChangeReason} />
		<div className="confirm-buttons">
			<button onClick={closeModal}>Cancel</button>
			<button onClick={onConfirmWrapped}>{confirmButton}</button>
		</div>
	</div>
}

export default ConfirmWithMessageProps