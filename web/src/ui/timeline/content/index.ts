import React from "react"
import { BeeperPerMessageProfile, MemDBEvent, MessageEventContent } from "@/api/types"
import ACLBody from "./ACLBody.tsx"
import EncryptedBody from "./EncryptedBody.tsx"
import HiddenEvent from "./HiddenEvent.tsx"
import LocationMessageBody from "./LocationMessageBody.tsx"
import MediaMessageBody from "./MediaMessageBody.tsx"
import MemberBody from "./MemberBody.tsx"
import PinnedEventsBody from "./PinnedEventsBody.tsx"
import PolicyRuleBody from "./PolicyRuleBody.tsx"
import PowerLevelBody from "./PowerLevelBody.tsx"
import RedactedBody from "./RedactedBody.tsx"
import RoomAvatarBody from "./RoomAvatarBody.tsx"
import RoomNameBody from "./RoomNameBody.tsx"
import RoomTombstoneBody from "./RoomTombstoneBody.tsx"
import TextMessageBody from "./TextMessageBody.tsx"
import UnknownMessageBody from "./UnknownMessageBody.tsx"
import EventContentProps from "./props.ts"
import "./index.css"

export { default as ACLBody } from "./ACLBody.tsx"
export { default as ContentErrorBoundary } from "./ContentErrorBoundary.tsx"
export { default as EncryptedBody } from "./EncryptedBody.tsx"
export { default as HiddenEvent } from "./HiddenEvent.tsx"
export { default as MediaMessageBody } from "./MediaMessageBody.tsx"
export { default as LocationMessageBody } from "./LocationMessageBody.tsx"
export { default as MemberBody } from "./MemberBody.tsx"
export { default as PinnedEventsBody } from "./PinnedEventsBody.tsx"
export { default as PolicyRuleBody } from "./PolicyRuleBody.tsx"
export { default as PowerLevelBody } from "./PowerLevelBody.tsx"
export { default as RedactedBody } from "./RedactedBody.tsx"
export { default as RoomAvatarBody } from "./RoomAvatarBody.tsx"
export { default as RoomNameBody } from "./RoomNameBody.tsx"
export { default as TextMessageBody } from "./TextMessageBody.tsx"
export { default as RoomTombstoneBody } from "./RoomTombstoneBody.tsx"
export { default as UnknownMessageBody } from "./UnknownMessageBody.tsx"
export type { default as EventContentProps } from "./props.ts"

export function getBodyType(evt: MemDBEvent, forReply = false): React.FunctionComponent<EventContentProps> {
	if (evt.relation_type === "m.replace") {
		return HiddenEvent
	}
	if (evt.state_key === "") {
		// State events which must have an empty state key
		switch (evt.type) {
		case "m.room.name":
			return RoomNameBody
		case "m.room.avatar":
			return RoomAvatarBody
		case "m.room.server_acl":
			return ACLBody
		case "m.room.pinned_events":
			return PinnedEventsBody
		case "m.room.power_levels":
			return PowerLevelBody
		case "m.room.tombstone":
			return RoomTombstoneBody
		}
	} else if (evt.state_key !== undefined) {
		// State events which must have a non-empty state key
		switch (evt.type) {
		case "m.room.member":
			return MemberBody
		case "m.policy.rule.user":
			return PolicyRuleBody
		case "m.policy.rule.room":
			return PolicyRuleBody
		case "m.policy.rule.server":
			return PolicyRuleBody
		}
	} else {
		const isRedacted = evt.redacted_by && !evt.viewing_redacted
		// Non-state events
		switch (evt.type) {
		case "m.room.message":
			if (isRedacted) {
				return RedactedBody
			}
			switch (evt.content?.msgtype) {
			case "m.text":
			case "m.notice":
			case "m.emote":
				return TextMessageBody
			case "m.image":
			case "m.video":
			case "m.audio":
			case "m.file":
				if (forReply) {
					return TextMessageBody
				}
				return MediaMessageBody
			case "m.location":
				if (forReply) {
					return TextMessageBody
				}
				return LocationMessageBody
			default:
				return UnknownMessageBody
			}
		case "m.sticker":
			if (isRedacted) {
				return RedactedBody
			} else if (forReply) {
				return TextMessageBody
			}
			return MediaMessageBody
		case "m.room.encrypted":
			if (isRedacted) {
				return RedactedBody
			}
			return EncryptedBody
		}
	}
	return HiddenEvent
}

export function isSmallEvent(bodyType: React.FunctionComponent<EventContentProps>): boolean {
	switch (bodyType) {
	case HiddenEvent:
	case MemberBody:
	case RoomNameBody:
	case RoomAvatarBody:
	case ACLBody:
	case PolicyRuleBody:
	case PinnedEventsBody:
	case PowerLevelBody:
	case RoomTombstoneBody:
		return true
	default:
		return false
	}
}

export function getPerMessageProfile(evt: MemDBEvent | null): BeeperPerMessageProfile | undefined {
	if (evt === null || evt.type !== "m.room.message" && evt.type !== "m.sticker") {
		return undefined
	}
	const profile = (evt.content as MessageEventContent)["com.beeper.per_message_profile"]
	if (profile?.displayname && typeof profile.displayname !== "string") {
		return undefined
	} else if (profile?.avatar_url && typeof profile.avatar_url !== "string") {
		return undefined
	} else if (profile?.id && typeof profile.id !== "string") {
		return undefined
	}
	return profile
}
