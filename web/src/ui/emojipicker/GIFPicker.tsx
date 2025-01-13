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
import React, { CSSProperties, use, useEffect, useState } from "react"
import { RoomStateStore, usePreference } from "@/api/statestore"
import { MediaMessageEventContent } from "@/api/types"
import { isMobileDevice } from "@/util/ismobile.ts"
import ClientContext from "../ClientContext.ts"
import { ModalCloseContext } from "../modal"
import { GIF, getTrendingGIFs, searchGIF } from "./gifsource.ts"
import CloseIcon from "@/icons/close.svg?react"
import SearchIcon from "@/icons/search.svg?react"

export interface MediaPickerProps {
	style: CSSProperties
	onSelect: (media: MediaMessageEventContent) => void
	room: RoomStateStore
}

const trendingCache = new Map<string, GIF[]>()

const GIFPicker = ({ style, onSelect, room }: MediaPickerProps) => {
	const [query, setQuery] = useState("")
	const [results, setResults] = useState<GIF[]>([])
	const [error, setError] = useState<unknown>()
	const close = use(ModalCloseContext)
	const client = use(ClientContext)!
	const provider = usePreference(client.store, room, "gif_provider")
	const providerName = provider.slice(0, 1).toUpperCase() + provider.slice(1)
	// const reuploadGIFs = room.preferences.reupload_gifs
	const onSelectGIF = (evt: React.MouseEvent<HTMLDivElement>) => {
		const idx = evt.currentTarget.getAttribute("data-gif-index")
		if (!idx) {
			return
		}
		const gif = results[+idx]
		// if (reuploadGIFs) {
		// 	// TODO
		// }
		onSelect({
			msgtype: "m.image",
			body: gif.filename,
			filename: gif.filename,
			info: {
				mimetype: "image/webp",
				size: gif.size,
				w: gif.width,
				h: gif.height,
			},
			url: gif.proxied_mxc,
		})
		close()
	}
	useEffect(() => {
		if (!query) {
			if (trendingCache.has(provider)) {
				setResults(trendingCache.get(provider)!)
				return
			} else {
				const abort = new AbortController()
				getTrendingGIFs(provider).then(
					res => {
						trendingCache.set(provider, res)
						if (!abort.signal.aborted) {
							setResults(res)
						}
					},
					err => !abort.signal.aborted && setError(err),
				)
				return () => abort.abort()
			}
		}
		const abort = new AbortController()
		const timeout = setTimeout(() => {
			searchGIF(provider, query, abort.signal).then(
				setResults,
				err => !abort.signal.aborted && setError(err),
			)
		}, 500)
		return () => {
			clearTimeout(timeout)
			abort.abort()
		}
	}, [query, provider])
	let poweredBySrc: string | undefined
	if (provider === "giphy") {
		poweredBySrc = "images/powered-by-giphy.png"
	} else if (provider === "tenor") {
		poweredBySrc = "images/powered-by-tenor.svg"
	}
	return <div className="gif-picker" style={style}>
		<div className="gif-search">
			<input
				autoFocus={!isMobileDevice}
				onChange={evt => setQuery(evt.target.value)}
				value={query}
				type="search"
				placeholder={`Search ${providerName}`}
			/>
			<button onClick={() => setQuery("")} disabled={query === ""}>
				{query !== "" ? <CloseIcon/> : <SearchIcon/>}
			</button>
		</div>
		{error ? <div className="gif-error">
			{`${error}`}
		</div> : null}
		<div className="gif-list">
			{results.map((gif, idx) => <div
				className="gif-entry"
				key={gif.key}
				data-gif-index={idx}
				onClick={onSelectGIF}
			>
				<img loading="lazy" src={gif.https_url} alt={gif.alt_text}/>
			</div>)}
			{poweredBySrc && <div className="powered-by-footer">
				<img src={poweredBySrc} alt={`Powered by ${providerName}`}/>
			</div>}
		</div>
	</div>
}

export default GIFPicker
