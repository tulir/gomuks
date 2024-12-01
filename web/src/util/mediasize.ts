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
import { CSSProperties } from "react"

export interface CalculatedMediaSize {
	container: CSSProperties
	media: CSSProperties
}

export interface ImageContainerSize {
	width: number
	height: number
}

export const defaultImageContainerSize: ImageContainerSize = { width: 320, height: 240 }
export const defaultVideoContainerSize: ImageContainerSize = { width: 400, height: 320 }

export function calculateMediaSize(
	width?: number,
	height?: number,
	imageContainer: ImageContainerSize | undefined = defaultImageContainerSize,
): CalculatedMediaSize {
	const { width: imageContainerWidth, height: imageContainerHeight } = imageContainer ?? defaultImageContainerSize
	if (!width || !height) {
		return {
			container: {
				width: `${imageContainerWidth}px`,
				height: `${imageContainerHeight}px`,
				containIntrinsicWidth: `${imageContainerWidth}px`,
				containIntrinsicHeight: `${imageContainerHeight}px`,
				contentVisibility: "auto",
				contain: "strict",
			},
			media: {},
		}
	}
	const imageContainerAspectRatio = imageContainerWidth / imageContainerHeight

	const origWidth = width
	const origHeight = height
	if (width > imageContainerWidth || height > imageContainerHeight) {
		const aspectRatio = width / height
		if (aspectRatio > imageContainerAspectRatio) {
			width = imageContainerWidth
			height = imageContainerWidth / aspectRatio
		} else if (aspectRatio < imageContainerAspectRatio) {
			width = imageContainerHeight * aspectRatio
			height = imageContainerHeight
		} else {
			width = imageContainerWidth
			height = imageContainerHeight
		}
	}
	return {
		container: {
			width: `${width}px`,
			height: `${height}px`,
			containIntrinsicWidth: `${width}px`,
			containIntrinsicHeight: `${height}px`,
			contentVisibility: "auto",
			contain: "strict",
		},
		media: {
			aspectRatio: `${origWidth} / ${origHeight}`,
		},
	}
}
