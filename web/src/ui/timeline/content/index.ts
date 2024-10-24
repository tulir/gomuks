import React from "react"
import { MemDBEvent } from "@/api/types"
import EncryptedBody from "./EncryptedBody.tsx"
import HiddenEvent from "./HiddenEvent.tsx"
import MediaMessageBody from "./MediaMessageBody.tsx"
import MemberBody from "./MemberBody.tsx"
import RedactedBody from "./RedactedBody.tsx"
import TextMessageBody from "./TextMessageBody.tsx"
import UnknownMessageBody from "./UnknownMessageBody.tsx"
import EventContentProps from "./props.ts"
import "./index.css"

export { default as ContentErrorBoundary } from "./ContentErrorBoundary.tsx"
export { default as EncryptedBody } from "./EncryptedBody.tsx"
export { default as HiddenEvent } from "./HiddenEvent.tsx"
export { default as MediaMessageBody } from "./MediaMessageBody.tsx"
export { default as MemberBody } from "./MemberBody.tsx"
export { default as RedactedBody } from "./RedactedBody.tsx"
export { default as TextMessageBody } from "./TextMessageBody.tsx"
export { default as UnknownMessageBody } from "./UnknownMessageBody.tsx"
export type { default as EventContentProps } from "./props.ts"

export default function getBodyType(evt: MemDBEvent, forReply = false): React.FunctionComponent<EventContentProps> {
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
			// return LocationMessageBody
			// fallthrough
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
	}
	return HiddenEvent
}
