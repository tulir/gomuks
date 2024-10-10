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

const imageContainerWidth = 320
const imageContainerHeight = 240
const imageContainerAspectRatio = imageContainerWidth / imageContainerHeight

export interface CalculatedMediaSize {
	container: CSSProperties
	media: CSSProperties
}

export function calculateMediaSize(width?: number, height?: number): CalculatedMediaSize {
	if (!width || !height) {
		return {
			container: {
				width: `${imageContainerWidth}px`,
				height: `${imageContainerHeight}px`,
			},
			media: {},
		}
	}
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
		},
		media: {
			aspectRatio: `${origWidth} / ${origHeight}`,
		},
	}
}
