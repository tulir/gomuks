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
import { TombstoneEventContent } from "@/api/types"
import EventContentProps from "./props.ts"

const RoomTombstoneBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as TombstoneEventContent
	const end = content.body.length > 0 ? ` with the message: ${content.body}` : "."
	const onClick = () => window.mainScreenContext.setActiveRoom(content.replacement_room)
	const description = (
		<span>
			replaced this room with&nbsp;
			<a onClick={onClick} className="hicli-matrix-uri hicli-matrix-uri-room-alias">
				<span className="room-id">{content.replacement_room}</span>
			</a>{end}
		</span>
	)
	return <div className="room-avatar-body">
		{sender?.content.displayname ?? event.sender} {description}
	</div>
}

export default RoomTombstoneBody
