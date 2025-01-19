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
import { use, useEffect, useState } from "react"
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import { MemDBEvent, MemberEventContent, Membership } from "@/api/types"
import { ModalContext } from "@/ui/modal"
import { RoomContext } from "@/ui/roomview/roomcontext.ts"
import ConfirmWithMessageModal from "@/ui/timeline/menu/ConfirmWithMessageModal.tsx"
import { getPowerLevels } from "@/ui/timeline/menu/util.ts"
import Block from "@/icons/block.svg?react"
import Gavel from "@/icons/gavel.svg?react"
import PersonAdd from "@/icons/person-add.svg?react"
import PersonRemove from "@/icons/person-remove.svg?react"

interface UserModerationProps {
	userID: string;
	client: Client;
	room?: RoomStateStore;
	member: MemDBEvent | null;
}

interface IgnoredUsersType {
	ignored_users: Record<string, object>;
}

const UserIgnoreButton = ({ userID, client }: { userID: string; client: Client }) => {
	const [ignoredUsers, setIgnoredUsers] = useState<IgnoredUsersType | null>(null)
	useEffect(() => {
		const data = client.store.accountData.get("m.ignored_user_list")
		if (data) {
			setIgnoredUsers(data as IgnoredUsersType)
		}
	}, [client.store.accountData])

	const isIgnored = ignoredUsers?.ignored_users[userID]
	const ignoreUser = () => {
		const newIgnoredUsers = { ...(ignoredUsers || { ignored_users: {}}) }
		newIgnoredUsers.ignored_users[userID] = {}
		client.rpc.setAccountData("m.ignored_user_list", newIgnoredUsers).then(() => {
			setIgnoredUsers(newIgnoredUsers)
		}).catch((e) => {
			console.error("Failed to ignore user", e)
		})
	}
	const unignoreUser = () => {
		const newIgnoredUsers = { ...(ignoredUsers || { ignored_users: {}}) }
		delete newIgnoredUsers.ignored_users[userID]
		client.rpc.setAccountData("m.ignored_user_list", newIgnoredUsers).then(() => {
			setIgnoredUsers(newIgnoredUsers)
		}).catch((e) => {
			console.error("Failed to unignore user", e)
		})
	}

	return (
		<button
			className={"moderation-actions " + (isIgnored ? "positive" : "dangerous")}
			onClick={isIgnored ? unignoreUser : ignoreUser}>
			<Block/>
			<span>{isIgnored ? "Unignore" : "Ignore"}</span>
		</button>
	)
}

const UserModeration = ({ userID, client, member }: UserModerationProps) => {
	const roomCtx = use(RoomContext)
	const openModal = use(ModalContext)
	const hasPl = (action: "invite" | "kick" | "ban") => {
		if(!roomCtx) {
			return false  // no room context
		}
		const [pls, ownPL] = getPowerLevels(roomCtx.store, client)
		const actionPL = pls[action] ?? pls.state_default ?? 50
		const otherUserPl = pls.users?.[userID] ?? pls.users_default ?? 0
		if(action === "invite") {
			return ownPL >= actionPL  // no need to check otherUserPl
		}
		return ownPL >= actionPL && ownPL > otherUserPl
	}

	const runAction = (mode: Membership) => {
		const callback = (reason: string) => {
			if (!roomCtx?.store.roomID) {
				console.error("Cannot action user without a room")
				return
			}
			const payload: MemberEventContent = {
				membership: mode,
			}
			if (reason) {
				payload["reason"] = reason
			}
			client.rpc
				.setState(roomCtx?.store.roomID, "m.room.member", userID, payload)
				.then(() => {
					console.debug("Actioned", userID)
				})
				.catch((e) => {
					console.error("Failed to action", e)
				})
		}
		return () => {
			openModal({
				dimmed: true,
				boxed: true,
				innerBoxClass: "confirm-message-modal",
				content: (
					<RoomContext value={roomCtx}>
						<ConfirmWithMessageModal
							title={`${mode.charAt(0).toUpperCase() + mode.slice(1)} User`}
							description={`Are you sure you want to ${mode} this user?`}
							placeholder="Optional reason"
							confirmButton={mode.charAt(0).toUpperCase() + mode.slice(1)}
							onConfirm={callback}
						/>
					</RoomContext>
				),
			})
		}
	}
	const membership = member?.content.membership || "leave"

	return (
		<div className="user-moderation">
			<h4>Moderation</h4>
			<div className="moderation-actions">
				{roomCtx && (["knock", "leave"].includes(membership) || !member) && hasPl("invite") && (
					<button className="moderation-action positive" onClick={runAction("invite")}>
						<PersonAdd />
						<span>{membership === "knock" ? "Accept request to join" : "Invite"}</span>
					</button>
				)}
				{roomCtx && ["knock", "invite"].includes(membership) && hasPl("kick") && (
					<button className="moderation-action dangerous" onClick={runAction("leave")}>
						<PersonRemove />
						<span>{membership === "invite" ? "Revoke invitation" : "Reject join request"}</span>
					</button>
				)}
				{roomCtx && membership === "join" && hasPl("kick") && (
					<button className="moderation-action dangerous" onClick={runAction("leave")}>
						<PersonRemove />
						<span>Kick</span>
					</button>
				)}
				{roomCtx && membership !== "ban" && hasPl("ban") && (
					<button className="moderation-action dangerous" onClick={runAction("ban")}>
						<Gavel />
						<span>Ban</span>
					</button>
				)}
				{roomCtx && membership === "ban" && hasPl("ban") && (
					<button className="moderation-action positive" onClick={runAction("leave")}>
						<Gavel />
						<span>Unban</span>
					</button>
				)}
				<UserIgnoreButton userID={userID} client={client} />
			</div>
		</div>
	)
}
export default UserModeration
