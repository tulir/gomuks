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
import { getMediaURL } from "../../../api/media.ts"
import { ContentURI } from "../../../api/types"
import { sanitizeHtmlParams } from "../../../util/html.ts"
import { calculateMediaSize } from "../../../util/mediasize.ts"
import { LightboxContext } from "../../Lightbox.tsx"
import { EventContentProps } from "./props.ts"

interface BaseMessageEventContent {
	msgtype: string
	body: string
	formatted_body?: string
	format?: "org.matrix.custom.html"
}

interface TextMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.text" | "m.notice" | "m.emote"
}

interface MediaMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.image" | "m.file" | "m.audio" | "m.video"
	filename?: string
	url?: ContentURI
	file?: {
		url: ContentURI
		k: string
		v: "v2"
		ext: true
		alg: "A256CTR"
		key_ops: string[]
		kty: "oct"
	}
	info?: {
		mimetype?: string
		size?: number
		w?: number
		h?: number
		duration?: number
	}
}

interface LocationMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.location"
	geo_uri: string
}

type MessageEventContent = TextMessageEventContent | MediaMessageEventContent | LocationMessageEventContent

const onClickHTML = (evt: React.MouseEvent<HTMLDivElement>) => {
	if ((evt.target as HTMLElement).closest("span[data-mx-spoiler]")?.classList.toggle("spoiler-revealed")) {
		// When unspoilering, don't trigger links and other clickables inside the spoiler
		evt.preventDefault()
	}
}

const MessageBody = ({ event }: EventContentProps) => {
	const content = event.content as MessageEventContent
	if (event.type === "m.sticker") {
		content.msgtype = "m.image"
	}
	const __html = useMemo(() => {
		if (content.format === "org.matrix.custom.html") {
			return sanitizeHtml(content.formatted_body!, sanitizeHtmlParams)
		}
		return undefined
	}, [content.format, content.formatted_body])
	switch (content.msgtype) {
	case "m.text":
	case "m.emote":
	case "m.notice":
		if (__html) {
			return <div onClick={onClickHTML} className="html-body" dangerouslySetInnerHTML={{ __html }}/>
		}
		return content.body
	case "m.image": {
		const openLightbox = use(LightboxContext)
		const style = calculateMediaSize(content.info?.w, content.info?.h)
		let caption = null
		if (__html) {
			caption = <div onClick={onClickHTML} className="html-body" dangerouslySetInnerHTML={{ __html }}/>
		} else if (content.body && content.filename && content.body !== content.filename) {
			caption = content.body
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
	}
	return <code>{`{ "type": "${event.type}", "content": { "msgtype": "${content.msgtype}" } }`}</code>
}

export default MessageBody
