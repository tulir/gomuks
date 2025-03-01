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
import EventContentProps from "./props.ts"
import LockIcon from "../../../icons/lock.svg?react"
import LockClockIcon from "../../../icons/lock.svg?react"

const unknownSessionErrorPrefix = "failed to decrypt megolm event: no session with given ID found"

const EncryptedBody = ({ event }: EventContentProps) => {
	const decryptionError = event.last_edit?.decryption_error ?? event.decryption_error
	if (decryptionError && !decryptionError.startsWith(unknownSessionErrorPrefix)) {
		return <div className="decryption-error-body"><LockIcon/> Failed to decrypt: {decryptionError}</div>
	}
	return <div className="decryption-pending-body"><LockClockIcon/> Waiting for message</div>
}

export default EncryptedBody
