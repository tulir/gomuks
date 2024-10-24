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
import { MemberEventContent, UserID } from "./types"

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

const fallbackColors = [
	"#a4041d", "#9b2200", "#803f00", "#005f00",
	"#005c45", "#00548c", "#064ab1", "#5d26cd",
	"#822198", "#9f0850",
]

// note: this should stay in sync with fallbackAvatarTemplate in cmd/gomuks.media.go
function makeFallbackAvatar(backgroundColor: string, fallbackCharacter: string): string {
	return "data:image/svg+xml," + encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000">
  <circle cx="500" cy="500" r="500" fill="${backgroundColor}"/>
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

export const getAvatarURL = (userID: UserID, content?: Partial<MemberEventContent>): string | undefined => {
	const fallbackCharacter = (content?.displayname?.[0]?.toUpperCase() ?? userID[1].toUpperCase()).toWellFormed()
	const charCodeSum = userID.split("").reduce((acc, char) => acc + char.charCodeAt(0), 0)
	const backgroundColor = fallbackColors[charCodeSum % fallbackColors.length]
	const [server, mediaID] = parseMXC(content?.avatar_url)
	if (!mediaID) {
		return makeFallbackAvatar(backgroundColor, fallbackCharacter)
	}
	const fallback = `${backgroundColor}:${fallbackCharacter}`
	return `_gomuks/media/${server}/${mediaID}?encrypted=false&fallback=${encodeURIComponent(fallback)}`
}
