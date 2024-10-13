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
import { CSSProperties, use, useMemo } from "react"
import sanitizeHtml from "sanitize-html"
import { getEncryptedMediaURL, getMediaURL } from "@/api/media.ts"
import type { EventType, MediaMessageEventContent, MessageEventContent } from "@/api/types"
import { sanitizeHtmlParams } from "@/util/html.ts"
import { calculateMediaSize } from "@/util/mediasize.ts"
import { LightboxContext } from "../../Lightbox.tsx"
import { EventContentProps } from "./props.ts"
import DownloadIcon from "@/icons/download.svg?react"

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

const useMediaContent = (
	content: MediaMessageEventContent, evtType: EventType,
): [React.ReactElement | null, string, CSSProperties] => {
	const mediaURL = content.url ? getMediaURL(content.url) : getEncryptedMediaURL(content.file?.url)
	const thumbnailURL = content.info?.thumbnail_url
		? getMediaURL(content.info.thumbnail_url) : getEncryptedMediaURL(content.info?.thumbnail_file?.url)
	if (content.msgtype === "m.image" || evtType === "m.sticker") {
		const style = calculateMediaSize(content.info?.w, content.info?.h)
		return [<img
			loading="lazy"
			style={style.media}
			src={mediaURL}
			alt={content.filename ?? content.body}
			onClick={use(LightboxContext)}
		/>, "image-container", style.container]
	} else if (content.msgtype === "m.video") {
		const autoplay = false
		const controls = !content.info?.["fi.mau.hide_controls"]
		const loop = !!content.info?.["fi.mau.loop"]
		let onMouseOver: React.MouseEventHandler<HTMLVideoElement> | undefined
		let onMouseOut: React.MouseEventHandler<HTMLVideoElement> | undefined
		if (!autoplay && !controls) {
			onMouseOver = (event: React.MouseEvent<HTMLVideoElement>) => event.currentTarget.play()
			onMouseOut = (event: React.MouseEvent<HTMLVideoElement>) => {
				event.currentTarget.pause()
				event.currentTarget.currentTime = 0
			}
		}
		return [<video
			autoPlay={autoplay}
			controls={controls}
			loop={loop}
			poster={thumbnailURL}
			onMouseOver={onMouseOver}
			onMouseOut={onMouseOut}
			preload="none"
		>
			<source src={mediaURL} type={content.info?.mimetype} />
		</video>, "video-container", {}]
	} else if (content.msgtype === "m.audio") {
		return [<audio controls src={mediaURL} preload="none"/>, "audio-container", {}]
	} else if (content.msgtype === "m.file") {
		return [
			<>
				<a
					href={mediaURL}
					target="_blank"
					rel="noopener noreferrer"
					download={content.filename ?? content.body}
				><DownloadIcon height={32} width={32}/> {content.filename ?? content.body}</a>
			</>,
			"file-container",
			{},
		]
	}
	return [null, "unknown-container", {}]
}

export const MediaMessageBody = ({ event, room }: EventContentProps) => {
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

export const UnknownMessageBody = ({ event }: EventContentProps) => {
	const content = event.content as MessageEventContent
	return <code>{`{ "type": "${event.type}", "content": { "msgtype": "${content.msgtype}" } }`}</code>
}
