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
import React, { CSSProperties, use } from "react"
import { Blurhash } from "react-blurhash"
import { getEncryptedMediaURL, getMediaURL } from "@/api/media.ts"
import type { EventType, MediaMessageEventContent } from "@/api/types"
import { ImageContainerSize, calculateMediaSize } from "@/util/mediasize.ts"
import { LightboxContext } from "../../modal/Lightbox.tsx"
import DownloadIcon from "@/icons/download.svg?react"

export const useMediaContent = (
	content: MediaMessageEventContent, evtType: EventType, containerSize?: ImageContainerSize,
): [React.ReactElement | null, string, CSSProperties] => {
	const imgEl = React.useRef<HTMLImageElement>(null)
	const [loaded, setLoaded] = React.useState(false)
	const onImageLoaded = () => setLoaded(true)

	const blurhashEl = React.useRef<Blurhash>(null)
	const [spoilerShowing, setSpoilerShowing] = React.useState(false)

	React.useEffect(() => {
		const imgElCurrent = imgEl.current

		if (imgElCurrent) {
			imgElCurrent.addEventListener("load", onImageLoaded)
			return () => imgElCurrent.removeEventListener("load", onImageLoaded)
		}
	}, [imgEl])

	const mediaURL = content.file?.url ? getEncryptedMediaURL(content.file.url) : getMediaURL(content.url)
	const thumbnailURL = content.info?.thumbnail_file?.url
		? getEncryptedMediaURL(content.info.thumbnail_file.url) : getMediaURL(content.info?.thumbnail_url)
	if (content.msgtype === "m.image" || evtType === "m.sticker") {
		const style = calculateMediaSize(content.info?.w, content.info?.h, containerSize)

		const blurhash = content.info ? content.info["xyz.amorgan.blurhash"] : undefined
		const isSpoiler = content["m.content_warning"]?.type === "m.spoiler" ||
			content["town.robin.msc3725.content_warning"]?.type === "town.robin.msc3725.spoiler"
		const showPlaceholder = (blurhash && !loaded) || (isSpoiler && !spoilerShowing)

		const mediaStyle = Object.assign(
			style.media,
			showPlaceholder ? { display: "none" } : { display: "inline-block" })

		return [
			<>
				<div
					onClick={() => setSpoilerShowing(!spoilerShowing)}
					style={!showPlaceholder ? { display: "none" } : {}}
					className="placeholder"
				>
					{blurhash && <Blurhash
						ref={blurhashEl}
						hash={blurhash}
						width={style.container.width}
						height={style.container.height}
						resolutionX={48}
						resolutionY={48}
					/>}
					{!blurhash && <div style={style.container}>
					</div>}
					{isSpoiler && !spoilerShowing && <div className="spoiler-indicator">Spoiler</div>}
				</div>
				<img
					ref={imgEl}
					style={mediaStyle}
					src={mediaURL}
					alt={content.filename ?? content.body}
					onClick={use(LightboxContext)}
				/>
			</>
			, "image-container", style.container]
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
			<source src={mediaURL} type={content.info?.mimetype}/>
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
