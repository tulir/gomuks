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
import { JSX } from "react"
import { RoomNameEventContent } from "@/api/types"
import { ensureString, getDisplayname } from "@/util/validation.ts"
import EventContentProps from "./props.ts"

function bidiIsolate(str: string): JSX.Element {
	return <span style={{ unicodeBidi: "isolate" }}>{str}</span>
}

const RoomNameBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as RoomNameEventContent
	const prevContent = event.unsigned.prev_content as RoomNameEventContent | undefined
	let changeDescription: JSX.Element | string
	const oldName = ensureString(prevContent?.name)
	const newName = ensureString(content.name)
	if (oldName === newName) {
		changeDescription = "sent a room name event with no change"
	} else if (oldName && newName) {
		changeDescription = <>changed the room name from {bidiIsolate(oldName)} to {bidiIsolate(newName)}</>
	} else if (!oldName) {
		changeDescription = <>set the room name to {bidiIsolate(newName)}</>
	} else {
		changeDescription = "removed the room name"
	}
	return <div className="room-name-body">
		{getDisplayname(event.sender, sender?.content)} {changeDescription}
	</div>
}

export default RoomNameBody
