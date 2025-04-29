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
import { PowerLevelEventContent } from "@/api/types"
import { objectDiff } from "@/util/diff.ts"
import { humanJoin } from "@/util/join.ts"
import { getDisplayname } from "@/util/validation.ts"
import EventContentProps from "./props.ts"

function intDiff(messageParts: TemplateStringsArray, oldVal: number, newVal: number): string | null {
	if (oldVal === newVal) {
		return null
	}
	return `${messageParts[0]}${oldVal}${messageParts[1]}${newVal}${messageParts[2] ?? ""}`
}

function renderPowerLevels(content: PowerLevelEventContent, prevContent?: PowerLevelEventContent): string[] {
	/* eslint-disable max-len */
	const output = [
		intDiff`the default user power level from ${prevContent?.users_default ?? 0} to ${content.users_default ?? 0}`,
		intDiff`the default event power level from ${prevContent?.events_default ?? 0} to ${content.events_default ?? 0}`,
		intDiff`the default state event power level from ${prevContent?.state_default ?? 50} to ${content.state_default ?? 50}`,
		intDiff`the ban power level from ${prevContent?.ban ?? 50} to ${content.ban ?? 50}`,
		intDiff`the kick power level from ${prevContent?.kick ?? 50} to ${content.kick ?? 50}`,
		intDiff`the redact power level from ${prevContent?.redact ?? 50} to ${content.redact ?? 50}`,
		intDiff`the invite power level from ${prevContent?.invite ?? 0} to ${content.invite ?? 0}`,
		intDiff`the @room notification power level from ${prevContent?.notifications?.room ?? 50} to ${content.notifications?.room ?? 50}`,
	]
	/* eslint-enable max-len */
	const userDiffs = objectDiff(
		content.users ?? {},
		prevContent?.users ?? {},
		content.users_default ?? 0,
		prevContent?.users_default ?? 0,
	)
	for (const [userID, { old: oldLevel, new: newLevel }] of userDiffs.entries()) {
		output.push(`changed ${userID}'s power level from ${oldLevel} to ${newLevel}`)
	}
	const eventDiffs = objectDiff(content.events ?? {}, prevContent?.events ?? {})
	for (const [eventType, { old: oldLevel, new: newLevel }] of eventDiffs.entries()) {
		if (oldLevel === undefined) {
			output.push(`set the power level for ${eventType} to ${newLevel}`)
		} else if (newLevel === undefined) {
			output.push(`removed the power level for ${eventType} (was ${oldLevel})`)
		} else {
			output.push(`changed the power level for ${eventType} from ${oldLevel} to ${newLevel}`)
		}
	}
	const filtered = output.filter(x => x !== null)
	if (filtered.length === 0) {
		return ["sent a power level event with no changes"]
	}
	return filtered
}

const PowerLevelBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as PowerLevelEventContent
	const prevContent = event.unsigned.prev_content as PowerLevelEventContent | undefined
	return <div className="power-level-body">
		{getDisplayname(event.sender, sender?.content)} {humanJoin(renderPowerLevels(content, prevContent))}
	</div>
}

export default PowerLevelBody
