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
import React, { CSSProperties, JSX, use, useState } from "react"
import { getEncryptedMediaURL, getMediaURL } from "@/api/media.ts"
import type { EventType, MediaMessageEventContent } from "@/api/types"
import { ImageContainerSize, calculateMediaSize, defaultVideoContainerSize } from "@/util/mediasize.ts"
import { ensureString } from "@/util/validation.ts"
import { LightboxContext } from "../../modal"
import DownloadIcon from "@/icons/download.svg?react"

export const useMediaContent = (
	content: MediaMessageEventContent,
	evtType: EventType,
	containerSize?: ImageContainerSize,
	onLoad?: () => void,
): [JSX.Element | null, string, CSSProperties] => {
	const mediaURL = content.file?.url ? getEncryptedMediaURL(content.file.url) : getMediaURL(content.url)
	const thumbnailURL = content.info?.thumbnail_file?.url
		? getEncryptedMediaURL(content.info.thumbnail_file.url) : getMediaURL(content.info?.thumbnail_url)
	const [errored, setErrored] = useState(false)
	if (content.msgtype === "m.image" || content.msgtype === "m.sticker" || evtType === "m.sticker") {
		const style = calculateMediaSize(content.info?.w, content.info?.h, containerSize)
		return [<img
			onLoad={onLoad}
			onError={() => {
				setErrored(true)
				onLoad?.()
			}}
			loading="lazy"
			style={style.media}
			src={mediaURL}
			alt={ensureString(content.filename ?? content.body)}
			title={ensureString(content.filename ?? content.body)}
			onClick={use(LightboxContext)}
			className={errored ? "errored" : undefined}
		/>, "image-container", style.container]
	} else if (content.msgtype === "m.video") {
		const style = calculateMediaSize(content.info?.w, content.info?.h, containerSize ?? defaultVideoContainerSize)
		// TODO optionally allow autoplaying gifs
		const autoplay = false // !!content.info?.["fi.mau.autoplay"]
		const controls = !content.info?.["fi.mau.hide_controls"]
		const loop = !!content.info?.["fi.mau.loop"]
		const muted = !!content.info?.["fi.mau.no_audio"]
		let onMouseOver: React.MouseEventHandler<HTMLVideoElement> | undefined
		let onMouseOut: React.MouseEventHandler<HTMLVideoElement> | undefined
		if (!autoplay && !controls && muted) {
			onMouseOver = (event: React.MouseEvent<HTMLVideoElement>) => event.currentTarget.play()
			onMouseOut = (event: React.MouseEvent<HTMLVideoElement>) => {
				event.currentTarget.pause()
				event.currentTarget.currentTime = 0
			}
		}
		return [<video
			autoPlay={autoplay && muted}
			controls={controls || !muted}
			style={style.media}
			loop={loop}
			muted={muted}
			poster={thumbnailURL}
			onMouseOver={onMouseOver}
			onMouseOut={onMouseOut}
			preload="none"
		>
			<source src={mediaURL} type={ensureString(content.info?.mimetype)}/>
		</video>, "video-container", style.container]
	} else if (content.msgtype === "m.audio") {
		return [<audio controls src={mediaURL} preload="none"/>, "audio-container", {}]
	} else if (content.msgtype === "m.file") {
		return [<a
			href={mediaURL}
			target="_blank"
			rel="noopener noreferrer"
			download={ensureString(content.filename ?? content.body)}
		>
			<DownloadIcon height={32} width={32}/> {ensureString(content.filename ?? content.body)}
		</a>, "file-container", {}]
	}
	return [null, "unknown-container", {}]
}
