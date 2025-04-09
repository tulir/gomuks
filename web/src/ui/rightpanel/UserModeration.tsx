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
import { RoomStateStore, useRoomTimeline } from "@/api/statestore"
import { MemDBEvent, MembershipAction } from "@/api/types"
import { useRoomContext } from "@/ui/roomview/roomcontext.ts"
import ConfirmWithMessageModal from "../menu/ConfirmWithMessageModal.tsx"
import { getPowerLevels } from "../menu/util.ts"
import { ModalContext } from "../modal"
import StartDMButton from "./StartDMButton.tsx"
import UserIgnoreButton from "./UserIgnoreButton.tsx"
import DeleteIcon from "@/icons/delete.svg?react"
import BanIcon from "@/icons/gavel.svg?react"
import InviteIcon from "@/icons/person-add.svg?react"
import KickIcon from "@/icons/person-remove.svg?react"

interface UserModerationProps {
	userID: string;
	client: Client;
	room: RoomStateStore | undefined;
	member: MemDBEvent | null;
}

const UserModeration = ({ userID, client, member, room }: UserModerationProps) => {
	const openModal = use(ModalContext)
	const roomCtx = useRoomContext()
	const timeline = useRoomTimeline(roomCtx.store)
	const hasPL = (action: "invite" | "kick" | "ban" | "redact") => {
		if (!room) {
			throw new Error("hasPL called without room")
		}
		const [pls, ownPL] = getPowerLevels(room, client)
		if(action === "invite") {
			return ownPL >= (pls.invite ?? 0)
		}
		const otherUserPL = pls.users?.[userID] ?? pls.users_default ?? 0
		return ownPL >= (pls[action] ?? pls.state_default ?? 50) && ownPL > otherUserPL
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
	const calculateRedactions = () => {
		if (!room) {
			return []
		}
		return timeline.filter(evt => {
			return evt !== null && evt.room_id == room.roomID && evt.sender === userID && !evt.redacted_by
		}) as MemDBEvent[]  // there's no nulls in this one
	}
	const redactRecentMessages = () => {
		if (!room) {
			throw new Error("redactRecentMessages called without room")
		}
		const callback = (reason: string) => {
			const tasks = []
			for (const evt of calculateRedactions()) {
				// for(let i=0;i<3;i++) {
				// 	try {
				// 		await client.rpc.redactEvent(evt.room_id, evt.event_id, reason)
				// 		break
				// 	} catch (e) {
				// 		// If the error is a 429, retry.
				// 		// TODO: get the ratelimit from the error and use it for a more precise sleep
				// 		if (e instanceof Error && e.message.includes("M_LIMIT_EXCEEDED")) {
				// 			const retryAfter = (2 ** i) + Math.random()
				// 			console.warn("Rate limited, retrying in %.2f seconds...", retryAfter)
				// 			await new Promise(resolve => setTimeout(resolve, retryAfter * 1000))
				// 		} else {
				// 			throw e
				// 		}
				// 	}
				// }
				tasks.push(client.rpc.redactEvent(evt.room_id, evt.event_id, reason))
			}
			Promise.all(tasks).catch((e) => {
				console.error(e)
				window.alert(`Failed to redact events: ${e}`)
			})
			return true
		}
		const evtCount = calculateRedactions().length
		return () => {
			openModal({
				dimmed: true,
				boxed: true,
				innerBoxClass: "confirm-message-modal",
				content: <ConfirmWithMessageModal
					title={`Redact recent timeline events of ${userID}`}
					description={
						<>Are you sure you want to redact all currently loaded timeline events
							of <code>{userID}</code>? This will remove approximately {evtCount} events.</>}
					placeholder="Reason (optional)"
					confirmButton={`Redact ~${evtCount} events`}
					onConfirm={callback}
				/>,
			})
		}
	}
	const membership = member?.content.membership || "leave"

	return <div className="user-moderation">
		<h4>Actions</h4>
		{!room || room.meta.current.dm_user_id !== userID ? <StartDMButton userID={userID} client={client} /> : null}
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
		{room && hasPL("redact") && (
			<button className="moderation-action dangerous" onClick={redactRecentMessages()}>
				<DeleteIcon />
				<span>Redact recent messages</span>
			</button>
		)}
		<UserIgnoreButton userID={userID} client={client} />
	</div>
}

export default UserModeration
