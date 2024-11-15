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
import type Client from "@/api/client.ts"
import type { ClientState } from "@/api/types"
import useEvent from "@/util/useEvent.ts"
import BeeperLogin from "./BeeperLogin.tsx"
import "./LoginScreen.css"

export interface LoginScreenProps {
	client: Client
	clientState: ClientState
}

const beeperServerRegex = /^https:\/\/matrix\.(beeper(?:-dev|-staging)?\.com)$/

export const LoginScreen = ({ client }: LoginScreenProps) => {
	const [username, setUsername] = useState("")
	const [password, setPassword] = useState("")
	const [homeserverURL, setHomeserverURL] = useState("")
	const [loginFlows, setLoginFlows] = useState<string[] | null>(null)
	const [error, setError] = useState("")

	const loginSSO = useEvent(() => {
		fetch("_gomuks/sso", {
			method: "POST",
			body: JSON.stringify({ homeserver_url: homeserverURL }),
			headers: { "Content-Type": "application/json" },
		}).then(resp => resp.json()).then(
			resp => {
				const redirectURL = new URL(window.location.href)
				if (!redirectURL.pathname.endsWith("/")) {
					redirectURL.pathname += "/"
				}
				redirectURL.pathname += "_gomuks/sso"
				redirectURL.search = `?gomuksSession=${resp.session_id}`
				redirectURL.hash = ""
				const redir = encodeURIComponent(redirectURL.toString())
				window.location.href = `${homeserverURL}/_matrix/client/v3/login/sso/redirect?redirectUrl=${redir}`
			},
			err => setError(`Failed to start SSO login: ${err}`),
		)
	})

	const login = useEvent((evt: React.FormEvent) => {
		evt.preventDefault()
		if (!loginFlows?.includes("m.login.password")) {
			loginSSO()
			return
		}
		client.rpc.login(homeserverURL, username, password).then(
			() => {},
			err => setError(err.toString()),
		)
	})

	const resolveLoginFlows = useCallback((serverURL: string) => {
		client.rpc.getLoginFlows(serverURL).then(
			resp => setLoginFlows(resp.flows.map(flow => flow.type)),
			err => setError(`Failed to get login flows: ${err}`),
		)
	}, [client])
	const resolveHomeserver = useCallback(() => {
		client.rpc.discoverHomeserver(username).then(
			resp => {
				const url = resp["m.homeserver"].base_url
				setLoginFlows(null)
				setHomeserverURL(url)
				resolveLoginFlows(url)
			},
			err => setError(`Failed to resolve homeserver: ${err}`),
		)
	}, [client, username, resolveLoginFlows])

	useEffect(() => {
		if (!username.startsWith("@") || !username.includes(":") || !username.includes(".")) {
			return
		}
		const timeout = setTimeout(resolveHomeserver, 500)
		return () => {
			clearTimeout(timeout)
		}
	}, [username, resolveHomeserver])
	const onChangeUsername = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setUsername(evt.target.value)
	}, [])
	const onChangePassword = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setPassword(evt.target.value)
	}, [])
	const onChangeHomeserverURL = useCallback((evt: React.ChangeEvent<HTMLInputElement>) => {
		setLoginFlows(null)
		setHomeserverURL(evt.target.value)
	}, [])

	const supportsSSO = loginFlows?.includes("m.login.sso") ?? false
	const supportsPassword = loginFlows?.includes("m.login.password")
	const beeperDomain = homeserverURL.match(beeperServerRegex)?.[1]
	return <main className="matrix-login">
		<h1>gomuks web</h1>
		<form onSubmit={login}>
			<input
				type="text"
				id="mxlogin-username"
				placeholder="User ID"
				value={username}
				onChange={onChangeUsername}
			/>
			{supportsPassword !== false && <input
				type="password"
				id="mxlogin-password"
				placeholder="Password"
				value={password}
				onChange={onChangePassword}
			/>}
			<input
				type="text"
				id="mxlogin-homeserver-url"
				placeholder="Homeserver URL"
				value={homeserverURL}
				onChange={onChangeHomeserverURL}
			/>
			<div className="buttons">
				{supportsSSO && <button
					className="mx-login-button"
					type={supportsPassword ? "button" : "submit"}
					onClick={supportsPassword ? loginSSO : undefined}
				>Login with SSO</button>}
				{supportsPassword !== false && <button
					className="mx-login-button"
					type="submit"
				>Login{supportsSSO || beeperDomain ? " with password" : ""}</button>}
			</div>
			{error && <div className="error">
				{error}
			</div>}
		</form>

		{beeperDomain && <>
			<hr/>
			<BeeperLogin domain={beeperDomain} client={client}/>
		</>}
	</main>
}
