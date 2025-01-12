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
import { use, useState } from "react"
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import { MemDBEvent, MemberEventContent, Membership } from "@/api/types"
import { ModalContext } from "@/ui/modal"
import { RoomContext } from "@/ui/roomview/roomcontext.ts"
import ConfirmWithMessageModal from "@/ui/timeline/menu/ConfirmWithMessageModal.tsx"
import Gavel from "@/icons/gavel.svg?react"
import PersonAdd from "@/icons/person-add.svg?react"
import PersonRemove from "@/icons/person-remove.svg?react"

interface UserModerationProps {
	userID: string;
	client: Client;
	room?: RoomStateStore;
	member: MemDBEvent | null;
}

const UserModeration = ({ userID, client, member }: UserModerationProps) => {
	const [actionInProgress, setActionInProgress] = useState(false)
	const roomCtx = use(RoomContext)
	const openModal = use(ModalContext)

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
			setActionInProgress(true)
			client.rpc
				.setState(roomCtx?.store.roomID, "m.room.member", userID, payload)
				.then(() => {
					console.debug("Actioned", userID)
					setActionInProgress(false)
				})
				.catch((e) => {
					console.error("Failed to action", e)
					setActionInProgress(false)
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
				{(["knock", "leave"].includes(membership) || !member) && (
					<button
						className="moderation-action invite"
						onClick={runAction("invite")}
						disabled={actionInProgress}
					>
						<PersonAdd />
						<span>Invite</span>
					</button>
				)}
				{["knock", "invite"].includes(membership) && (
					<button
						className="moderation-action dangerous"
						onClick={runAction("leave")}
						disabled={actionInProgress}
					>
						<PersonRemove />
						<span>{membership === "invite" ? "Revoke invitation" : "Reject join request"}</span>
					</button>
				)}
				{membership === "join" && (
					<button
						className="moderation-action dangerous"
						onClick={runAction("leave")}
						disabled={actionInProgress}
					>
						<PersonRemove />
						<span>Kick</span>
					</button>
				)}
				{membership !== "ban" && (
					<button
						className="moderation-action dangerous"
						onClick={runAction("ban")}
						disabled={actionInProgress}
					>
						<Gavel />
						<span>Ban</span>
					</button>
				)}
				{membership === "ban" && (
					<button
						className="moderation-action invite"
						onClick={runAction("leave")}
						disabled={actionInProgress}
					>
						<Gavel />
						<span>Unban</span>
					</button>
				)}
			</div>
		</div>
	)
}
export default UserModeration
