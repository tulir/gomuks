// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Nexus Nicholson
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
import { useAccountData } from "@/api/statestore"
import { IgnoredUsersEventContent } from "@/api/types"
import IgnoreIcon from "@/icons/block.svg?react"

const UserIgnoreButton = ({ userID, client }: { userID: string; client: Client }) => {
	const ignoredUsers = useAccountData(client.store, "m.ignored_user_list") as IgnoredUsersEventContent | null

	const isIgnored = Boolean(ignoredUsers?.ignored_users?.[userID])
	const ignoreUser = () => {
		if (!window.confirm(`Are you sure you want to ignore ${userID}?`)) {
			return
		}
		const newIgnoredUsers = { ...(ignoredUsers || { ignored_users: {}}) }
		newIgnoredUsers.ignored_users[userID] = {}
		client.rpc.setAccountData("m.ignored_user_list", newIgnoredUsers).catch(err => {
			console.error("Failed to ignore user", err)
			window.alert(`Failed to ignore ${userID}: ${err}`)
		})
	}
	const unignoreUser = () => {
		const newIgnoredUsers = { ...(ignoredUsers || { ignored_users: {}}) }
		delete newIgnoredUsers.ignored_users[userID]
		client.rpc.setAccountData("m.ignored_user_list", newIgnoredUsers).catch(err => {
			console.error("Failed to unignore user", err)
			window.alert(`Failed to unignore ${userID}: ${err}`)
		})
	}

	return (
		<button
			className={"moderation-action " + (isIgnored ? "positive" : "dangerous")}
			onClick={isIgnored ? unignoreUser : ignoreUser}>
			<IgnoreIcon/>
			<span>{isIgnored ? "Unignore" : "Ignore"}</span>
		</button>
	)
}

export default UserIgnoreButton
