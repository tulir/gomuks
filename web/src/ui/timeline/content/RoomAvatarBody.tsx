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
import { JSX, use } from "react"
import { getRoomAvatarThumbnailURL, getRoomAvatarURL } from "@/api/media.ts"
import { ContentURI, RoomAvatarEventContent } from "@/api/types"
import { ensureString, getDisplayname } from "@/util/validation.ts"
import { LightboxContext } from "../../modal"
import EventContentProps from "./props.ts"

const RoomAvatarBody = ({ event, sender, room }: EventContentProps) => {
	const content = event.content as RoomAvatarEventContent
	const prevContent = event.unsigned.prev_content as RoomAvatarEventContent | undefined
	let changeDescription: JSX.Element | string
	const oldURL = ensureString(prevContent?.url)
	const newURL = ensureString(content.url)
	const openLightbox = use(LightboxContext)!
	const makeAvatar = (url: ContentURI) => <img
		className="small avatar"
		loading="lazy"
		height={16}
		src={getRoomAvatarThumbnailURL(room.meta.current, url)}
		data-full-src={getRoomAvatarURL(room.meta.current, url)}
		onClick={openLightbox}
		alt=""
	/>
	if (oldURL === newURL) {
		changeDescription = "sent a room avatar event with no change"
	} else if (oldURL && newURL) {
		changeDescription = <>changed the room avatar from {makeAvatar(oldURL)} to {makeAvatar(newURL)}</>
	} else if (!oldURL) {
		changeDescription = <>set the room avatar to {makeAvatar(newURL)}</>
	} else {
		changeDescription = "removed the room avatar"
	}
	return <div className="room-avatar-body">
		{getDisplayname(event.sender, sender?.content)} {changeDescription}
	</div>
}

export default RoomAvatarBody
