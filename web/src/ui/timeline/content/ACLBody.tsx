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
import { JSX } from "react"
import { ACLEventContent } from "@/api/types"
import { listDiff } from "@/util/diff.ts"
import { humanJoin } from "@/util/join.ts"
import { humanJoinReact, joinReact } from "@/util/reactjoin.tsx"
import { ensureArray, ensureStringArray, getDisplayname } from "@/util/validation.ts"
import EventContentProps from "./props.ts"

function joinServers(arr: string[]): JSX.Element[] {
	return humanJoinReact(arr.map(item => <code className="server-name">{item}</code>))
}

function makeACLChangeString(
	addedAllow: string[], removedAllow: string[],
	addedDeny: string[], removedDeny: string[],
	prevAllowIP: boolean, newAllowIP: boolean,
): JSX.Element[] {
	const parts = []
	if (addedDeny.length > 0) {
		parts.push(<>Servers matching {joinServers(addedDeny)} are now banned.</>)
	}
	if (removedDeny.length > 0) {
		parts.push(<>Servers matching {joinServers(removedDeny)} were removed from the ban list.</>)
	}
	if (addedAllow.length > 0) {
		parts.push(<>Servers matching {joinServers(addedAllow)} are now allowed.</>)
	}
	if (removedAllow.length > 0) {
		parts.push(<>Servers matching {joinServers(removedAllow)} were removed from the allowed list.</>)
	}
	if (prevAllowIP !== newAllowIP) {
		parts.push(
			<>Participating from a server using an IP literal hostname is now {newAllowIP ? "allowed" : "banned"}.</>,
		)
	}
	return joinReact(parts)
}

function makeACLChangeStringSummary(
	addedAllow: string[], removedAllow: string[],
	addedDeny: string[], removedDeny: string[],
	prevAllowIP: boolean, newAllowIP: boolean,
): string {
	const pluralEntryCount = (list: string[]) => `${list.length} ${list.length > 1 ? "entries" : "entry"}`
	const parts = []
	if (addedDeny.length > 0) {
		parts.push(`added ${pluralEntryCount(addedDeny)} to the ban list`)
	}
	if (removedDeny.length > 0) {
		parts.push(`removed ${pluralEntryCount(removedDeny)} from the ban list`)
	}
	if (addedAllow.length > 0) {
		parts.push(`added ${pluralEntryCount(addedAllow)} to the allow list`)
	}
	if (removedAllow.length > 0) {
		parts.push(`removed ${pluralEntryCount(removedAllow)} from the allow list`)
	}
	if (prevAllowIP !== newAllowIP) {
		parts.push(
			`${newAllowIP ? "allowed" : "banned"} participating from a server using an IP literal hostname`)
	}
	return humanJoin(parts)
}

const ACLBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as ACLEventContent
	const prevContent = event.unsigned.prev_content as ACLEventContent | undefined
	const [addedAllow, removedAllow] = listDiff(ensureStringArray(content.allow), ensureStringArray(prevContent?.allow))
	const [addedDeny, removedDeny] = listDiff(ensureStringArray(content.deny), ensureStringArray(prevContent?.deny))
	const prevAllowIP = prevContent?.allow_ip_literals ?? true
	const newAllowIP = content.allow_ip_literals ?? true
	if (
		prevAllowIP === newAllowIP
		&& !addedAllow.length && !removedAllow.length
		&& !addedDeny.length && !removedDeny.length
	) {
		return <div className="acl-body">
			{getDisplayname(event.sender, sender?.content)} sent a server ACL event with no changes
		</div>
	}
	if (ensureArray(content.allow).length === 0 || ensureArray(content.deny).includes("*")) {
		return <div className="acl-body">
			{getDisplayname(event.sender, sender?.content)} changed the server ACLs:
			🎉 All servers are banned from participating! This room can no longer be used.
		</div>
	}
	const changeString = makeACLChangeString(addedAllow, removedAllow, addedDeny, removedDeny, prevAllowIP, newAllowIP)
	const changeStringSummary = makeACLChangeStringSummary(
		addedAllow, removedAllow, addedDeny, removedDeny, prevAllowIP, newAllowIP)
	return <div className="acl-body">
		<details>
			<summary>
				{getDisplayname(event.sender, sender?.content)} {changeStringSummary}
			</summary>
			{changeString}
		</details>
	</div>
}

export default ACLBody
