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
import { useState } from "react"
import { isMobileDevice } from "@/util/ismobile.ts"
import ConfirmModal, { ConfirmProps } from "./ConfirmModal.tsx"

export interface ConfirmWithMessageProps extends Omit<ConfirmProps<readonly [string]>, "confirmArgs" | "extraContent"> {
	placeholder: string
}

const ConfirmWithMessageModal = (props: ConfirmWithMessageProps) => {
	const [confirmArgs, setConfirmArgs] = useState<readonly [string]>([""])
	return <ConfirmModal<readonly [string]>
		{...props}
		confirmArgs={confirmArgs}
	>
		<input
			autoFocus={!isMobileDevice}
			value={confirmArgs[0]}
			type="text"
			placeholder={props.placeholder}
			onChange={evt => setConfirmArgs([evt.target.value])}
		/>
	</ConfirmModal>
}

export default ConfirmWithMessageModal
