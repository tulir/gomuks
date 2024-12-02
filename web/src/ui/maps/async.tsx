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
import { Suspense, lazy } from "react"
import { GridLoader } from "react-spinners"
import type { LeafletPickerProps, LeafletViewerProps } from "./leaflet.tsx"

const locationLoader = <div className="location-importer"><GridLoader color="var(--primary-color)" size={25}/></div>

const LazyLeafletViewer = lazy(
	() => import("./leaflet.tsx").then(res => ({ default: res.LeafletViewer })))

const LazyLeafletPicker = lazy(
	() => import("./leaflet.tsx").then(res => ({ default: res.LeafletPicker })))

export const LeafletViewer = (props: LeafletViewerProps) => {
	return <Suspense fallback={locationLoader}>
		<LazyLeafletViewer {...props}/>
	</Suspense>
}

export const LeafletPicker = (props: LeafletPickerProps) => {
	return <Suspense fallback={locationLoader}>
		<LazyLeafletPicker {...props}/>
	</Suspense>
}
