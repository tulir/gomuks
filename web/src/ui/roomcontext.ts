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
import { RefObject, createContext, createRef, use } from "react"
import { RoomStateStore } from "@/api/statestore"
import { EventID, EventRowID, MemDBEvent } from "@/api/types"
import { NonNullCachedEventDispatcher } from "@/util/eventdispatcher.ts"

const noop = (name: string) => () => {
	console.warn(`${name} called before initialization`)
}

export class RoomContextData {
	public timelineBottomRef: RefObject<HTMLDivElement | null> = createRef()
	public setReplyTo: (eventID: EventID | null) => void = noop("setReplyTo")
	public setEditing: (evt: MemDBEvent | null) => void = noop("setEditing")
	public isEditing = new NonNullCachedEventDispatcher<boolean>(false)
	public ownMessages: EventRowID[] = []
	public scrolledToBottom = true

	constructor(public store: RoomStateStore) {}

	scrollToBottom() {
		if (this.scrolledToBottom) {
			this.timelineBottomRef.current?.scrollIntoView()
		}
	}
}

export const RoomContext = createContext<RoomContextData | undefined>(undefined)

export const useRoomContext = () => {
	const roomCtx = use(RoomContext)
	if (!roomCtx) {
		throw new Error("useRoomContext called outside RoomContext provider")
	}
	return roomCtx
}
