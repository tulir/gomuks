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
import type { ContentURI, MemberEventContent, UserID } from "@/api/types"
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

export function getAutocompleteMemberList(room: RoomStateStore) {
	const states = room.state.get("m.room.member")
	if (!states) {
		return []
	}
	const output = []
	for (const [stateKey, rowID] of states) {
		const memberEvt = room.eventsByRowID.get(rowID)
		if (!memberEvt) {
			continue
		}
		const content = memberEvt.content as MemberEventContent
		output.push({
			userID: stateKey,
			displayName: content.displayname ?? stateKey,
			avatarURL: content.avatar_url,
			searchString: toSearchableString(`${content.displayname ?? ""}${stateKey.slice(1)}`),
		})
	}
	return output
}

interface filteredUserCache {
	query: string
	result: AutocompleteUser[]
}

export function useFilteredMembers(room: RoomStateStore, query: string): AutocompleteUser[] {
	const allMembers = useMemo(() => getAutocompleteMemberList(room), [room])
	const prev = useRef<filteredUserCache>({ query: "", result: allMembers })
	if (!query) {
		prev.current.query = ""
		prev.current.result = allMembers
	} else if (prev.current.query !== query) {
		prev.current.result = filterAndSort(
			query.startsWith(prev.current.query) ? prev.current.result : allMembers,
			query,
		)
		prev.current.query = query
	}
	return prev.current.result
}
