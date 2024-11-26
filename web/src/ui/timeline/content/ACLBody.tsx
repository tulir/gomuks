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
import { Fragment, JSX } from "react"
import { ACLEventContent } from "@/api/types"
import { listDiff } from "@/util/diff.ts"
import { humanJoinReact, joinReact } from "@/util/reactjoin.tsx"
import { ensureArray, ensureStringArray } from "@/util/validation.ts"
import EventContentProps from "./props.ts"

function joinServers(arr: string[]): JSX.Element[] {
	return humanJoinReact(arr.map(item => <code className="server-name">{item}</code>))
}

function makeACLChangeString(
	addedAllow: string[], removedAllow: string[],
	addedDeny: string[], removedDeny: string[],
	prevAllowIP: boolean, newAllowIP: boolean,
) {
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
			{sender?.content.displayname ?? event.sender} sent a server ACL event with no changes
		</div>
	}
	let changeString = makeACLChangeString(addedAllow, removedAllow, addedDeny, removedDeny, prevAllowIP, newAllowIP)
	if (ensureArray(content.allow).length === 0) {
		changeString = [<Fragment key="yay">
			🎉 All servers are banned from participating! This room can no longer be used.
		</Fragment>]
	}
	return <div className="acl-body">
		{sender?.content.displayname ?? event.sender} changed the server ACLs: {changeString}
	</div>
}

export default ACLBody
