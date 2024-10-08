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
import React, { useCallback, useEffect, useState } from "react"
import type Client from "../../api/client.ts"
import "./LoginScreen.css"
import { ClientState } from "../../api/types/hievents.ts"

export interface LoginScreenProps {
	client: Client
	clientState: ClientState
}

export const LoginScreen = ({ client }: LoginScreenProps) => {
	const [username, setUsername] = useState("")
	const [password, setPassword] = useState("")
	const [homeserverURL, setHomeserverURL] = useState("")
	const [error, setError] = useState("")

	const login = useCallback((evt: React.FormEvent) => {
		evt.preventDefault()
		client.rpc.login(homeserverURL, username, password).then(
			() => {},
			err => setError(err.toString()),
		)
	}, [homeserverURL, username, password, client])

	const resolveHomeserver = useCallback(() => {
		client.rpc.discoverHomeserver(username).then(
			resp => setHomeserverURL(resp["m.homeserver"].base_url),
			err => setError(`Failed to resolve homeserver: ${err}`),
		)
	}, [client, username])

	useEffect(() => {
		if (!username.startsWith("@") || !username.includes(":") || !username.includes(".")) {
			return
		}
		const timeout = setTimeout(resolveHomeserver, 500)
		return () => {
			clearTimeout(timeout)
		}
	}, [username, resolveHomeserver])

	return <main className="matrix-login">
		<h1>gomuks web</h1>
		<form onSubmit={login}>
			<input
				type="text"
				id="mxlogin-username"
				placeholder="User ID"
				value={username}
				onChange={evt => setUsername(evt.target.value)}
			/>
			<input
				type="password"
				id="mxlogin-password"
				placeholder="Password"
				value={password}
				onChange={evt => setPassword(evt.target.value)}
			/>
			<input
				type="text"
				id="mxlogin-homeserver-url"
				placeholder="Homeserver URL"
				value={homeserverURL}
				onChange={evt => setHomeserverURL(evt.target.value)}
			/>
			<button className="mx-login-button" type="submit">Login</button>
		</form>
		{error && <div className="error">
			{error}
		</div>}
	</main>
}
