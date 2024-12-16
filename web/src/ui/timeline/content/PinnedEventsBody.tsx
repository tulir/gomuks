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
import { IntlShape, useIntl } from "react-intl"
import { PinnedEventsContent } from "@/api/types"
import { listDiff } from "@/util/diff.ts"
import EventContentProps from "./props.ts"

function renderPinChanges(intl: IntlShape, content: PinnedEventsContent, prevContent?: PinnedEventsContent): string {
	const list = (items: ReadonlyArray<string>) => intl.formatList(items, { type: "conjunction" })
	const [added, removed] = listDiff(content.pinned ?? [], prevContent?.pinned ?? [])
	if (added.length || removed.length) {
		const items = []
		if (added.length) {
			items.push(`pinned ${list(added)}`)
		}
		if (removed.length) {
			items.push(`unpinned ${list(removed)}`)
		}
		return list(items)
	} else {
		return "sent a no-op pin event"
	}
}

const PinnedEventsBody = ({ event, sender }: EventContentProps) => {

	const intl = useIntl()
	const content = event.content as PinnedEventsContent
	const prevContent = event.unsigned.prev_content as PinnedEventsContent | undefined
	return <div className="pinned-events-body">
		{sender?.content.displayname ?? event.sender} {renderPinChanges(intl, content, prevContent)}
	</div>
}

export default PinnedEventsBody
