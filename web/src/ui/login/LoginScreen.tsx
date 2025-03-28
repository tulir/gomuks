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
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState("")

	const loginSSO = () => {
		setLoading(true)
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
		).finally(() => setLoading(false))
	}

	const login = (evt: React.FormEvent) => {
		evt.preventDefault()
		if (!loginFlows) {
			// do nothing
		} else if (!loginFlows.includes("m.login.password")) {
			loginSSO()
		} else {
			setLoading(true)
			client.rpc.login(homeserverURL, username, password).then(
				() => {},
				err => setError(err.toString()),
			).finally(() => setLoading(false))
		}
	}

	const resolveLoginFlows = useCallback((serverURL: string) => {
		client.rpc.getLoginFlows(serverURL).then(
			resp => {
				setLoginFlows(resp.flows.map(flow => flow.type))
				setError("")
			},
			err => setError(`Failed to get login flows: ${err}`),
		)
	}, [client])
	const resolveHomeserver = useCallback(() => {
		client.rpc.discoverHomeserver(username).then(
			resp => {
				const url = resp["m.homeserver"].base_url
				setLoginFlows([])
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
	useEffect(() => {
		if (loginFlows !== null || loginFlows === "resolving" || !homeserverURL) {
			return
		}
		const timeout = setTimeout(() => resolveLoginFlows(homeserverURL), 500)
		return () => {
			clearTimeout(timeout)
		}
	}, [homeserverURL, loginFlows, resolveLoginFlows])
	const onChangeHomeserverURL = (evt: React.ChangeEvent<HTMLInputElement>) => {
		setLoginFlows(null)
		setHomeserverURL(evt.target.value)
	}

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
				onChange={evt => setUsername(evt.target.value)}
			/>
			<input
				type="text"
				id="mxlogin-homeserver-url"
				placeholder="Homeserver URL (will autofill)"
				value={homeserverURL}
				onChange={onChangeHomeserverURL}
			/>
			{supportsPassword && <input
				type="password"
				id="mxlogin-password"
				placeholder="Password"
				value={password}
				onChange={evt => setPassword(evt.target.value)}
			/>}
			<div className="buttons">
				{supportsSSO && <button
					className="mx-login-button primary-color-button"
					type={supportsPassword ? "button" : "submit"}
					disabled={loading}
					onClick={supportsPassword ? loginSSO : undefined}
				>Login with SSO</button>}
				{supportsPassword && <button
					className="mx-login-button primary-color-button"
					type="submit"
					disabled={loading}
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
