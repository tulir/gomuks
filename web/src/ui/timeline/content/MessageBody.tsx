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
import { use, useMemo } from "react"
import sanitizeHtml from "sanitize-html"
import { getMediaURL } from "@/api/media.ts"
import type { MediaMessageEventContent, MessageEventContent } from "@/api/types"
import { sanitizeHtmlParams } from "@/util/html.ts"
import { calculateMediaSize } from "@/util/mediasize.ts"
import { LightboxContext } from "../../Lightbox.tsx"
import { EventContentProps } from "./props.ts"

const onClickHTML = (evt: React.MouseEvent<HTMLDivElement>) => {
	if ((evt.target as HTMLElement).closest("span[data-mx-spoiler]")?.classList.toggle("spoiler-revealed")) {
		// When unspoilering, don't trigger links and other clickables inside the spoiler
		evt.preventDefault()
	}
}

export const TextMessageBody = ({ event }: EventContentProps) => {
	const content = event.content as MessageEventContent
	const __html = useMemo(() => {
		if (content.format === "org.matrix.custom.html") {
			return sanitizeHtml(content.formatted_body!, sanitizeHtmlParams)
		}
		return undefined
	}, [content.format, content.formatted_body])
	if (__html) {
		return <div onClick={onClickHTML} className="message-text html-body" dangerouslySetInnerHTML={{ __html }}/>
	}
	return <div className="message-text plaintext-body">{content.body}</div>
}

export const MediaMessageBody = ({ event, room }: EventContentProps) => {
	const content = event.content as MediaMessageEventContent
	if (event.type === "m.sticker") {
		content.msgtype = "m.image"
	}
	const openLightbox = use(LightboxContext)
	const style = calculateMediaSize(content.info?.w, content.info?.h)
	let caption = null
	if (content.body && content.filename && content.body !== content.filename) {
		caption = <TextMessageBody event={event} room={room} />
	}
	return <>
		<div className="media-container" style={style.container}>
			<img
				loading="lazy"
				style={style.media}
				src={getMediaURL(content.url ?? content.file?.url)}
				alt={content.filename ?? content.body}
				onClick={openLightbox}
			/>
		</div>
		{caption}
	</>
}

export const UnknownMessageBody = ({ event }: EventContentProps) => {
	const content = event.content as MessageEventContent
	return <code>{`{ "type": "${event.type}", "content": { "msgtype": "${content.msgtype}" } }`}</code>
}
