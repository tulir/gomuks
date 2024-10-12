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

import { UserID } from "./types"

const mediaRegex = /^mxc:\/\/([a-zA-Z0-9.:-]+)\/([a-zA-Z0-9_-]+)$/

export const getMediaURL = (mxc?: string): string | undefined => {
	if (!mxc) {
		return undefined
	}
	const match = mxc.match(mediaRegex)
	if (!match) {
		return undefined
	}
	return `_gomuks/media/${match[1]}/${match[2]}`
}

export const getAvatarURL = (_userID: UserID, mxc?: string): string | undefined => {
	if (!mxc) {
		return undefined
		// return `_gomuks/avatar/${encodeURIComponent(userID)}`
	}
	const match = mxc.match(mediaRegex)
	if (!match) {
		return undefined
		// return `_gomuks/avatar/${encodeURIComponent(userID)}`
	}
	return `_gomuks/media/${match[1]}/${match[2]}`
	// return `_gomuks/avatar/${encodeURIComponent(userID)}/${match[1]}/${match[2]}`
}
