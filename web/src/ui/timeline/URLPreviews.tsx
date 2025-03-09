// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Sumner Evans
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
import React, { use } from "react"
import { getEncryptedMediaURL, getMediaURL } from "@/api/media"
import { RoomStateStore, usePreference } from "@/api/statestore"
import { MemDBEvent, URLPreview } from "@/api/types"
import { ImageContainerSize, calculateMediaSize } from "@/util/mediasize"
import ClientContext from "../ClientContext"
import { LightboxContext } from "../modal"
import "./URLPreviews.css"

const URLPreviews = ({ event, room }: {
	room: RoomStateStore
	event: MemDBEvent
}) => {
	const client = use(ClientContext)!
	const renderPreviews = usePreference(client.store, room, "render_url_previews")
	// TODO support blurhashes and clicking to view image previews here?
	const showPreviewImages = usePreference(client.store, room, "show_media_previews")
	if (event.redacted_by || !renderPreviews) {
		return null
	}

	const previews = (event.content["com.beeper.linkpreviews"] ?? event.content["m.url_previews"]) as URLPreview[]
	if (!previews || !previews.length) {
		return null
	}
	return <div className="url-previews">
		{previews
			.filter(p => p["og:title"] || p["og:image"] || p["beeper:image:encryption"])
			.map(p => {
				const mediaURL = p["beeper:image:encryption"]
					? getEncryptedMediaURL(p["beeper:image:encryption"].url)
					: getMediaURL(p["og:image"])
				const aspectRatio = (p["og:image:width"] ?? 1) / (p["og:image:height"] ?? 1)
				let containerSize: ImageContainerSize | undefined
				let inline = false
				if (aspectRatio < 1.2) {
					containerSize = { width: 80, height: 80 }
					inline = true
				}
				const style = calculateMediaSize(p["og:image:width"], p["og:image:height"], containerSize)

				const url = p["og:url"] ?? p.matched_url
				const title = p["og:title"] ?? p["og:url"] ?? url
				const mediaContainer = <div className="media-container" style={style.container}>
					<img
						loading="lazy"
						style={style.media}
						src={mediaURL}
						onClick={use(LightboxContext)!}
						alt=""
					/>
				</div>
				return <div
					key={url}
					className={inline ? "url-preview inline" : "url-preview"}
					style={inline ? {} : { width: style.container.width }}
				>
					<div className="title">
						<a href={url} title={title} target="_blank" rel="noreferrer noopener">{title}</a>
					</div>
					<div className="description" title={p["og:description"]}>{p["og:description"]}</div>
					{mediaURL && showPreviewImages && (inline
						? <div className="inline-media-wrapper">{mediaContainer}</div>
						: mediaContainer)}
				</div>
			})}
	</div>
}

export default React.memo(URLPreviews)
