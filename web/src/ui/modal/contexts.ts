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
import React, { JSX, createContext } from "react"

export interface LightboxParams {
	src: string
	alt: string
}

export type OpenLightboxType = (params: LightboxParams | React.MouseEvent<HTMLImageElement>) => void

export const LightboxContext = createContext<OpenLightboxType>(() =>
	console.error("Tried to open lightbox without being inside context"))

export interface ModalState {
	content: JSX.Element
	dimmed?: boolean
	boxed?: boolean
	boxClass?: string
	innerBoxClass?: string
	onClose?: () => void
	captureInput?: boolean
	noDismiss?: boolean
}

export type openModal = (state: ModalState) => void

export const ModalContext = createContext<openModal>(() =>
	console.error("Tried to open modal without being inside context"))

export const NestableModalContext = createContext<openModal>(() =>
	console.error("Tried to open nestable modal without being inside context"))

export const ModalCloseContext = createContext<() => void>(() => {})
