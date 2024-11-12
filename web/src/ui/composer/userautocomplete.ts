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
import { useMemo, useRef } from "react"
import { RoomStateStore } from "@/api/statestore"
import type { ContentURI, UserID } from "@/api/types"
import toSearchableString from "@/util/searchablestring.ts"

export interface AutocompleteUser {
	userID: UserID
	displayName: string
	avatarURL?: ContentURI
	searchString: string
}

export function filterAndSort(users: AutocompleteUser[], query: string): AutocompleteUser[] {
	query = toSearchableString(query)
	return users
		.map(user => ({ user, matchIndex: user.searchString.indexOf(query) }))
		.filter(({ matchIndex }) => matchIndex !== -1)
		.sort((e1, e2) => e1.matchIndex - e2.matchIndex)
		.map(({ user }) => user)
}

interface filteredUserCache {
	query: string
	result: AutocompleteUser[]
}

export function useFilteredMembers(room: RoomStateStore, query: string): AutocompleteUser[] {
	const allMembers = useMemo(
		() => room.getAutocompleteMembers(),
		// fullMembersLoaded needs to be monitored for when the member list loads
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[room, room.fullMembersLoaded],
	)
	const prev = useRef<filteredUserCache>({ query: "", result: allMembers })
	if (!query) {
		prev.current.query = ""
		prev.current.result = allMembers
	} else if (prev.current.query !== query) {
		prev.current.result = filterAndSort(
			query.startsWith(prev.current.query) ? prev.current.result : allMembers,
			query,
		)
		if (prev.current.result.length > 100) {
			prev.current.result = prev.current.result.slice(0, 100)
		}
		prev.current.query = query
	}
	return prev.current.result
}
