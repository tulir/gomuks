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
import * as beeper from "@/api/beeper.ts"
import type Client from "@/api/client.ts"

interface BeeperLoginProps {
	domain: string
	client: Client
}

const BeeperLogin = ({ domain, client }: BeeperLoginProps) => {
	const [email, setEmail] = useState("")
	const [requestID, setRequestID] = useState("")
	const [code, setCode] = useState("")
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState("")

	const onChangeEmail = (evt: React.ChangeEvent<HTMLInputElement>) => {
		setEmail(evt.target.value)
	}
	const onChangeCode = (evt: React.ChangeEvent<HTMLInputElement>) => {
		let codeDigits = evt.target.value.replace(/\D/g, "").slice(0, 6)
		if (codeDigits.length > 3) {
			codeDigits = codeDigits.slice(0, 3) + " " + codeDigits.slice(3)
		}
		setCode(codeDigits)
	}

	const requestCode = (evt: React.FormEvent) => {
		evt.preventDefault()
		setLoading(true)
		beeper.doStartLogin(domain).then(
			request => beeper.doRequestCode(domain, request, email).then(
				() => setRequestID(request),
				err => setError(`Failed to request code: ${err}`),
			),
			err => setError(`Failed to start login: ${err}`),
		).finally(() => setLoading(false))
	}
	const submitCode = (evt: React.FormEvent) => {
		evt.preventDefault()
		setLoading(true)
		beeper.doSubmitCode(domain, requestID, code).then(
			token => {
				client.rpc.loginCustom(`https://matrix.${domain}`, {
					type: "org.matrix.login.jwt",
					token,
				}).catch(err => setError(`Failed to login with token: ${err}`))
			},
			err => setError(`Failed to submit code: ${err}`),
		).finally(() => setLoading(false))
	}

	return <form onSubmit={requestID ? submitCode : requestCode} className="beeper-login">
		<h2>Beeper email login</h2>
		<input
			type="email"
			id="beeperlogin-email"
			placeholder="Email"
			value={email}
			onChange={onChangeEmail}
			disabled={!!requestID}
		/>
		{requestID && <input
			type="text"
			pattern="[0-9]{3} [0-9]{3}"
			id="beeperlogin-code"
			placeholder="Confirmation Code"
			value={code}
			onChange={onChangeCode}
		/>}
		<button
			className="beeper-login-button primary-color-button"
			type="submit"
			disabled={loading}
		>{requestID ? "Submit Code" : "Request Code"}</button>
		{error && <div className="error">
			{error}
		</div>}
	</form>
}

export default BeeperLogin
