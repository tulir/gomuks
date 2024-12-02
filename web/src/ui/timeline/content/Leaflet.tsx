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
import L from "leaflet"
import markerIconRetinaUrl from "leaflet/dist/images/marker-icon-2x.png"
import markerIconUrl from "leaflet/dist/images/marker-icon.png"
import markerShadowUrl from "leaflet/dist/images/marker-shadow.png"
import "leaflet/dist/leaflet.css"
import { HTMLAttributes, useEffect, useRef } from "react"

L.Icon.Default.prototype.options.iconUrl = markerIconUrl
L.Icon.Default.prototype.options.iconRetinaUrl = markerIconRetinaUrl
L.Icon.Default.prototype.options.shadowUrl = markerShadowUrl
L.Icon.Default.imagePath = ""

const attribution = `&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors`

export interface GomuksLeafletProps extends HTMLAttributes<HTMLDivElement> {
	tileTemplate: string
	lat: number
	long: number
	prec: number
	marker: string
}

const GomuksLeaflet = ({ tileTemplate, lat, long, prec, marker, ...rest }: GomuksLeafletProps) => {
	const ref = useRef<HTMLDivElement>(null)
	useEffect(() => {
		const container = ref.current
		if (!container) {
			return
		}
		const rendered = L.map(container)
		rendered.setView([lat, long], prec)
		const markerElem = L.marker([lat, long]).addTo(rendered)
		markerElem.bindPopup(marker).openPopup()
		L.tileLayer(tileTemplate, { attribution }).addTo(rendered)
		return () => {
			rendered.remove()
		}
	}, [lat, long, prec, marker, tileTemplate])
	return <div {...rest} ref={ref}/>
}

export default GomuksLeaflet
