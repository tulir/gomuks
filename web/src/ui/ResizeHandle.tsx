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
import React, { CSSProperties } from "react"
import useEvent from "@/util/useEvent.ts"
import "./ResizeHandle.css"

export interface ResizeHandleProps {
	width: number
	minWidth: number
	maxWidth: number
	setWidth: (width: number) => void
	className?: string
	style?: CSSProperties
	inverted?: boolean
}

const ResizeHandle = ({ width, minWidth, maxWidth, setWidth, style, className, inverted }: ResizeHandleProps) => {
	const onMouseDown = useEvent((evt: React.MouseEvent<HTMLDivElement>) => {
		const origWidth = width
		const startPos = evt.clientX
		const onMouseMove = (evt: MouseEvent) => {
			let delta = evt.clientX - startPos
			if (inverted) {
				delta = -delta
			}
			setWidth(Math.max(minWidth, Math.min(maxWidth, origWidth + delta)))
			evt.preventDefault()
		}
		const onMouseUp = () => {
			document.removeEventListener("mousemove", onMouseMove)
			document.removeEventListener("mouseup", onMouseUp)
		}
		document.addEventListener("mousemove", onMouseMove)
		document.addEventListener("mouseup", onMouseUp)
		evt.preventDefault()
	})
	return <div className={`resize-handle-outer ${className ?? ""}`} style={style}>
		<div className="resize-handle-inner" onMouseDown={onMouseDown}/>
	</div>
}

export default ResizeHandle
