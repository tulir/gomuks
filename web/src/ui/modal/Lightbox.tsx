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
import React, { Component, createRef, useCallback, useLayoutEffect, useState } from "react"
import { keyToString } from "../keybindings.ts"
import { LightboxContext, LightboxParams } from "./contexts.ts"
import CloseIcon from "@/icons/close.svg?react"
import DownloadIcon from "@/icons/download.svg?react"
import RotateLeftIcon from "@/icons/rotate-left.svg?react"
import RotateRightIcon from "@/icons/rotate-right.svg?react"
import ZoomInIcon from "@/icons/zoom-in.svg?react"
import ZoomOutIcon from "@/icons/zoom-out.svg?react"
import "./Lightbox.css"

const isTouchDevice = window.ontouchstart !== undefined

const LightboxWrapper = ({ children }: { children: React.ReactNode }) => {
	const [params, setParams] = useState<LightboxParams | null>(null)
	const onOpen = useCallback((params: LightboxParams | React.MouseEvent<HTMLImageElement>) => {
		if ((params as React.MouseEvent).target) {
			const evt = params as React.MouseEvent<HTMLImageElement>
			const target = evt.currentTarget as HTMLImageElement
			if (!target.src) {
				return
			}
			params = {
				src: target.getAttribute("data-full-src") ?? target.src,
				alt: target.alt,
			}
			setParams(params)
		} else {
			setParams(params as LightboxParams)
		}
		history.pushState({ ...(history.state ?? {}), lightbox: params }, "")
	}, [])
	useLayoutEffect(() => {
		window.openLightbox = onOpen
		const listener = (evt: PopStateEvent) => {
			if (evt.state?.lightbox) {
				setParams(evt.state.lightbox)
			} else {
				setParams(null)
			}
		}
		window.addEventListener("popstate", listener)
		return () => window.removeEventListener("popstate", listener)
	}, [onOpen])
	const onClose = useCallback(() => {
		setParams(null)
		if (params?.src && history.state?.lightbox?.src === params?.src) {
			history.back()
		}
	}, [params])
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

interface Point {
	x: number
	y: number
}

export class Lightbox extends Component<LightboxProps> {
	translate = { x: 0, y: 0 }
	zoom = 1
	rotate = 0
	maybePanning = false
	readonly ref = createRef<HTMLImageElement>()
	readonly wrapperRef = createRef<HTMLDivElement>()
	prevTouch1: Point | null = null
	prevTouch2: Point | null = null
	prevTouchDist: number | null = null

	get style() {
		return {
			translate: `${this.translate.x}px ${this.translate.y}px`,
			rotate: `${this.rotate}deg`,
			scale: `${this.zoom}`,
		}
	}

	get orientation(): number {
		let rot = (this.rotate / 90) % 4
		if (rot < 0) {
			rot += 4
		}
		return rot
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
		this.#doZoom(-evt.deltaY / 1000, evt.nativeEvent.offsetX, evt.nativeEvent.offsetY, false)
		const style = this.style
		this.ref.current.style.translate = style.translate
		this.ref.current.style.scale = style.scale
	}

	#getTouchDistance(p1: Point, p2: Point): number {
		return Math.hypot(p1.x - p2.x, p1.y - p2.y)
	}

	#getTouchMidpoint(p1: Point, p2: Point): Point {
		const contentRect = this.ref.current!.getBoundingClientRect()
		const p1X = p1.x - contentRect.left
		const p1Y = p1.y - contentRect.top
		const p2X = p2.x - contentRect.left
		const p2Y = p2.y - contentRect.top
		const point = {
			x: (p1X + p2X) / 2 / this.zoom,
			y: (p1Y + p2Y) / 2 / this.zoom,
		}
		const orientation = this.orientation
		if (orientation === 1 || orientation === 3) {
			// This is slightly weird because doZoom will flip the x and y values again,
			// but maybe the flipped subtraction from clientWidth/Height is important.
			return { x: point.y, y: point.x }
		}
		return point
	}

	#doZoom(delta: number, offsetX: number, offsetY: number, touch: boolean) {
		if (!this.ref.current) {
			return
		}
		const oldZoom = this.zoom
		const newDelta = oldZoom + delta * this.zoom
		this.zoom = Math.min(Math.max(newDelta, 0.01), 10)
		const zoomDelta = this.zoom - oldZoom

		const orientation = this.orientation
		const negateX = !touch && (orientation === 2 || orientation == 3) ? -1 : 1
		const negateY = !touch && (orientation === 2 || orientation == 1) ? -1 : 1
		const flipXY = orientation === 1 || orientation === 3

		const deltaX = zoomDelta * (this.ref.current.clientWidth / 2 - offsetX) * negateX
		const deltaY = zoomDelta * (this.ref.current.clientHeight / 2 - offsetY) * negateY
		this.translate.x += flipXY ? deltaY : deltaX
		this.translate.y += flipXY ? deltaX : deltaY
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

	onTouchStart = (evt: React.TouchEvent) => {
		if (evt.touches.length === 1) {
			this.maybePanning = true
			this.prevTouch1 = { x: evt.touches[0].pageX, y: evt.touches[0].pageY }
			this.prevTouch2 = null
		} else if (evt.touches.length === 2) {
			this.prevTouch1 = { x: evt.touches[0].pageX, y: evt.touches[0].pageY }
			this.prevTouch2 = { x: evt.touches[1].pageX, y: evt.touches[1].pageY }
			this.prevTouchDist = this.#getTouchDistance(this.prevTouch1, this.prevTouch2)
		} else {
			return
		}
		evt.preventDefault()
		evt.stopPropagation()
	}

	onTouchEnd = () => {
		this.prevTouch1 = null
		this.prevTouch2 = null
		this.prevTouchDist = null
	}

	onTouchMove = (evt: React.TouchEvent) => {
		if (!this.ref.current) {
			return
		}
		if (evt.touches.length > 0 && this.prevTouch1) {
			this.translate.x += evt.touches[0].pageX - this.prevTouch1.x
			this.translate.y += evt.touches[0].pageY - this.prevTouch1.y
			this.prevTouch1 = { x: evt.touches[0].pageX, y: evt.touches[0].pageY }
			if (evt.touches.length === 1) {
				this.ref.current.style.translate = this.style.translate
				this.ref.current.style.cursor = "grabbing"
			}
		}
		if (evt.touches.length > 1 && this.prevTouch1 && this.prevTouch2 && this.prevTouchDist) {
			this.prevTouch2 = { x: evt.touches[1].pageX, y: evt.touches[1].pageY }
			const newDist = this.#getTouchDistance(this.prevTouch1, this.prevTouch2)
			const midpoint = this.#getTouchMidpoint(
				{ x: evt.touches[0].clientX, y: evt.touches[0].clientY },
				{ x: evt.touches[1].clientX, y: evt.touches[1].clientY },
			)
			this.#doZoom((newDist - this.prevTouchDist) / 100, midpoint.x, midpoint.y, true)
			this.prevTouchDist = newDist
			const style = this.style
			this.ref.current.style.translate = style.translate
			this.ref.current.style.scale = style.scale
		}
		evt.preventDefault()
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
			onTouchStart={isTouchDevice ? this.onTouchStart : undefined}
			onTouchMove={isTouchDevice ? this.onTouchMove : undefined}
			onTouchEnd={isTouchDevice ? this.onTouchEnd : undefined}
			onTouchCancel={isTouchDevice ? this.onTouchEnd : undefined}
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

export default LightboxWrapper
