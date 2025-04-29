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
import { JSX, use } from "react"
import { PolicyRuleContent } from "@/api/types"
import { ensureString, getDisplayname } from "@/util/validation.ts"
import MainScreenContext from "../../MainScreenContext.ts"
import EventContentProps from "./props.ts"

const PolicyRuleBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as PolicyRuleContent
	const prevContent = event.unsigned.prev_content as PolicyRuleContent | undefined
	const mainScreen = use(MainScreenContext)

	const entity = ensureString(content.entity ?? prevContent?.entity)
	const hashedEntity = ensureString(content["org.matrix.msc4205.hashes"]?.sha256
		?? prevContent?.["org.matrix.msc4205.hashes"]?.sha256)
	const recommendation = ensureString(content.recommendation ?? prevContent?.recommendation)
	if ((!entity && !hashedEntity) || !recommendation) {
		return <div className="policy-body">
			{getDisplayname(event.sender, sender?.content)} sent an invalid policy rule
		</div>
	}
	const target = event.type.replace(/^m\.policy\.rule\./, "")
	let entityElement = <>{entity}</>
	let matchingWord = `${target}s matching`
	if (!entity && hashedEntity) {
		matchingWord = `the ${target} with hash`
		entityElement = <>{hashedEntity}</>
	} else if (event.type === "m.policy.rule.user" && entity && !entity.includes("*") && !entity.includes("?")) {
		entityElement = (
			<a
				className="hicli-matrix-uri hicli-matrix-uri-user"
				href={`matrix:u/${entity.slice(1)}`}
				onClick={mainScreen.clickRightPanelOpener}
				data-target-panel="user"
				data-target-user={entity}
			>
				{entity}
			</a>
		)
		matchingWord = "user"
	}
	let recommendationElement: JSX.Element | string = <code>{recommendation}</code>
	if (recommendation === "m.ban") {
		recommendationElement = "ban"
	} else if (recommendation === "org.matrix.msc4204.takedown") {
		recommendationElement = "takedown"
	}
	const action = prevContent ? ((content.entity && content.recommendation) ? "updated" : "removed") : "added"
	return <div className="policy-body">
		{getDisplayname(event.sender, sender?.content)} {action} a {recommendationElement} rule
		for {matchingWord} <code>{entityElement}</code>
		{content.reason ? <> for <code>{ensureString(content.reason)}</code></> : null}
	</div>
}

export default PolicyRuleBody
