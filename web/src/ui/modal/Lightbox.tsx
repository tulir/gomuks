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
import React, { Component, createContext, createRef, useCallback, useLayoutEffect, useState } from "react"
import { keyToString } from "../keybindings.ts"
import CloseIcon from "@/icons/close.svg?react"
import DownloadIcon from "@/icons/download.svg?react"
import RotateLeftIcon from "@/icons/rotate-left.svg?react"
import RotateRightIcon from "@/icons/rotate-right.svg?react"
import ZoomInIcon from "@/icons/zoom-in.svg?react"
import ZoomOutIcon from "@/icons/zoom-out.svg?react"
import "./Lightbox.css"

const isTouchDevice = window.ontouchstart !== undefined

export interface LightboxParams {
	src: string
	alt: string
}

export type OpenLightboxType = (params: LightboxParams | React.MouseEvent<HTMLImageElement>) => void

export const LightboxContext = createContext<OpenLightboxType>(() =>
	console.error("Tried to open lightbox without being inside context"))

export const LightboxWrapper = ({ children }: { children: React.ReactNode }) => {
	const [params, setParams] = useState<LightboxParams | null>(null)
	const onOpen = useCallback((params: LightboxParams | React.MouseEvent<HTMLImageElement>) => {
		if ((params as React.MouseEvent).target) {
			const evt = params as React.MouseEvent<HTMLImageElement>
			const target = evt.currentTarget as HTMLImageElement
			if (!target.src) {
				return
			}
			setParams({
				src: target.src,
				alt: target.alt,
			})
		} else {
			setParams(params as LightboxParams)
		}
	}, [])
	useLayoutEffect(() => {
		window.openLightbox = onOpen
	}, [onOpen])
	const onClose = useCallback(() => setParams(null), [])
	return <>
		<LightboxContext value={onOpen}>
			{children}
		</LightboxContext>
		{params && <Lightbox {...params} onClose={onClose}/>}
	</>
}

export interface LightboxProps extends LightboxParams {
	onClose: () => void
}

export class Lightbox extends Component<LightboxProps> {
	translate = { x: 0, y: 0 }
	zoom = 1
	rotate = 0
	maybePanning = false
	readonly ref = createRef<HTMLImageElement>()
	readonly wrapperRef = createRef<HTMLDivElement>()

	get style() {
		return {
			translate: `${this.translate.x}px ${this.translate.y}px`,
			rotate: `${this.rotate}deg`,
			scale: `${this.zoom}`,
		}
	}

	close = () => {
		this.translate = { x: 0, y: 0 }
		this.rotate = 0
		this.zoom = 1
		this.props.onClose()
	}

	onClick = () => {
		if (!this.ref.current) {
			return
		}
		if (this.ref.current.style.cursor === "grabbing") {
			this.ref.current.style.cursor = "auto"
			this.maybePanning = false
		} else {
			this.close()
		}
	}

	onWheel = (evt: React.WheelEvent) => {
		if (!this.ref.current) {
			return
		}
		evt.preventDefault()
		const oldZoom = this.zoom
		const delta = -evt.deltaY / 1000
		const newDelta = this.zoom + delta * this.zoom
		this.zoom = Math.min(Math.max(newDelta, 0.01), 10)
		const zoomDelta = this.zoom - oldZoom
		this.translate.x += zoomDelta * (this.ref.current.clientWidth / 2 - evt.nativeEvent.offsetX)
		this.translate.y += zoomDelta * (this.ref.current.clientHeight / 2 - evt.nativeEvent.offsetY)
		const style = this.style
		this.ref.current.style.translate = style.translate
		this.ref.current.style.scale = style.scale
	}

	onMouseDown = (evt: React.MouseEvent) => {
		if (evt.buttons === 1) {
			evt.preventDefault()
			evt.stopPropagation()
			this.maybePanning = true
		}
	}

	onMouseMove = (evt: React.MouseEvent) => {
		if (!this.ref.current) {
			return
		}
		if (evt.buttons !== 1 || !this.maybePanning) {
			this.maybePanning = false
			return
		}
		evt.preventDefault()
		this.translate.x += evt.movementX
		this.translate.y += evt.movementY
		this.ref.current.style.translate = this.style.translate
		this.ref.current.style.cursor = "grabbing"
	}

	onKeyDown = (evt: React.KeyboardEvent<HTMLDivElement>) => {
		const key = keyToString(evt)
		if (key === "Escape") {
			this.close()
		}
		evt.stopPropagation()
	}

	transformer = (callback: () => void) => (evt: React.MouseEvent) => {
		evt.stopPropagation()
		if (!this.ref.current) {
			return
		}
		callback()
		const style = this.style
		this.ref.current.style.rotate = style.rotate
		this.ref.current.style.scale = style.scale
	}

	componentDidMount() {
		if (
			this.wrapperRef.current
			&& (!document.activeElement || !this.wrapperRef.current.contains(document.activeElement))
		) {
			this.wrapperRef.current.focus()
		}
	}

	stopPropagation = (evt: React.MouseEvent) => evt.stopPropagation()
	zoomIn = this.transformer(() => this.zoom = Math.min(this.zoom * 1.1, 10))
	zoomOut = this.transformer(() => this.zoom = Math.max(this.zoom / 1.1, 0.01))
	rotateLeft = this.transformer(() => this.rotate -= 90)
	rotateRight = this.transformer(() => this.rotate += 90)

	render() {
		return <div
			className="overlay dimmed lightbox"
			onClick={this.onClick}
			onMouseMove={isTouchDevice ? undefined : this.onMouseMove}
			tabIndex={-1}
			onKeyDown={this.onKeyDown}
			ref={this.wrapperRef}
		>
			<div className="controls" onClick={this.stopPropagation}>
				<button onClick={this.zoomOut}><ZoomOutIcon/></button>
				<button onClick={this.zoomIn}><ZoomInIcon/></button>
				<button onClick={this.rotateLeft}><RotateLeftIcon/></button>
				<button onClick={this.rotateRight}><RotateRightIcon/></button>
				<a className="button" href={this.props.src} target="_blank" rel="noopener noreferrer">
					<DownloadIcon/>
				</a>
				<button onClick={this.props.onClose}><CloseIcon/></button>
			</div>
			<img
				onMouseDown={isTouchDevice ? undefined : this.onMouseDown}
				onWheel={isTouchDevice ? undefined : this.onWheel}
				src={this.props.src}
				alt={this.props.alt}
				ref={this.ref}
				style={this.style}
				draggable="false"
			/>
		</div>
	}
}
