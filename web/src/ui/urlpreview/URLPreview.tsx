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
import { ScaleLoader } from "react-spinners"
import { getEncryptedMediaURL, getMediaURL } from "@/api/media"
import { URLPreview as URLPreviewType } from "@/api/types"
import { ImageContainerSize, calculateMediaSize } from "@/util/mediasize"
import { LightboxContext } from "../modal"
import DeleteIcon from "@/icons/delete.svg?react"
import RefreshIcon from "@/icons/refresh.svg?react"
import "./URLPreview.css"

const URLPreview = ({ url, preview, startLoadingPreview, clearPreview }: {
	url: string,
	preview: URLPreviewType | "awaiting_user" | "loading",
	startLoadingPreview?: () => void,
	clearPreview?: () => void,
}) => {
	if (preview === "awaiting_user" || preview === "loading") {
		return <div key={url} className="url-preview inline"
			title={preview ==="awaiting_user"
				? `Load preview for ${url}?`
				: `Loading preview for ${url}`}
		>
			<div className="title">
				<a href={url} target="_blank" rel="noreferrer noopener">{url}</a>
			</div>
			{preview === "awaiting_user"
				? <div className="load-preview-button">
					<button onClick={startLoadingPreview}><RefreshIcon/> Load Preview</button>
				</div>
				: <div className="loading-preview-indicator">
					<ScaleLoader color="var(--primary-color)"/>
				</div>}
		</div>
	}

	if (!preview["og:title"] && !preview["og:image"] && !preview["beeper:image:encryption"]) {
		return null
	}

	const mediaURL = preview["beeper:image:encryption"]
		? getEncryptedMediaURL(preview["beeper:image:encryption"].url)
		: getMediaURL(preview["og:image"])
	const aspectRatio = (preview["og:image:width"] ?? 1) / (preview["og:image:height"] ?? 1)
	let containerSize: ImageContainerSize | undefined
	let inline = false
	if (aspectRatio < 1.2) {
		containerSize = { width: 80, height: 80 }
		inline = true
	}
	const style = calculateMediaSize(preview["og:image:width"], preview["og:image:height"], containerSize)

	const previewingUrl = preview["og:url"] ?? preview.matched_url ?? url
	const title = preview["og:title"] ?? preview["og:url"] ?? previewingUrl
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
			<a href={previewingUrl} title={title} target="_blank" rel="noreferrer noopener">{title}</a>
		</div>
		{clearPreview && <div className="actions">
			<button onClick={clearPreview}><DeleteIcon/></button>
		</div>}
		<div className="description">{preview["og:description"]}</div>
		{mediaURL && (inline
			? <div className="inline-media-wrapper">{mediaContainer}</div>
			: mediaContainer)}
	</div>
}

export default React.memo(URLPreview)
