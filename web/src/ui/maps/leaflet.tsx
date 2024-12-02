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

export interface LeafletViewerProps extends HTMLAttributes<HTMLDivElement> {
	tileTemplate: string
	lat: number
	long: number
	prec: number
	marker: string
}

export const LeafletViewer = ({ tileTemplate, lat, long, prec, marker, ...rest }: LeafletViewerProps) => {
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

export interface LocationValue {
	lat: number
	long: number
	prec?: number
}

export interface LeafletPickerProps extends Omit<HTMLAttributes<HTMLDivElement>, "onChange"> {
	tileTemplate: string
	onChange: (geoURI: LocationValue) => void
	initialLocation?: LocationValue
}

export const LeafletPicker = ({ tileTemplate, onChange, initialLocation, ...rest }: LeafletPickerProps) => {
	const ref = useRef<HTMLDivElement>(null)
	const leafletRef = useRef<L.Map>(null)
	const markerRef = useRef<L.Marker>(null)
	useEffect(() => {
		const container = ref.current
		if (!container) {
			return
		}
		const rendered = L.map(container)
		if (initialLocation) {
			rendered.setView([initialLocation.lat, initialLocation.long], initialLocation.prec ?? 13)
			markerRef.current = L.marker([initialLocation.lat, initialLocation.long]).addTo(rendered)
		}
		leafletRef.current = rendered
		L.tileLayer(tileTemplate, { attribution }).addTo(rendered)
		return () => {
			rendered.remove()
		}
		// initialLocation is intentionally immutable/only read once
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [tileTemplate])
	useEffect(() => {
		const map = leafletRef.current
		if (!map) {
			return
		}
		const handler = (evt: L.LeafletMouseEvent) => {
			markerRef.current?.removeFrom(map)
			markerRef.current = L.marker(evt.latlng).addTo(map)
			onChange({ lat: evt.latlng.lat, long: evt.latlng.lng })
		}
		map.on("click", handler)
		return () => {
			map.off("click", handler)
		}
	}, [onChange])
	return <div {...rest} ref={ref}/>
}
