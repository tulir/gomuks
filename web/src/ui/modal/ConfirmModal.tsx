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
import React, { JSX, use } from "react"
import { MemDBEvent } from "@/api/types"
import TimelineEvent from "../timeline/TimelineEvent.tsx"
import { ModalCloseContext } from "./contexts.ts"
import "./ConfirmModal.css"

export interface ConfirmProps<T extends readonly unknown[] = []> {
	evt?: MemDBEvent
	title: string
	description?: string | JSX.Element
	confirmButton: string
	children?: JSX.Element | JSX.Element[]
	onConfirm: (...args: T) => void
	confirmArgs: T
}

const ConfirmModal = <T extends readonly unknown[] = []>({
	evt,
	title,
	description,
	children,
	confirmButton,
	onConfirm,
	confirmArgs,
}: ConfirmProps<T>) => {
	const closeModal = use(ModalCloseContext)
	const onConfirmWrapped = (evt: React.FormEvent) => {
		evt.preventDefault()
		closeModal()
		onConfirm(...confirmArgs)
	}
	return <form className="confirm-message-modal" onSubmit={onConfirmWrapped}>
		<h3>{title}</h3>
		{evt ? <div className="timeline-event-container">
			<TimelineEvent evt={evt} prevEvt={null} disableMenu={true} />
		</div> : null}
		{description ? <div className="confirm-description">
			{description}
		</div> : null}
		{children}
		<div className="confirm-buttons">
			<button type="button" onClick={closeModal}>Cancel</button>
			<button type="submit">{confirmButton}</button>
		</div>
	</form>
}


export default ConfirmModal
