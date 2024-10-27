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
import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import App from "./App.tsx"
import "./index.css"

const styleTags = document.createElement("style")
// styleTags.textContent = `
// @import "_gomuks/codeblock/github-dark.css" (prefers-color-scheme: dark);
// @import "_gomuks/codeblock/github.css" (prefers-color-scheme: light);
// `
// TODO switch to the above version after adding global dark theme support
styleTags.textContent = `@import "_gomuks/codeblock/github.css";`
document.head.appendChild(styleTags)

fetch("_gomuks/auth", { method: "POST" }).then(resp => {
	if (resp.ok) {
		createRoot(document.getElementById("root")!).render(
			<StrictMode>
				<App/>
			</StrictMode>,
		)
	} else {
		window.alert("Authentication failed: " + resp.statusText)
	}
}, err => {
	window.alert("Authentication failed: " + err)
})
