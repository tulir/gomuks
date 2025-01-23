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
import { use } from "react"
import Client from "@/api/client.ts"
import { RoomStateStore, useAccountData } from "@/api/statestore"
import { IgnoredUsersEventContent, MemDBEvent, MembershipAction } from "@/api/types"
import { ModalContext } from "../modal"
import ConfirmWithMessageModal from "../timeline/menu/ConfirmWithMessageModal.tsx"
import { getPowerLevels } from "../timeline/menu/util.ts"
import IgnoreIcon from "@/icons/block.svg?react"
import BanIcon from "@/icons/gavel.svg?react"
import InviteIcon from "@/icons/person-add.svg?react"
import KickIcon from "@/icons/person-remove.svg?react"

interface UserModerationProps {
	userID: string;
	client: Client;
	room: RoomStateStore | undefined;
	member: MemDBEvent | null;
}

const UserIgnoreButton = ({ userID, client }: { userID: string; client: Client }) => {
	const ignoredUsers = useAccountData(client.store, "m.ignored_user_list") as IgnoredUsersEventContent | null

	const isIgnored = Boolean(ignoredUsers?.ignored_users?.[userID])
	const ignoreUser = () => {
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

const UserModeration = ({ userID, client, member, room }: UserModerationProps) => {
	const openModal = use(ModalContext)
	const hasPL = (action: "invite" | "kick" | "ban") => {
		if (!room) {
			throw new Error("hasPL called without room")
		}
		const [pls, ownPL] = getPowerLevels(room, client)
		if(action === "invite") {
			return ownPL >= (pls.invite ?? 0)
		}
		const otherUserPL = pls.users?.[userID] ?? pls.users_default ?? 0
		return ownPL >= (pls[action] ?? 50) && ownPL > otherUserPL
	}

	const runAction = (action: MembershipAction) => {
		if (!room) {
			throw new Error("runAction called without room")
		}
		const callback = (reason: string) => {
			client.rpc.setMembership(room.roomID, userID, action, reason).then(
				() => console.debug("Actioned", userID),
				err => {
					console.error("Failed to action", err)
					window.alert(`Failed to ${action} ${userID}: ${err}`)
				},
			)
		}
		const titleCasedAction = action.charAt(0).toUpperCase() + action.slice(1)
		return () => {
			openModal({
				dimmed: true,
				boxed: true,
				innerBoxClass: "confirm-message-modal",
				content: <ConfirmWithMessageModal
					title={`${titleCasedAction} user`}
					description={<>Are you sure you want to {action} <code>{userID}</code>?</>}
					placeholder="Reason (optional)"
					confirmButton={titleCasedAction}
					onConfirm={callback}
				/>,
			})
		}
	}
	const membership = member?.content.membership || "leave"

	return <div className="user-moderation">
		<h4>Moderation</h4>
		{room && (["knock", "leave"].includes(membership) || !member) && hasPL("invite") && (
			<button className="moderation-action positive" onClick={runAction("invite")}>
				<InviteIcon />
				<span>{membership === "knock" ? "Accept join request" : "Invite"}</span>
			</button>
		)}
		{room && ["knock", "invite", "join"].includes(membership) && hasPL("kick") && (
			<button className="moderation-action dangerous" onClick={runAction("kick")}>
				<KickIcon />
				<span>{
					membership === "join"
						? "Kick"
						: membership === "invite"
							? "Revoke invitation"
							: "Reject join request"
				}</span>
			</button>
		)}
		{room && membership !== "ban" && hasPL("ban") && (
			<button className="moderation-action dangerous" onClick={runAction("ban")}>
				<BanIcon />
				<span>Ban</span>
			</button>
		)}
		{room && membership === "ban" && hasPL("ban") && (
			<button className="moderation-action positive" onClick={runAction("unban")}>
				<BanIcon />
				<span>Unban</span>
			</button>
		)}
		<UserIgnoreButton userID={userID} client={client} />
	</div>
}

export default UserModeration
