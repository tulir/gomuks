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
import React, { useState } from "react"
import { LoginScreenProps } from "./LoginScreen.tsx"
import "./LoginScreen.css"

export const VerificationScreen = ({ client, clientState }: LoginScreenProps) => {
	if (!clientState.is_logged_in) {
		throw new Error("Invalid state")
	}
	const [recoveryKey, setRecoveryKey] = useState("")
	const [error, setError] = useState("")

	const verify = (evt: React.FormEvent) => {
		evt.preventDefault()
		client.rpc.verify(recoveryKey).then(
			() => {},
			err => setError(err.toString()),
		)
	}

	return <main className="matrix-login">
		<h1>gomuks web</h1>
		<form onSubmit={verify}>
			<p>Successfully logged in as <code>{clientState.user_id}</code></p>
			<input
				type="text"
				autoComplete="off"
				id="mxlogin-recoverykey"
				placeholder="Recovery key or passphrase"
				value={recoveryKey}
				onChange={evt => setRecoveryKey(evt.target.value)}
			/>
			<button className="mx-login-button primary-color-button" type="submit">Verify</button>
		</form>
		{error && <div className="error">
			{error}
		</div>}
	</main>
}
