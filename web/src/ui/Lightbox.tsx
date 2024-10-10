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
import React, { Component, RefObject, createContext, createRef, useCallback, useState } from "react"
import "./Lightbox.css"

const isTouchDevice = window.ontouchstart !== undefined

export interface LightboxParams {
	src: string
	alt: string
}

type openLightbox = (params: LightboxParams | React.MouseEvent<HTMLImageElement>) => void

export const LightboxContext = createContext<openLightbox>(() =>
	console.error("Tried to open lightbox without being inside context"))

export const LightboxWrapper = ({ children }: { children: React.ReactNode }) => {
	const [params, setParams] = useState<LightboxParams | null>(null)
	const onOpen = useCallback((params: LightboxParams | React.MouseEvent<HTMLImageElement>) => {
		if ((params as React.MouseEvent).target) {
			const evt = params as React.MouseEvent<HTMLImageElement>
			const target = evt.currentTarget as HTMLImageElement
			setParams({
				src: target.src,
				alt: target.alt,
			})
		} else {
			setParams(params as LightboxParams)
		}
	}, [])
	const onClose = useCallback(() => setParams(null), [])
	return <>
		<LightboxContext value={onOpen}>
			{children}
		</LightboxContext>
		{params && <Lightbox {...params} onClose={onClose} />}
	</>
}

export interface LightboxProps extends LightboxParams {
	onClose: () => void
}

export class Lightbox extends Component<LightboxProps> {
	transform = { zoom: 1, x: 0, y: 0 }
	maybePanning = false
	readonly ref: RefObject<HTMLImageElement | null>

	constructor(props: LightboxProps) {
		super(props)
		this.ref = createRef<HTMLImageElement>()
	}

	transformString = () => {
		return `translate(${this.transform.x}px, ${this.transform.y}px) scale(${this.transform.zoom})`
	}

	onClick = () => {
		if (!this.ref.current) {
			return
		}
		if (this.ref.current.style.cursor === "grabbing") {
			this.ref.current.style.cursor = "auto"
			this.maybePanning = false
		} else {
			this.transform = { zoom: 1, x: 0, y: 0 }
			this.props.onClose()
		}
	}

	onWheel = (evt: React.WheelEvent) => {
		if (!this.ref.current) {
			return
		}
		evt.preventDefault()
		const oldZoom = this.transform.zoom
		const delta = -evt.deltaY / 1000
		const newDelta = this.transform.zoom + delta * this.transform.zoom
		this.transform.zoom = Math.min(Math.max(newDelta, 0.01), 10)
		const zoomDelta = this.transform.zoom - oldZoom
		this.transform.x += zoomDelta * (this.ref.current.clientWidth / 2 - evt.nativeEvent.offsetX)
		this.transform.y += zoomDelta * (this.ref.current.clientHeight / 2 - evt.nativeEvent.offsetY)
		this.ref.current.style.transform = this.transformString()
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
		this.transform.x += evt.movementX
		this.transform.y += evt.movementY
		this.ref.current.style.transform = this.transformString()
		this.ref.current.style.cursor = "grabbing"
	}

	get style() {
		return {
			transform: this.transformString(),
		}
	}

	render() {
		return <div
			className="overlay lightbox"
			onClick={this.onClick}
			onMouseMove={isTouchDevice ? undefined : this.onMouseMove}
		>
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
