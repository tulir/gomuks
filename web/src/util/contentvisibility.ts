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
import { RefObject, useLayoutEffect, useRef, useState } from "react"

export default function useContentVisibilit<T extends HTMLElement>(
	allowRevert = false,
): [boolean, RefObject<T | null>] {
	const ref = useRef<T>(null)
	const [isVisible, setVisible] = useState(false)
	useLayoutEffect(() => {
		const element = ref.current
		if (!element) {
			return
		}
		const listener = (evt: unknown) => {
			if (!(evt as ContentVisibilityAutoStateChangeEvent).skipped) {
				setVisible(true)
			} else if (allowRevert) {
				setVisible(false)
			}
		}
		element.addEventListener("contentvisibilityautostatechange", listener)
		return () => element.removeEventListener("contentvisibilityautostatechange", listener)
	}, [allowRevert])
	return [isVisible, ref]
}
