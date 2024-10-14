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
import { ContentURI, EventID, RoomAlias, RoomID, UserID } from "@/api/types"

const simpleHomeserverRegex = /^[a-zA-Z0-9.:-]+$/
const mediaRegex = /^mxc:\/\/([a-zA-Z0-9.:-]+)\/([a-zA-Z0-9_-]+)$/

function isIdentifier<T>(identifier: unknown, sigil: string, requiresServer: boolean): identifier is T {
	if (typeof identifier !== "string" || !identifier.startsWith(sigil)) {
		return false
	}
	if (requiresServer) {
		const idx = identifier.indexOf(":")
		return idx > 0 && simpleHomeserverRegex.test(identifier.slice(idx+1))
	}
	return true
}

export function validated<T>(value: T | undefined, validator: (value: T) => boolean): value is T {
	return value !== undefined && validator(value)
}

export const isEventID = (eventID: unknown) => isIdentifier<EventID>(eventID, "$", false)
export const isUserID = (userID: unknown) => isIdentifier<UserID>(userID, "@", true)
export const isRoomID = (roomID: unknown) => isIdentifier<RoomID>(roomID, "!", true)
export const isRoomAlias = (roomAlias: unknown) => isIdentifier<RoomAlias>(roomAlias, "#", true)
export const isMXC = (mxc: unknown): mxc is ContentURI => typeof mxc === "string" && mediaRegex.test(mxc)

export function parseMXC(mxc: unknown): [string, string] | [] {
	if (typeof mxc !== "string") {
		return []
	}
	const match = mxc.match(mediaRegex)
	if (!match) {
		return []
	}
	return [match[1], match[2]]
}