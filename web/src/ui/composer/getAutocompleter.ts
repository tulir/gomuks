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
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import {
	AutocompleteQuery,
	AutocompleterProps,
	EmojiAutocompleter,
	RoomAutocompleter,
	UserAutocompleter,
} from "./Autocompleter.tsx"

export function charToAutocompleteType(newChar?: string): AutocompleteQuery["type"] | null {
	switch (newChar) {
	case ":":
		return "emoji"
	case "@":
		return "user"
	case "#":
		return "room"
	default:
		return null
	}
}

export const emojiQueryRegex = /[a-zA-Z0-9_+-]*$/

export function getAutocompleter(
	params: AutocompleteQuery | null, client: Client, room: RoomStateStore,
): React.ElementType<AutocompleterProps> | null {
	switch (params?.type) {
	case "user": {
		const memberCount = room.state.get("m.room.member")?.size ?? 0
		if (memberCount > 500 && params.query.length < 2) {
			return null
		} else if (memberCount > 5000 && params.query.length < 3) {
			return null
		}
		if (!room.membersRequested) {
			room.membersRequested = true
			client.loadRoomState(room.roomID, { omitMembers: false, refetch: false })
		}
		return UserAutocompleter
	}
	case "emoji":
		if (params.query.length < 3) {
			return null
		}
		return EmojiAutocompleter
	case "room":
		if (params.query.length < 3) {
			return null
		}
		return RoomAutocompleter
	default:
		return null
	}
}
