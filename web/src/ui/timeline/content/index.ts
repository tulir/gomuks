import React from "react"
import { MemDBEvent } from "@/api/types"
import ACLBody from "./ACLBody.tsx"
import EncryptedBody from "./EncryptedBody.tsx"
import HiddenEvent from "./HiddenEvent.tsx"
import LocationMessageBody from "./LocationMessageBody.tsx"
import MediaMessageBody from "./MediaMessageBody.tsx"
import MemberBody from "./MemberBody.tsx"
import PinnedEventsBody from "./PinnedEventsBody.tsx"
import PowerLevelBody from "./PowerLevelBody.tsx"
import RedactedBody from "./RedactedBody.tsx"
import RoomAvatarBody from "./RoomAvatarBody.tsx"
import RoomNameBody from "./RoomNameBody.tsx"
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
export { default as PowerLevelBody } from "./PowerLevelBody.tsx"
export { default as RedactedBody } from "./RedactedBody.tsx"
export { default as RoomAvatarBody } from "./RoomAvatarBody.tsx"
export { default as RoomNameBody } from "./RoomNameBody.tsx"
export { default as TextMessageBody } from "./TextMessageBody.tsx"
export { default as UnknownMessageBody } from "./UnknownMessageBody.tsx"
export type { default as EventContentProps } from "./props.ts"

export function getBodyType(evt: MemDBEvent, forReply = false): React.FunctionComponent<EventContentProps> {
	if (evt.relation_type === "m.replace") {
		return HiddenEvent
	}
	switch (evt.type) {
	case "m.room.message":
		if (evt.redacted_by) {
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
			return LocationMessageBody
		default:
			return UnknownMessageBody
		}
	case "m.sticker":
		if (evt.redacted_by) {
			return RedactedBody
		} else if (forReply) {
			return TextMessageBody
		}
		return MediaMessageBody
	case "m.room.encrypted":
		if (evt.redacted_by) {
			return RedactedBody
		}
		return EncryptedBody
	case "m.room.member":
		return MemberBody
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
	case PinnedEventsBody:
	case PowerLevelBody:
		return true
	default:
		return false
	}
}
