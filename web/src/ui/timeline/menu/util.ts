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
import React, { CSSProperties } from "react"
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import { MemDBEvent, PowerLevelEventContent } from "@/api/types"

export const getPending = (evt: MemDBEvent): [pending: boolean, pendingTitle: string | undefined] => {
	const isPending = evt.event_id.startsWith("~")
	const pendingTitle = isPending ? "Can't action messages that haven't been sent yet" : undefined
	return [isPending, pendingTitle]
}

export const getPowerLevels = (room: RoomStateStore, client: Client): [pls: PowerLevelEventContent, ownPL: number] => {
	const plEvent = room.getStateEvent("m.room.power_levels", "")
	const pls = (plEvent?.content ?? {}) as PowerLevelEventContent
	const ownPL = pls.users?.[client.userID] ?? pls.users_default ?? 0
	return [pls, ownPL]
}

export const getEncryption = (room: RoomStateStore): boolean =>{
	const encryptionEvent = room.getStateEvent("m.room.encryption", "")
	return encryptionEvent?.content?.algorithm === "m.megolm.v1.aes-sha2"
}

export function getModalStyleFromMouse(
	evt: React.MouseEvent, modalHeight: number, modalWidth = 10 * 16,
): CSSProperties {
	const style: CSSProperties = { right: window.innerWidth - evt.clientX }
	if (evt.clientX - modalWidth < 4) {
		delete style.right
		style.left = "4px"
	}
	if (evt.clientY + modalHeight > window.innerHeight) {
		style.bottom = window.innerHeight - evt.clientY
	} else {
		style.top = evt.clientY
	}
	return style
}

export function getModalStyleFromButton(button: HTMLButtonElement, modalHeight: number): CSSProperties {
	const rect = button.getBoundingClientRect()
	const style: CSSProperties = { right: window.innerWidth - rect.right }
	if (rect.bottom + modalHeight > window.innerHeight) {
		style.bottom = window.innerHeight - rect.top
	} else {
		style.top = rect.bottom
	}
	return style
}
