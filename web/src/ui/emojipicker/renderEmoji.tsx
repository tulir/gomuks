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
import { JSX } from "react"
import { getMediaURL } from "@/api/media.ts"
import { Emoji } from "@/util/emoji"

export default function renderEmoji(emoji: Emoji): JSX.Element | string {
	if (emoji.u.startsWith("mxc://")) {
		return <img loading="lazy" src={getMediaURL(emoji.u)} alt={`:${emoji.n}:`}/>
	}
	return emoji.u
}
