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
import { ContentURI, EventID, RoomAlias, RoomID, UserID, UserProfile } from "@/api/types"

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

export interface ParsedMatrixURI {
	identifier: UserID | RoomID | RoomAlias
	eventID?: EventID
	params: URLSearchParams
}

export function parseMatrixURI(uri: unknown): ParsedMatrixURI | undefined {
	if (typeof uri !== "string") {
		return
	}
	let parsed: URL
	try {
		parsed = new URL(uri)
	} catch {
		return
	}
	if (parsed.protocol !== "matrix:") {
		return
	}
	const [type, ident1, subtype, ident2] = parsed.pathname.split("/")
	const output: Partial<ParsedMatrixURI> = {
		params: parsed.searchParams,
	}
	if (type === "u") {
		output.identifier = `@${decodeURIComponent(ident1)}`
	} else if (type === "r") {
		output.identifier = `#${decodeURIComponent(ident1)}`
	} else if (type === "roomid") {
		output.identifier = `!${decodeURIComponent(ident1)}`
		if (subtype === "e") {
			output.eventID = `$${decodeURIComponent(ident2)}`
		}
	} else {
		return
	}
	return output as ParsedMatrixURI
}

export function getLocalpart(userID: UserID): string {
	const idx = userID.indexOf(":")
	return idx > 0 ? userID.slice(1, idx) : userID.slice(1)
}

export function getServerName(userID: UserID): string {
	const idx = userID.indexOf(":")
	return userID.slice(idx+1)
}

export function getDisplayname(userID: UserID, profile?: UserProfile | null): string {
	return ensureString(profile?.displayname) || getLocalpart(userID)
}

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

export function ensureNumber(value: unknown): number {
	if (typeof value !== "number" || isNaN(value)) {
		return 0
	}
	return value
}

export function ensureString(value: unknown): string {
	if (typeof value !== "string") {
		return ""
	}
	return value
}

export function ensureArray(val: unknown): unknown[] {
	return Array.isArray(val) ? val : []
}

export function isString(val: unknown): val is string {
	return typeof val === "string"
}

export function ensureStringArray(val: unknown): string[] {
	return ensureTypedArray(val, isString)
}

export function ensureTypedArray<T>(val: unknown, isCorrectType: (val: unknown) => val is T): T[] {
	if (!Array.isArray(val)) {
		return []
	}
	// Check all items first, don't create a new array if the types are correct
	for (const item of val) {
		if (!isCorrectType(item)) {
			return val.filter(isCorrectType)
		}
	}
	return val
}
