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
import type { IWidget } from "matrix-widget-api"
import { Suspense, lazy, use } from "react"
import { GridLoader } from "react-spinners"
import ClientContext from "../ClientContext"
import { RoomContext } from "../roomview/roomcontext"

const Widget = lazy(() => import("./widget"))

const widgetLoader = <div className="widget-container widget-loading">
	<GridLoader color="var(--primary-color)" size={20} />
</div>


export interface LazyWidgetProps {
	info: IWidget
	onClose?: () => void
}

const LazyWidget = ({ info, onClose }: LazyWidgetProps) => {
	const room = use(RoomContext)?.store
	const client = use(ClientContext)
	if (!room || !client) {
		return null
	}
	return (
		<Suspense fallback={widgetLoader}>
			<Widget info={info} room={room} client={client} onClose={onClose} />
		</Suspense>
	)
}

export default LazyWidget
