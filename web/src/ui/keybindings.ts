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
import React from "react"
import { RoomStateStore, StateStore } from "@/api/statestore"
import { MainScreenContextFields } from "@/ui/MainScreenContext.ts"

export function keyToString(evt: React.KeyboardEvent | KeyboardEvent) {
	let key = evt.key
	if (evt.shiftKey) {
		key = "Shift+" + key
	}
	if (evt.altKey) {
		key = "Alt+" + key
	}
	if (evt.metaKey) {
		key = "Super+" + key
	}
	if (evt.ctrlKey) {
		key = "Ctrl+" + key
	}
	return key
}

type KeyMap = Record<string, (evt: KeyboardEvent) => void>

export default class Keybindings {
	public activeRoom: RoomStateStore | null = null
	constructor(private store: StateStore, private context: MainScreenContextFields) {}

	private keyDownMap: KeyMap = {
		"Ctrl+k": () => document.getElementById("room-search")?.focus(),
		"Alt+ArrowUp": () => {
			if (!this.activeRoom) {
				return
			}
			const activeRoomID = this.activeRoom.roomID
			const filteredRoomList = this.store.getFilteredRoomList()
			const selectedIdx = filteredRoomList.findLastIndex(room => room.room_id === activeRoomID)
			if (selectedIdx < filteredRoomList.length - 1) {
				this.context.setActiveRoom(filteredRoomList[selectedIdx + 1].room_id)
			} else {
				this.context.setActiveRoom(null)
			}
		},
		"Alt+ArrowDown": () => {
			const filteredRoomList = this.store.getFilteredRoomList()
			const selectedIdx = this.activeRoom
				? filteredRoomList.findLastIndex(room => room.room_id === this.activeRoom?.roomID)
				: -1
			if (selectedIdx === -1) {
				this.context.setActiveRoom(filteredRoomList[filteredRoomList.length - 1].room_id)
			} else if (selectedIdx > 0) {
				this.context.setActiveRoom(filteredRoomList[selectedIdx - 1].room_id)
			}
		},
	}

	private keyUpMap: KeyMap = {
		// "Escape": evt => evt.target === evt.currentTarget && this.context.clearActiveRoom(),
	}

	listen(): () => void {
		document.body.addEventListener("keydown", this.onKeyDown)
		document.body.addEventListener("keyup", this.onKeyUp)
		return () => {
			document.body.removeEventListener("keydown", this.onKeyDown)
			document.body.removeEventListener("keyup", this.onKeyUp)
		}
	}

	onKeyDown = (evt: KeyboardEvent) => {
		const key = keyToString(evt)
		const handler = this.keyDownMap[key]
		if (handler !== undefined) {
			evt.preventDefault()
			handler(evt)
		} else if (
			evt.target === evt.currentTarget
			&& this.keyUpMap[keyToString(evt)] === undefined
			&& ((!evt.ctrlKey && !evt.metaKey) || evt.key === "v" || evt.key === "a")
			&& !evt.altKey
			&& key !== "PageUp" && key !== "PageDown"
			&& key !== "Home" && key !== "End"
		) {
			document.getElementById("message-composer")?.focus()
		}
	}

	onKeyUp = (evt: KeyboardEvent) => {
		this.keyUpMap[keyToString(evt)]?.(evt)
	}
}
