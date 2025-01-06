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
import { use } from "react"
import { PolicyRuleContent } from "@/api/types"
import MainScreenContext from "@/ui/MainScreenContext.ts"
import EventContentProps from "./props.ts"

const BanPolicyBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as PolicyRuleContent
	const prevContent = event.unsigned.prev_content as PolicyRuleContent | undefined
	const mainScreen = use(MainScreenContext)

	let entity = <span>{content.entity || prevContent?.entity}</span>
	if(event.type === "m.policy.rule.user" && !content.entity?.includes("*") && !content.entity?.includes("?")) {
		// Is user policy, and does not include the glob chars * and ?
		entity = (
			<a
				className="hicli-matrix-uri hicli-matrix-uri-user"
				href={`matrix:u/${content.entity.slice(1)}`}
				onClick={mainScreen.clickRightPanelOpener}
				data-target-panel="user"
				data-target-user={content.entity}
			>
				{content.entity}
			</a>
		)
	}

	let action = "added"
	if (prevContent) {
		if (!content) {
			// If the content is empty, the ban is revoked.
			action = "removed"
		} else {
			// There is still content, so the policy was updated
			action = "updated"
		}
	}
	return <div className="policy-body">
		{sender?.content.displayname ?? event.sender} {action} a policy rule {action === "removed" ? "un" : null}&nbsp;
		banning {entity} for: {content.reason}
	</div>
}

export default BanPolicyBody
