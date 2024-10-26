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
import React, { JSX, createContext, useCallback, useReducer } from "react"

export interface ModalState {
	content: JSX.Element
	dimmed?: boolean
	wrapperClass?: string
	onClose?: () => void
}

type openModal = (state: ModalState) => void

export const ModalContext = createContext<openModal>(() =>
	console.error("Tried to open modal without being inside context"))

export const ModalCloseContext = createContext<() => void>(() => {})

export const ModalWrapper = ({ children }: { children: React.ReactNode }) => {
	const [state, setState] = useReducer((prevState: ModalState | null, newState: ModalState | null) => {
		prevState?.onClose?.()
		return newState
	}, null)
	const onClickWrapper = useCallback((evt?: React.MouseEvent) => {
		if (evt && evt.target !== evt.currentTarget) {
			return
		}
		setState(null)
	}, [])
	const onKeyWrapper = useCallback((evt: React.KeyboardEvent<HTMLDivElement>) => {
		if (evt.key === "Escape") {
			setState(null)
		}
	}, [])
	return <>
		<ModalContext value={setState}>
			{children}
			{state && <div
				className={`overlay ${state.wrapperClass ?? "modal"} ${state.dimmed ? "dimmed" : ""}`}
				onClick={onClickWrapper}
				onKeyDown={onKeyWrapper}
			>
				<ModalCloseContext value={onClickWrapper}>
					{state.content}
				</ModalCloseContext>
			</div>}
		</ModalContext>
	</>
}
