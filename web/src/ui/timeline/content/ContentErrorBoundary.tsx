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
import React from "react"

export default class ContentErrorBoundary extends React.Component<{ children: React.ReactNode }, { error?: Error }> {
	constructor(props: { children: React.ReactNode }) {
		super(props)
		this.state = { error: undefined }
	}

	static getDerivedStateFromError(error: unknown) {
		if (error instanceof Error) {
			error = new Error(`${error}`)
		}
		return { error }
	}

	render() {
		if (this.state.error) {
			return <div className="render-error-body">
				Failed to render event: {this.state.error.message.replace(/^Error: /, "")}
			</div>
		}

		return this.props.children
	}
}
