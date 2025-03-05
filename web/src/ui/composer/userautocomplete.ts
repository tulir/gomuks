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
import { useRef } from "react"
import { AutocompleteMemberEntry, RoomStateStore, useRoomMembers } from "@/api/statestore"
import toSearchableString from "@/util/searchablestring.ts"

export function filterAndSort(users: AutocompleteMemberEntry[], query: string): AutocompleteMemberEntry[] {
	query = toSearchableString(query)
	return users
		.map(user => ({ user, matchIndex: user.searchString.indexOf(query) }))
		.filter(({ matchIndex }) => matchIndex !== -1)
		.sort((e1, e2) => e1.matchIndex - e2.matchIndex)
		.map(({ user }) => user)
}

export function filter(users: AutocompleteMemberEntry[], query: string): AutocompleteMemberEntry[] {
	query = toSearchableString(query)
	return users.filter(user => user.searchString.includes(query))
}

interface filteredUserCache {
	query: string
	result: AutocompleteMemberEntry[]
	slicedResult?: AutocompleteMemberEntry[]
}

export function useFilteredMembers(
	room: RoomStateStore | undefined, query: string, sort = true, slice = true,
): AutocompleteMemberEntry[] {
	const allMembers = useRoomMembers(room)
	const prev = useRef<filteredUserCache>({ query: "", result: allMembers })
	if (!query) {
		prev.current.query = ""
		prev.current.result = allMembers
		prev.current.slicedResult = slice && allMembers.length > 100 ? allMembers.slice(0, 100) : undefined
	} else if (prev.current.query !== query) {
		prev.current.result = (sort ? filterAndSort : filter)(
			query.startsWith(prev.current.query) ? prev.current.result : allMembers,
			query,
		)
		prev.current.slicedResult = prev.current.result.length > 100 && slice
			? prev.current.result.slice(0, 100)
			: undefined
		prev.current.query = query
	}
	return prev.current.slicedResult ?? prev.current.result
}
