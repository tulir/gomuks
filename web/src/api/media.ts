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
import { parseMXC } from "@/util/validation.ts"
import { ContentURI, RoomID, UserID, UserProfile } from "./types"

export const getMediaURL = (mxc?: string, encrypted: boolean = false): string | undefined => {
	const [server, mediaID] = parseMXC(mxc)
	if (!mediaID) {
		return undefined
	}
	return `_gomuks/media/${server}/${mediaID}?encrypted=${encrypted}`
}

export const getEncryptedMediaURL = (mxc?: string): string | undefined => {
	return getMediaURL(mxc, true)
}

const FALLBACK_COLOR_COUNT = 10

export const getUserColorIndex = (userID: UserID) =>
	userID.split("").reduce((acc, char) => acc + char.charCodeAt(0), 0) % FALLBACK_COLOR_COUNT

function initFallbackColors(): string[] {
	const style = getComputedStyle(document.body)
	const output = []
	for (let i = 0; i < FALLBACK_COLOR_COUNT; i++) {
		output.push(style.getPropertyValue(`--sender-color-${i}`))
	}
	return output
}

let fallbackColors: string[]

export const getUserColor = (userID: UserID) => {
	if (!fallbackColors) {
		fallbackColors = initFallbackColors()
	}
	return fallbackColors[getUserColorIndex(userID)]
}

// note: this should stay in sync with fallbackAvatarTemplate in cmd/gomuks.media.go
function makeFallbackAvatar(backgroundColor: string, fallbackCharacter: string): string {
	return "data:image/svg+xml," + encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000">
  <rect x="0" y="0" width="1000" height="1000" fill="${backgroundColor}"/>
  <text x="500" y="750" text-anchor="middle" fill="#fff" font-weight="bold" font-size="666"
    font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
  >${escapeHTMLChar(fallbackCharacter)}</text>
</svg>`)
}

function escapeHTMLChar(char: string): string {
	switch (char) {
	case "&": return "&amp;"
	case "<": return "&lt;"
	case ">": return "&gt;"
	default: return char
	}
}

function getFallbackCharacter(from: unknown, idx: number): string {
	if (!from || typeof from !== "string" || from.length <= idx) {
		return ""
	}
	// Array.from appears to be the only way to handle Unicode correctly
	return Array.from(from.slice(0, (idx + 1) * 2))[idx]?.toUpperCase().toWellFormed() ?? ""
}

export const getAvatarURL = (
	userID: UserID,
	content?: UserProfile | null,
	thumbnail = false,
	forceFallback = false,
): string | undefined => {
	const fallbackCharacter = getFallbackCharacter(content?.displayname, 0) || getFallbackCharacter(userID, 1)
	const backgroundColor = getUserColor(userID)
	const [server, mediaID] = parseMXC(content?.avatar_file?.url ?? content?.avatar_url)
	if (!mediaID || forceFallback) {
		return makeFallbackAvatar(backgroundColor, fallbackCharacter)
	}
	const encrypted = !!content?.avatar_file
	const fallback = `${backgroundColor}:${fallbackCharacter}`
	const url = `_gomuks/media/${server}/${mediaID}?encrypted=${encrypted}&fallback=${encodeURIComponent(fallback)}`
	return thumbnail ? `${url}&thumbnail=avatar` : url
}

export const getAvatarThumbnailURL = (
	userID: UserID,
	content?: UserProfile | null,
	forceFallback = false,
): string | undefined => {
	return getAvatarURL(userID, content, true, forceFallback)
}

interface RoomForAvatarURL {
	room_id: RoomID
	name?: string
	dm_user_id?: UserID
	avatar?: ContentURI
	avatar_url?: ContentURI
}

export const getRoomAvatarURL = (
	room: RoomForAvatarURL,
	avatarOverride?: ContentURI,
	thumbnail = false,
	forceFallback = false,
): string | undefined => {
	return getAvatarURL(room.dm_user_id ?? room.room_id, {
		displayname: room.name,
		avatar_url: avatarOverride ?? room.avatar ?? room.avatar_url,
	}, thumbnail, forceFallback)
}

export const getRoomAvatarThumbnailURL = (
	room: RoomForAvatarURL,
	avatarOverride?: ContentURI,
	forceFallback = false,
): string | undefined => {
	return getRoomAvatarURL(room, avatarOverride, true, forceFallback)
}
