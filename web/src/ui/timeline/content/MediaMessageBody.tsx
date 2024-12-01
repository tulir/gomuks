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
import { CSSProperties, JSX, use, useReducer } from "react"
import { Blurhash } from "react-blurhash"
import { GridLoader } from "react-spinners"
import { usePreference } from "@/api/statestore"
import { ContentWarningType, MediaMessageEventContent } from "@/api/types"
import { ensureString } from "@/util/validation.ts"
import ClientContext from "../../ClientContext.ts"
import TextMessageBody from "./TextMessageBody.tsx"
import EventContentProps from "./props.ts"
import { useMediaContent } from "./useMediaContent.tsx"

const loaderSize = (style: CSSProperties): number | undefined => {
	if (!style.width) {
		return
	}
	const width = +(style.width as string).replace("px", "")
	const height = +(style.height as string).replace("px", "")
	// GridLoader takes size of individual bubbles for some reason (so need to divide by 3),
	// and we want the size to be slightly smaller than the container, so just divide by 5
	return Math.min(Math.round(Math.min(width, height) / 5), 30)
}

const switchToTrue = () => true

const MediaMessageBody = ({ event, room, sender }: EventContentProps) => {
	const content = event.content as MediaMessageEventContent
	let caption = null
	if (content.body && content.filename && content.body !== content.filename) {
		caption = <TextMessageBody event={event} room={room} sender={sender} />
	}

	const client = use(ClientContext)!
	const supportsLoadingPlaceholder = event.type === "m.sticker" || content.msgtype === "m.image"
	const supportsClickToShow = supportsLoadingPlaceholder || content.msgtype === "m.video"
	const showPreviewsByDefault = usePreference(client.store, room, "show_media_previews")
	const [loaded, onLoad] = useReducer(switchToTrue, !supportsLoadingPlaceholder)
	const [clickedShow, onClickShow] = useReducer(switchToTrue, false)

	let contentWarning = content["town.robin.msc3725.content_warning"]
	if (content["page.codeberg.everypizza.msc4193.spoiler"]) {
		contentWarning = {
			type: ContentWarningType.Spoiler,
			description: content["page.codeberg.everypizza.msc4193.spoiler.reason"],
		}
	}
	const renderMediaElem = !supportsClickToShow || showPreviewsByDefault || clickedShow
	const renderPlaceholderElem = supportsClickToShow && (!renderMediaElem || !!contentWarning || !loaded)
	const isLoadingOnlyCover = !loaded && !contentWarning && renderMediaElem

	const [mediaContent, containerClass, containerStyle] = useMediaContent(content, event.type, undefined, onLoad)

	let placeholderElem: JSX.Element | null = null
	if (renderPlaceholderElem) {
		const blurhash = ensureString(
			content.info?.["xyz.amorgan.blurhash"] ?? content.info?.thumbnail_info?.["xyz.amorgan.blurhash"],
		)
		placeholderElem = <div
			onClick={onClickShow}
			className="placeholder"
		>
			{(blurhash && containerStyle.width) ? <Blurhash
				hash={blurhash}
				width={containerStyle.width}
				height={containerStyle.height}
				resolutionX={48}
				resolutionY={48}
			/> : <div className="empty-placeholder" style={containerStyle}/>}
			{isLoadingOnlyCover
				? <div className="placeholder-spinner">
					<GridLoader color="var(--primary-color)" size={loaderSize(containerStyle)}/>
				</div>
				: <div className="placeholder-reason">
					{ensureString(contentWarning?.description) || "Show media"}
				</div>}
		</div>
	}

	return <>
		<div className={`media-container ${containerClass}`} style={containerStyle}>
			{placeholderElem}
			{renderMediaElem ? mediaContent : null}
		</div>
		{caption}
	</>
}

export default MediaMessageBody
