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
import { PinnedEventsContent } from "@/api/types"
import { listDiff } from "@/util/diff.ts"
import { humanJoin } from "@/util/join.ts"
import { ensureTypedArray, getDisplayname, isEventID } from "@/util/validation.ts"
import EventContentProps from "./props.ts"

function renderPinChanges(content: PinnedEventsContent, prevContent?: PinnedEventsContent): string {
	const [added, removed] = listDiff(
		ensureTypedArray(content.pinned ?? [], isEventID),
		ensureTypedArray(prevContent?.pinned ?? [], isEventID),
	)
	if (added.length) {
		if (removed.length) {
			return `pinned ${humanJoin(added)} and unpinned ${humanJoin(removed)}`
		}
		return `pinned ${humanJoin(added)}`
	} else if (removed.length) {
		return `unpinned ${humanJoin(removed)}`
	} else {
		return "sent a no-op pin event"
	}
}

const PinnedEventsBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as PinnedEventsContent
	const prevContent = event.unsigned.prev_content as PinnedEventsContent | undefined
	return <div className="pinned-events-body">
		{getDisplayname(event.sender, sender?.content)} {renderPinChanges(content, prevContent)}
	</div>
}

export default PinnedEventsBody
