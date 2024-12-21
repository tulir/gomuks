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
import { useEffect, useRef } from "react"

const useCategoryUnderline = () => {
	const emojiCategoryBarRef = useRef<HTMLDivElement>(null)
	const emojiListRef = useRef<HTMLDivElement>(null)

	useEffect(() => {
		const cats = emojiCategoryBarRef.current
		const lists = emojiListRef.current
		if (!cats || !lists) {
			return
		}
		const observer = new IntersectionObserver(entries => {
			for (const entry of entries) {
				const catID = entry.target.getAttribute("data-category-id")
				const cat = catID && cats.querySelector(`.emoji-category-icon[data-category-id="${catID}"]`)
				if (!cat) {
					continue
				}
				if (entry.isIntersecting) {
					cat.classList.add("visible")
				} else {
					cat.classList.remove("visible")
				}
			}
		}, {
			rootMargin: `-8px 0px 0px 0px`,
			root: lists.parentElement,
		})
		for (const cat of lists.getElementsByClassName("emoji-category")) {
			observer.observe(cat)
		}
		return () => observer.disconnect()
	}, [])

	return [emojiCategoryBarRef, emojiListRef] as const
}

export default useCategoryUnderline
