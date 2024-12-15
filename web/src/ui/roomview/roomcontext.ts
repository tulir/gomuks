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
import { escapeMarkdown } from "@/util/markdown.ts"

const noop = (name: string) => () => {
	console.warn(`${name} called before initialization`)
}

export class RoomContextData {
	public readonly timelineBottomRef: RefObject<HTMLDivElement | null> = createRef()
	public setReplyTo: (eventID: EventID | null) => void = noop("setReplyTo")
	public setEditing: (evt: MemDBEvent | null) => void = noop("setEditing")
	public insertText: (text: string) => void = noop("insertText")
	public readonly isEditing = new NonNullCachedEventDispatcher<boolean>(false)
	public ownMessages: EventRowID[] = []
	public scrolledToBottom = true

	constructor(public store: RoomStateStore) {}

	scrollToBottom = () => {
		if (this.scrolledToBottom) {
			this.timelineBottomRef.current?.scrollIntoView()
		}
	}

	appendMentionToComposer = (evt: React.MouseEvent<HTMLSpanElement>) => {
		const targetUser = evt.currentTarget.getAttribute("data-target-user")
		if (!targetUser) {
			return
		}
		const targetUserName = evt.currentTarget.innerText
		this.insertText(`[${escapeMarkdown(targetUserName)}](https://matrix.to/#/${encodeURIComponent(targetUser)}) `)
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
