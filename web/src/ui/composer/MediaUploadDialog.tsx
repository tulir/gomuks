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
import React, { JSX, use, useEffect, useRef, useState } from "react"
import type { MediaEncodingOptions } from "@/api/types"
import { ModalCloseContext } from "@/ui/modal"
import { isMobileDevice } from "@/util/ismobile.ts"
import "./MediaUploadDialog.css"

export interface MediaUploadDialogProps {
	file: File
	blobURL: string
	doUploadFile: (file: BodyInit, filename: string, encodingOpts?: MediaEncodingOptions) => void
}

function formatSize(bytes: number): string {
	const units = ["B", "KiB", "MiB", "GiB", "TiB"]
	let unitIndex = 0
	let size = bytes
	while (size >= 1024 && unitIndex < units.length - 1) {
		size /= 1024
		unitIndex++
	}
	return `${unitIndex === 0 ? size : size.toFixed(2)} ${units[unitIndex]}`
}

const imageReencTargets = ["image/webp", "image/jpeg", "image/png", "image/gif"]
const nonEncodableSources = ["image/bmp", "image/tiff", "image/heif", "image/heic"]
const imageReencSources = [...imageReencTargets, ...nonEncodableSources]
const videoReencTargets = ["video/webm", "video/mp4", "image/webp+anim"]

interface dimensions {
	width: number
	height: number
}

const MediaUploadDialog = ({ file, blobURL, doUploadFile }: MediaUploadDialogProps) => {
	const videoRef = useRef<HTMLVideoElement>(null)
	const [name, setName] = useState(file.name)
	const [reencTarget, setReencTarget] = useState(nonEncodableSources.includes(file.type) ? "image/jpeg" : "")
	const [jpegQuality, setJPEGQuality] = useState(80)
	const [resizeSlider, setResizeSlider] = useState(100)
	const [origDimensions, setOrigDimensions] = useState<dimensions | null>(null)
	const closeModal = use(ModalCloseContext)
	let previewContent: JSX.Element | null = null
	let reencTargets: string[] | null = null
	let resizedWidth: number | undefined = undefined
	let resizedHeight: number | undefined = undefined
	if (origDimensions) {
		resizedWidth = Math.floor(origDimensions.width * (resizeSlider / 100))
		resizedHeight = Math.floor(origDimensions.height * (resizeSlider / 100))
	}
	useEffect(() => {
		if (file.type.startsWith("image/")) {
			createImageBitmap(file).then(res => {
				setOrigDimensions({ width: res.width, height: res.height })
				res.close()
			})
		}
	}, [file, blobURL])
	if (file.type.startsWith("image/")) {
		previewContent = <img src={blobURL} alt={file.name} />
		if (imageReencSources.includes(file.type)) {
			reencTargets = imageReencTargets
		}
	} else if (file.type.startsWith("video/")) {
		const videoMetaLoaded = () => {
			if (videoRef.current) {
				setOrigDimensions({ width: videoRef.current.videoWidth, height: videoRef.current.videoHeight })
			}
		}
		previewContent = <video controls onLoadedMetadata={videoMetaLoaded} ref={videoRef}>
			<source src={blobURL} type={file.type} />
		</video>
		reencTargets = videoReencTargets
	} else if (file.type.startsWith("audio/")) {
		previewContent = <audio controls>
			<source src={blobURL} type={file.type} />
		</audio>
	}
	const submit = (evt: React.FormEvent) => {
		evt.preventDefault()
		doUploadFile(file, name, {
			encode_to: reencTarget || undefined,
			quality: reencTarget === "image/jpeg" ? jpegQuality : undefined,
			resize_width: resizeSlider !== 100 ? resizedWidth : undefined,
			resize_height: resizeSlider !== 100 ? resizedHeight : undefined,
			resize_percent: resizeSlider,
		})
		closeModal()
	}
	return <form className="media-upload-modal" onSubmit={submit}>
		<h3>Upload attachment</h3>
		<div className="attachment-preview">{previewContent}</div>
		<div className="attachment-meta">
			<div className="meta-key">Original type</div>
			<div className="meta-value">{file.type}</div>

			<div className="meta-key">Original size</div>
			<div className="meta-value">{formatSize(file.size)}</div>

			<div className="meta-key">File name</div>
			<div className="meta-value">
				<input
					autoFocus={!isMobileDevice}
					type="text"
					value={name}
					onChange={evt => setName(evt.target.value)}
				/>
			</div>

			<div className="meta-key">{origDimensions ? "Dimensions" : null}</div>
			<div className="meta-value">
				{origDimensions ? `${resizedWidth}×${resizedHeight}` : null}
			</div>

			{reencTargets && <>
				<div className="meta-key">Re-encode</div>
				<div className="meta-value meta-value-long">
					<select value={reencTarget} onChange={evt => {
						setReencTarget(evt.target.value)
						setResizeSlider(100)
					}}>
						<option value="">No re-encoding</option>
						{reencTargets.map(target => <option key={target} value={target}>{target}</option>)}
					</select>
				</div>

				<div className="meta-key">Resize</div>
				<div className="meta-value meta-value-long">
					<input
						type="range"
						min={1}
						max={100}
						value={resizeSlider}
						onChange={evt => {
							setResizeSlider(parseInt(evt.target.value))
							if (reencTarget === "") {
								setReencTarget(reencTargets?.includes(file.type) ? file.type : "image/jpeg")
							}
						}}
					/>
					<span>{resizeSlider}%</span>
				</div>
			</>}

			{(reencTarget === "image/jpeg" || reencTarget === "image/webp") && <>
				<div className="meta-key">Quality</div>
				<div className="meta-value meta-value-long">
					<input
						type="range"
						min={1}
						max={reencTarget === "image/webp" ? 101 : 100}
						value={jpegQuality}
						onChange={evt => setJPEGQuality(parseInt(evt.target.value))}
					/>
					<span>{jpegQuality === 101 ? "Lossless" : `${jpegQuality}%`}</span>
				</div>
			</>}
		</div>
		<div className="confirm-buttons">
			<button className="cancel-button" type="button" onClick={closeModal}>Cancel</button>
			<button className="confirm-button" type="submit">Upload</button>
		</div>
	</form>
}

export default MediaUploadDialog
