import React from "react"
import { MemDBEvent } from "@/api/types"
import EncryptedBody from "./EncryptedBody.tsx"
import HiddenEvent from "./HiddenEvent.tsx"
import MemberBody from "./MemberBody.tsx"
import { MediaMessageBody, TextMessageBody, UnknownMessageBody } from "./MessageBody.tsx"
import RedactedBody from "./RedactedBody.tsx"
import { EventContentProps } from "./props.ts"

export default function getBodyType(evt: MemDBEvent, forReply = false): React.FunctionComponent<EventContentProps> {
	if (evt.relation_type === "m.replace") {
		return HiddenEvent
	}
	switch (evt.type) {
	case "m.room.message":
		if (evt.redacted_by) {
			return RedactedBody
		}
		switch (evt.content.msgtype) {
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
