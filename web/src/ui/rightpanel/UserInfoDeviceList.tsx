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
import { useEffect, useState, useTransition } from "react"
import { ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import { ProfileDevice, ProfileEncryptionInfo, TrustState, UserID } from "@/api/types"
import UserInfoError from "./UserInfoError.tsx"
import DevicesIcon from "@/icons/devices.svg?react"
import EncryptedOffIcon from "@/icons/encrypted-off.svg?react"
import EncryptedQuestionIcon from "@/icons/encrypted-question.svg?react"
import EncryptedIcon from "@/icons/encrypted.svg?react"

interface DeviceListProps {
	client: Client
	room?: RoomStateStore
	userID: UserID
}

const DeviceList = ({ client, room, userID }: DeviceListProps) => {
	const [view, setEncryptionInfo] = useState<ProfileEncryptionInfo | null>(null)
	const [errors, setErrors] = useState<string[] | null>(null)
	const [trackChangePending, startTransition] = useTransition()
	const doTrackDeviceList = () => {
		startTransition(async () => {
			try {
				const resp = await client.rpc.trackUserDevices(userID)
				startTransition(() => {
					setEncryptionInfo(resp)
					setErrors(resp.errors)
				})
			} catch (err) {
				startTransition(() => setErrors([`${err}`]))
			}
		})
	}
	useEffect(() => {
		setEncryptionInfo(null)
		setErrors(null)
		client.rpc.getProfileEncryptionInfo(userID).then(
			resp => {
				setEncryptionInfo(resp)
				setErrors(resp.errors)
			},
			err => setErrors([`${err}`]),
		)
	}, [client, userID])
	const isEncrypted = room?.meta.current.encryption_event?.algorithm === "m.megolm.v1.aes-sha2"
	const encryptionMessage = isEncrypted
		? "Messages in this room are end-to-end encrypted."
		: "Messages in this room are not end-to-end encrypted."
	if (view === null) {
		return <div className="devices not-tracked">
			<h4>Security</h4>
			<p>{encryptionMessage}</p>
			{!errors ? <ScaleLoader className="user-info-loader" color="var(--primary-color)"/> : null}
			<UserInfoError errors={errors}/>
		</div>
	}
	if (!view.devices_tracked) {
		return <div className="devices not-tracked">
			<h4>Security</h4>
			<p>{encryptionMessage}</p>
			<p>This user's device list is not being tracked.</p>
			<button className="action" onClick={doTrackDeviceList} disabled={trackChangePending}>
				<DevicesIcon /> Start tracking device list
			</button>
			<UserInfoError errors={errors}/>
		</div>
	}
	let verifiedMessage = null
	if (view.user_trusted) {
		verifiedMessage = <p className="verified-message verified" title={view.master_key}>
			<EncryptedIcon/> You have verified this user
		</p>
	} else if (view.master_key) {
		if (view.master_key === view.first_master_key) {
			verifiedMessage = <p className="verified-message tofu" title={view.master_key}>
				<EncryptedIcon/> Trusted master key on first use
			</p>
		} else {
			verifiedMessage = <p className="verified-message tofu-broken" title={view.master_key}>
				<EncryptedQuestionIcon/> Master key has changed
			</p>
		}
	}
	return <div className="devices">
		<h4>Security</h4>
		<p>{encryptionMessage}</p>
		{verifiedMessage}
		<details>
			<summary><h4>{view.devices.length} devices</h4></summary>
			<ul>{view.devices.map(dev => renderDevice(dev, view.master_key !== ""))}</ul>
		</details>
		<UserInfoError errors={errors}/>
	</div>
}

function renderDevice(device: ProfileDevice, hasCSKeys: boolean) {
	let Icon = EncryptedIcon
	if (device.trust_state === "blacklisted") {
		Icon = EncryptedOffIcon
	} else if (device.trust_state === "cross-signed-untrusted" || device.trust_state === "unverified") {
		Icon = EncryptedQuestionIcon
	}
	return <li key={device.device_id} className="device">
		<div
			className={`icon-wrapper trust-${device.trust_state} ${hasCSKeys ? "has-master-key" : "no-master-key"}`}
			title={trustStateDescription(device.trust_state, hasCSKeys)}
		><Icon/></div>
		<div title={device.device_id}>{device.name || device.device_id}</div>
	</li>
}

function trustStateDescription(state: TrustState, hasCSKeys: boolean): string {
	switch (state) {
	case "blacklisted":
		return "Device has been blacklisted manually"
	case "unverified":
		if (hasCSKeys) {
			return "Device has not been verified by cross-signing keys"
		} else {
			return "No cross-signing keys were found"
		}
	case "verified":
		return "Device was verified manually"
	case "cross-signed-untrusted":
		return "Device is cross-signed, cross-signing keys are NOT trusted"
	case "cross-signed-tofu":
		return "Device is cross-signed, cross-signing keys were trusted on first use"
	case "cross-signed-verified":
		return "Device is cross-signed, cross-signing keys were verified manually"
	default:
		return "Invalid trust state"
	}
}

export default DeviceList
