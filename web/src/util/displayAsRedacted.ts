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
import type { RoomStateStore } from "@/api/statestore"
import { MemDBEvent, MemberEventContent, PowerLevelEventContent } from "@/api/types"

export function displayAsRedacted(
	evt: MemDBEvent,
	profile?: MemberEventContent,
	memberEvt?: MemDBEvent | null,
	room?: RoomStateStore,
): boolean {
	if (evt.viewing_redacted) {
		return false
	} else if (evt.redacted_by) {
		return true
	} else if (profile?.["org.matrix.msc4293.redact_events"] && profile.membership === "ban") {
		if (memberEvt && room) {
			// It would be more proper to pass the power levels as a parameter so it can use useRoomState,
			// but subscribing to updates isn't that important here.
			const pl = room?.getStateEvent("m.room.power_levels", "")?.content as PowerLevelEventContent | undefined
			const redactPL = pl?.redact ?? 50
			const senderPL = pl?.users?.[memberEvt.sender] ?? pl?.users_default ?? 0
			if (redactPL <= senderPL) {
				return false
			}
		} else {
			return true
		}
	}
	return false
}
