// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Nexus Nicholson
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
import { TombstoneEventContent } from "@/api/types"
import EventContentProps from "./props.ts"

const RoomTombstoneBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as TombstoneEventContent
	const end = content.body?.length > 0 ? ` with the message: ${content.body}` : "."
	const onClick = (e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
		e.preventDefault()
		window.mainScreenContext.setActiveRoom(content.replacement_room)
	}
	let description: JSX.Element
	if (content.replacement_room?.length && content.replacement_room.startsWith("!")) {
		description = (
			<span>
			replaced this room with&nbsp;
				<a onClick={onClick} href={"matrix:roomid/" + content.replacement_room.slice(1)}
				   className="hicli-matrix-uri">
					{content.replacement_room}
				</a>{end}
			</span>
		)
	} else {
		description = <span>shut down this room{end}</span>
	}
	return <div className="room-tombstone-body">
		{sender?.content.displayname ?? event.sender} {description}
	</div>
}

export default RoomTombstoneBody
