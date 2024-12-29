import { useEffect, useState } from "react"
import { UserProfile } from "@/api/types"
import { ensureArray } from "@/util/validation.ts"
import Client from "@/api/client.ts";

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
	client: Client
	userID: string
}

interface SetTimezoneProps {
	tz?: string
	client: Client
}

const getCurrentTimezone = () => Intl.DateTimeFormat().resolvedOptions().timeZone

const currentTimeAdjusted = (tz: string) => {
	const lang = navigator.language || "en-US"
	const now = new Date()
	try {
		return new Intl.DateTimeFormat(lang, { timeStyle: "long", timeZone: tz }).format(now)
	} catch (e) {
		return `Error: ${e}`
	}
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

function SetTimezoneElement({ tz, client }: SetTimezoneProps) {
	const zones = Intl.supportedValuesOf("timeZone")
	const setTz = (newTz: string) => {
		if (zones.includes(newTz) && newTz !== tz) {
			return client.rpc.setProfileField("us.cloke.msc4175.tz", newTz).then(
				() => client.rpc.getProfile(client.userID),
				(err) => console.error("Error setting timezone", err),
			)
		}
	}

	return (
		<>
			<input
				list={"timezones"}
				className={"text-input"}
				defaultValue={tz || getCurrentTimezone()}
				onChange={(e) => setTz(e.currentTarget.value)}
			/>
			<datalist id={"timezones"}>
				{
					zones.map((zone) => <option key={zone} value={zone} />)
				}
			</datalist>
		</>
	)
}


export default function UserExtendedProfile({ profile, client, userID }: ExtendedProfileProps) {
	if (!profile) return null

	const extendedProfileKeys = ["us.cloke.msc4175.tz", "io.fsky.nyx.pronouns"]
	const hasExtendedProfile = extendedProfileKeys.some((key) => key in profile)
	if (!hasExtendedProfile && client.userID !== userID) return null
	// Explicitly only return something if the profile has the keys we're looking for.
	// otherwise there's an ugly and pointless <hr/> for no real reason.

	const pronouns: PronounSet[] = ensureArray(profile["io.fsky.nyx.pronouns"]) as PronounSet[]
	return (
		<>
			<hr/>
			<div className={"extended-profile"}>
				{
					profile["us.cloke.msc4175.tz"] && (
						<>
							<div title={profile["us.cloke.msc4175.tz"]}>Time:</div>
							<ClockElement tz={profile["us.cloke.msc4175.tz"]} />
						</>
					)
				}
				{
					userID === client.userID && (
						<>
							<div>Set Timezone:</div>
							<SetTimezoneElement tz={profile["us.cloke.msc4175.tz"]} client={client} />
						</>
					)
				}
				{
					pronouns.length >= 1 && (
						<>
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
						</>
					)
				}
			</div>
		</>
	)
}
