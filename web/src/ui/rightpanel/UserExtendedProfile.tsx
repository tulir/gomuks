import { useEffect, useState } from "react"
import { UserProfile } from "@/api/types"
import { ensureArray } from "@/util/validation.ts"

interface PronounSet {
	subject: string
	object: string
	possessive_determiner: string
	possessive_pronoun: string
	reflexive: string
	summary: string
}

interface ExtendedProfileAttributes {
	"us.cloke.msc4175.tz"?: string
	"io.fsky.nyx.pronouns"?: PronounSet[]
}

interface ExtendedProfileProps {
	profile: UserProfile & ExtendedProfileAttributes
}


const currentTimeAdjusted = (tz: string) => {
	const lang = navigator.language || "en-US"
	const now = new Date()
	return new Intl.DateTimeFormat(lang, { timeStyle: "long", timeZone: tz }).format(now)
}

function ClockElement({ tz }: { tz: string }) {
	const [time, setTime] = useState(currentTimeAdjusted(tz))
	useEffect(() => {
		const interval = setInterval(() => {
			setTime(currentTimeAdjusted(tz))
		}, (1000 - Date.now() % 1000))
		return () => clearInterval(interval)
	}, [tz])
	return <div>{time}</div>
}


export default function UserExtendedProfile({ profile }: ExtendedProfileProps) {
	if (!profile) return null

	const pronouns: PronounSet[] = ensureArray(profile["io.fsky.nyx.pronouns"]) as PronounSet[]
	return (
		<div className={"extended-profile"}>
			{
				profile["us.cloke.msc4175.tz"] && (
					<div className={"profile-row"}>
						<div title={profile["us.cloke.msc4175.tz"]}>Time:</div>
						<ClockElement tz={profile["us.cloke.msc4175.tz"]} />
					</div>
				)
			}
			{
				pronouns.length >= 1 && (
					<div className={"profile-row"}>
						<div>Pronouns:</div>
						<div>
							{
								pronouns.map(
									(pronounSet: PronounSet) => (
										pronounSet.summary || `${pronounSet.subject}/${pronounSet.object}`
									),
								).join("/")
							}
						</div>
					</div>
				)
			}
		</div>
	)
}
