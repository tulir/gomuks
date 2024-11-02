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
import katex from "katex"
import katexCSS from "katex/dist/katex.min.css?inline"

const sheet = new CSSStyleSheet()
sheet.replaceSync(katexCSS)

class HicliMath extends HTMLElement {
	static observedAttributes = ["displaymode", "latex"]
	#root?: HTMLElement
	#latex?: string
	#displayMode?: boolean

	constructor() {
		super()
	}

	connectedCallback() {
		const root = this.attachShadow({ mode: "open" })
		root.adoptedStyleSheets = [sheet]
		// This seems to work fine
		this.#root = root as unknown as HTMLElement
		this.#render()
	}

	attributeChangedCallback(name: string, _oldValue: string, newValue: string) {
		if (name === "latex") {
			this.#latex = newValue
		} else if (name === "displaymode") {
			this.#displayMode = newValue === "block"
		}
		this.#render()
	}

	#render() {
		if (!this.#root || !this.#latex) {
			return
		}
		try {
			katex.render(this.#latex, this.#root, {
				output: "htmlAndMathml",
				maxSize: 10,
				displayMode: this.#displayMode,
			})
		} catch (err) {
			console.error("Failed to render math", this.#latex, err)
			const errorNode = document.createElement("span")
			errorNode.innerText = `Failed to render math: ${err}`
			errorNode.style.color = "var(--error-color)"
			this.#root.appendChild(errorNode)
		}
	}
}

customElements.define("hicli-math", HicliMath)
