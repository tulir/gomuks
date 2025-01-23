import { useEffect, useState } from "react"
import Client from "@/api/client.ts"
import { PronounSet, UserProfile } from "@/api/types"
import { ensureArray, ensureString } from "@/util/validation.ts"

interface ExtendedProfileProps {
	profile: UserProfile
	refreshProfile: () => void
	client: Client
	userID: string
}

interface SetTimezoneProps {
	tz?: string
	client: Client
	refreshProfile: () => void
}

const getCurrentTimezone = () => new Intl.DateTimeFormat().resolvedOptions().timeZone

const currentTimeAdjusted = (tz: string) => {
	try {
		return new Intl.DateTimeFormat("en-GB", {
			hour: "numeric",
			minute: "numeric",
			second: "numeric",
			timeZoneName: "short",
			timeZone: tz,
		}).format(new Date())
	} catch {
		return null
	}
}

const ClockElement = ({ tz }: { tz: string }) => {
	const cta = currentTimeAdjusted(tz)
	const isValidTZ = cta !== null
	const [time, setTime] = useState(cta)
	useEffect(() => {
		if (!isValidTZ) {
			return
		}
		let interval: number | undefined
		const updateTime = () => setTime(currentTimeAdjusted(tz))
		const timeout = setTimeout(() => {
			interval = setInterval(updateTime, 1000)
			updateTime()
		}, (1001 - Date.now() % 1000))
		return () => interval ? clearInterval(interval) : clearTimeout(timeout)
	}, [tz, isValidTZ])

	if (!isValidTZ) {
		return null
	}
	return <>
		<div title={tz}>Time:</div>
		<div title={tz}>{time}</div>
	</>
}

const SetTimeZoneElement = ({ tz, client, refreshProfile }: SetTimezoneProps) =>  {
	const zones = Intl.supportedValuesOf("timeZone")
	const saveTz = (newTz: string) => {
		if (!zones.includes(newTz)) {
			return
		}
		client.rpc.setProfileField("us.cloke.msc4175.tz", newTz).then(
			() => refreshProfile(),
			err => {
				console.error("Failed to set time zone:", err)
				window.alert(`Failed to set time zone: ${err}`)
			},
		)
	}

	const defaultValue = tz || getCurrentTimezone()
	return <>
		<label htmlFor="userprofile-timezone-input">Set time zone:</label>
		<input
			list="timezones"
			id="userprofile-timezone-input"
			defaultValue={defaultValue}
			onKeyDown={evt => evt.key === "Enter" && saveTz(evt.currentTarget.value)}
			onBlur={evt => evt.currentTarget.value !== defaultValue && saveTz(evt.currentTarget.value)}
		/>
		<datalist id="timezones">
			{zones.map((zone) => <option key={zone} value={zone} />)}
		</datalist>
	</>
}


const UserExtendedProfile = ({ profile, refreshProfile, client, userID }: ExtendedProfileProps)=>  {
	if (!profile) {
		return null
	}

	const extendedProfileKeys = ["us.cloke.msc4175.tz", "io.fsky.nyx.pronouns"]
	const hasExtendedProfile = extendedProfileKeys.some((key) => profile[key])
	if (!hasExtendedProfile && client.userID !== userID) {
		return null
	}
	// Explicitly only return something if the profile has the keys we're looking for.
	// otherwise there's an ugly and pointless <hr/> for no real reason.

	const pronouns = ensureArray(profile["io.fsky.nyx.pronouns"]) as PronounSet[]
	const userTimeZone = ensureString(profile["us.cloke.msc4175.tz"])
	return <>
		<hr/>
		<div className="extended-profile">
			{userTimeZone && <ClockElement tz={userTimeZone} />}
			{userID === client.userID &&
				<SetTimeZoneElement tz={userTimeZone} client={client} refreshProfile={refreshProfile} />}
			{pronouns.length > 0 && <>
				<div>Pronouns:</div>
				<div>{pronouns.map(pronounSet => ensureString(pronounSet.summary)).join(", ")}</div>
			</>}
		</div>
	</>
}

export default UserExtendedProfile
