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
import { Fragment, JSX } from "react"

export function humanJoinReact(
	arr: (string | JSX.Element)[],
	sep: string | JSX.Element = ", ",
	sep2: string | JSX.Element = " and ",
	lastSep: string | JSX.Element = " and ",
): JSX.Element[] {
	return arr.map((elem, idx) => {
		let separator = sep
		if (idx === arr.length - 2) {
			separator = (arr.length === 2) ? sep2 : lastSep
		}
		return <Fragment key={idx}>
			{elem}
			{idx < arr.length - 1 ? separator : null}
		</Fragment>
	})
}

export const oxfordHumanJoinReact = (arr: (string | JSX.Element)[]) => humanJoinReact(arr, ", ", " and ", ", and ")
export const joinReact = (arr: (string | JSX.Element)[]) => humanJoinReact(arr, " ", " ", " ")
