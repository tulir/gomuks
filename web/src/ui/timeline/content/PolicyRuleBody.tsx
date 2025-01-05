import { PolicyRuleContent } from "@/api/types"
import EventContentProps from "./props.ts"

const BanPolicyBody = ({ event, sender }: EventContentProps) => {
	const content = event.content as PolicyRuleContent
	const prevContent = event.unsigned.prev_content as PolicyRuleContent | undefined

	if (prevContent !== undefined) {
		// all fields for content are missing (this is against spec?) so we need to use the prev event's content
		if (content.entity === undefined) {
			// unban
			return <div className="policy-body">
				{sender?.content.displayname ?? event.sender} Removed the policy rule banning {prevContent.entity}
			</div>
		}
		// update
		return <div className="policy-body">
			{sender?.content.displayname ?? event.sender} Updated a policy rule banning {content.entity} for
			{content.reason}
		</div>
	}
	// add
	return <div className="policy-body">
		{sender?.content.displayname ?? event.sender} Added a policy rule banning {content.entity} for {content.reason}
	</div>
}

export default BanPolicyBody
