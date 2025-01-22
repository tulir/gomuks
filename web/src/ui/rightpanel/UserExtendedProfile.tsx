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

interface PronounsElementProps {
	userID: string
	pronouns: PronounSet[]
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
	} catch (e) {
		return `${e}`
	}
}

const ClockElement = ({ tz }: { tz: string }) => {
	const [time, setTime] = useState(currentTimeAdjusted(tz))
	useEffect(() => {
		let interval: number | undefined
		const updateTime = () => setTime(currentTimeAdjusted(tz))
		const timeout = setTimeout(() => {
			interval = setInterval(updateTime, 1000)
			updateTime()
		}, (1001 - Date.now() % 1000))
		return () => interval ? clearInterval(interval) : clearTimeout(timeout)
	}, [tz])

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

const PronounsElement = ({ userID, pronouns, client, refreshProfile }: PronounsElementProps) => {
	const display = pronouns.map(pronounSet => ensureString(pronounSet.summary)).join(", ")
	if (userID !== client.userID) {
		return <>
			<div>Pronouns:</div>
			<div>{display}</div>
		</>
	}
	const savePronouns = (newPronouns: string) => {
		// convert to pronouns object
		const newPronounsArray = newPronouns.split(",").map(pronoun => ({ summary: pronoun.trim(), language: "en" }))
		console.debug("Rendered new pronouns:", newPronounsArray)
		client.rpc.setProfileField("io.fsky.nyx.pronouns", newPronounsArray).then(
			() => {console.debug("Set new pronouns."); refreshProfile()},
			err => {
				console.error("Failed to set pronouns:", err)
				window.alert(`Failed to set pronouns: ${err}`)
			},
		)
	}
	return <>
		<label htmlFor="userprofile-pronouns-input">Pronouns:</label>
		<input
			id="userprofile-pronouns-input"
			defaultValue={display}
			onKeyDown={evt => evt.key === "Enter" && savePronouns(evt.currentTarget.value)}
			onBlur={evt => evt.currentTarget.value !== display && savePronouns(evt.currentTarget.value)}
		/>
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
	const displayPronouns = pronouns.length > 0 || client.userID === userID
	return <>
		<hr/>
		<div className="extended-profile">
			{userTimeZone && <ClockElement tz={userTimeZone} />}
			{userID === client.userID &&
				<SetTimeZoneElement tz={userTimeZone} client={client} refreshProfile={refreshProfile} />}
			{displayPronouns &&
				<PronounsElement userID={userID} pronouns={pronouns} client={client} refreshProfile={refreshProfile} />}
		</div>
	</>
}

export default UserExtendedProfile
