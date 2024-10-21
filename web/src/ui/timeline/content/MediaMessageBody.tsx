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
import { MediaMessageEventContent } from "@/api/types"
import TextMessageBody from "./TextMessageBody.tsx"
import EventContentProps from "./props.ts"
import { useMediaContent } from "./useMediaContent.tsx"

const MediaMessageBody = ({ event, room }: EventContentProps) => {
	const content = event.content as MediaMessageEventContent
	let caption = null
	if (content.body && content.filename && content.body !== content.filename) {
		caption = <TextMessageBody event={event} room={room} />
	}
	const [mediaContent, containerClass, containerStyle] = useMediaContent(content, event.type)
	return <>
		<div className={`media-container ${containerClass}`} style={containerStyle}>
			{mediaContent}
		</div>
		{caption}
	</>
}

export default MediaMessageBody
