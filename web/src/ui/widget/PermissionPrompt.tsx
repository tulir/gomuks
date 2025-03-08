// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import { MatrixCapabilities } from "matrix-widget-api"
import { use, useState } from "react"
import { ModalCloseContext } from "../modal"

interface PermissionPromptProps {
	capabilities: Set<string>
	onConfirm: (approvedCapabilities: Set<string>) => void
}

const getCapabilityName = (capability: string): string => {
	const paramIdx = capability.indexOf(":")
	const capabilityID = paramIdx === -1 ? capability : capability.slice(0, paramIdx)
	const parameter = paramIdx === -1 ? null : capability.slice(paramIdx + 1)

	// Map capability IDs to human-readable names
	const capabilityNames: Record<string, string> = {
		[MatrixCapabilities.MSC2931Navigate]: "Navigate to other rooms",
		[MatrixCapabilities.MSC3846TurnServers]: "Request TURN servers from the homeserver",
		[MatrixCapabilities.MSC4157SendDelayedEvent]: "Send delayed events",
		[MatrixCapabilities.MSC4157UpdateDelayedEvent]: "Update delayed events",
		[MatrixCapabilities.MSC4039UploadFile]: "Upload files",
		[MatrixCapabilities.MSC4039DownloadFile]: "Download files",
		"org.matrix.msc2762.timeline": "Read room history",
		"org.matrix.msc2762.send.event": "Send timeline events",
		"org.matrix.msc2762.receive.event": "Receive timeline events",
		"org.matrix.msc2762.send.state_event": "Send state events",
		"org.matrix.msc2762.receive.state_event": "Receive state events",
		"org.matrix.msc3819.send.to_device": "Send to-device events",
		"org.matrix.msc3819.receive.to_device": "Receive to-device events",
	}

	const name = capabilityNames[capabilityID] || capabilityID

	if (parameter) {
		return `${name} (${parameter})`
	}

	return name
}

const PermissionPrompt = ({ capabilities, onConfirm }: PermissionPromptProps) => {
	const [selectedCapabilities, setSelectedCapabilities] = useState<Set<string>>(() => new Set(capabilities))
	const closeModal = use(ModalCloseContext)

	const handleToggleCapability = (capability: string) => {
		const newCapabilities = new Set(selectedCapabilities)
		if (newCapabilities.has(capability)) {
			newCapabilities.delete(capability)
		} else {
			newCapabilities.add(capability)
		}
		setSelectedCapabilities(newCapabilities)
	}

	const doConfirm = () => {
		onConfirm(selectedCapabilities)
		closeModal()
	}

	const doReject = () => {
		onConfirm(new Set())
		closeModal()
	}

	return <>
		<h2>Widget Permissions</h2>
		<p>This widget is requesting the following permissions:</p>

		<div className="capability-list">
			{Array.from(capabilities).map((capability) => (
				<div key={capability} className="capability-item">
					<label>
						<input
							type="checkbox"
							checked={selectedCapabilities.has(capability)}
							onChange={() => handleToggleCapability(capability)}
						/>
						{getCapabilityName(capability)}
					</label>
				</div>
			))}
		</div>

		<div className="permission-actions">
			<button onClick={doReject}>Reject all</button>
			<button
				onClick={doConfirm}
				className="confirm-button"
			>
				Accept selected
			</button>
		</div>
	</>
}

export default PermissionPrompt
