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
import { CSSProperties, use } from "react"
import { getEncryptedMediaURL, getMediaURL } from "@/api/media.ts"
import type { EventType, MediaMessageEventContent } from "@/api/types"
import { ImageContainerSize, calculateMediaSize } from "@/util/mediasize.ts"
import { LightboxContext } from "../../modal/Lightbox.tsx"
import DownloadIcon from "@/icons/download.svg?react"

export const useMediaContent = (
	content: MediaMessageEventContent, evtType: EventType, containerSize?: ImageContainerSize,
): [React.ReactElement | null, string, CSSProperties] => {
	const mediaURL = content.file?.url ? getEncryptedMediaURL(content.file.url) : getMediaURL(content.url)
	const thumbnailURL = content.info?.thumbnail_file?.url
		? getEncryptedMediaURL(content.info.thumbnail_file.url) : getMediaURL(content.info?.thumbnail_url)
	if (content.msgtype === "m.image" || evtType === "m.sticker") {
		const style = calculateMediaSize(content.info?.w, content.info?.h, containerSize)
		const classes = ["image-container"]
		if(content["m.spoiler"] === true) {
			classes.push("attachment-spoiler")
		}
		const lightBox = use(LightboxContext)

		const onClick = (event: React.MouseEvent<HTMLImageElement>) => {
			// Check if it is spoilered. If it is, remove the attachment-spoiler class
			if(event.currentTarget.parentElement?.classList.contains("attachment-spoiler")) {
				event.preventDefault()
				event.currentTarget.parentElement?.classList.remove("attachment-spoiler")
			} else {
				return lightBox(event)
			}
		}

		return [<img
			loading="lazy"
			style={style.media}
			src={mediaURL}
			alt={content.filename ?? content.body}
			title={content["m.spoiler.reason"]}
			onClick={onClick}
		/>, classes.join(" "), style.container]
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
		let classes = ["video-container"]
		if(content["m.spoiler"] === true) {
			classes.push("attachment-spoiler")
		}
		const onPlay = (event: React.MouseEvent<HTMLVideoElement>) => {
			// onclick doesn't appear to work for <video> elements, so we use onPlay instead
			if(classes.includes("attachment-spoiler")) {
				event.preventDefault()
				event.currentTarget.pause()  // stop it autoplaying before the spoiler is removed
				classes = classes.filter((c) => c !== "attachment-spoiler")
				event.currentTarget.parentElement?.classList.remove("attachment-spoiler")
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
			title={content["m.spoiler.reason"]}
			onPlay={onPlay}
		>
			<source src={mediaURL} type={content.info?.mimetype}/>
		</video>, classes.join(" "), {}]
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
