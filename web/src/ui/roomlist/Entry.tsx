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
import { getMediaURL } from "../../api/media.ts"
import type { RoomListEntry } from "../../api/statestore.ts"
import type { DBEvent } from "../../api/types/hitypes.ts"

export interface RoomListEntryProps {
	room: RoomListEntry
	setActiveRoom: (evt: React.MouseEvent) => void
}

function makePreviewText(evt?: DBEvent): string {
	if (!evt) {
		return ""
	}
	if (evt.type === "m.room.message" || evt.type === "m.sticker") {
		// @ts-expect-error TODO add content types
		return evt.content.body
	}
	return ""
}

const Entry = ({ room, setActiveRoom }: RoomListEntryProps) => {
	const previewText = makePreviewText(room.preview_event)
	return <div className="room-entry" onClick={setActiveRoom} data-room-id={room.room_id}>
		<div className="room-entry-left">
			<img loading="lazy" className="room-avatar" src={getMediaURL(room.avatar)} alt=""/>
		</div>
		<div className="room-entry-right">
			<div className="room-name">{room.name}</div>
			{previewText && <div className="message-preview" title={previewText}>{previewText}</div>}
		</div>
	</div>
}

export default Entry