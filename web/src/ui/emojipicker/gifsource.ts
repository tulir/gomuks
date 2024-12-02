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
import { ContentURI } from "@/api/types"
import { GIFProvider } from "@/api/types/preferences"
import { GIPHY_API_KEY, TENOR_API_KEY } from "@/util/keys.ts"

export interface GIF {
	key: string
	filename: string
	title: string
	alt_text: string
	proxied_mxc: ContentURI
	https_url: string
	width: number
	height: number
	size: number
}

function mapGiphyResults(results: unknown[]): GIF[] {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	return results.map((entry: any): GIF => ({
		key: entry.id,
		filename: `${entry.slug}.webp`,
		title: entry.title,
		alt_text: entry.alt_text,
		proxied_mxc: `mxc://giphy.mau.dev/${entry.id}`,
		https_url: entry.images.original.webp,
		size: entry.images.original.webp_size,
		width: entry.images.original.width,
		height: entry.images.original.height,
	}))
}

const tenorMediaURLRegex = /https:\/\/media\.tenor\.com\/([A-Za-z0-9_-]+)\/.+/

function mapTenorResults(results: unknown[]): GIF[] {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	return results.map((entry: any): GIF | undefined => {
		const id = tenorMediaURLRegex.exec(entry.media_formats.webp.url)?.[1]
		if (!id) {
			return
		}
		return {
			key: entry.id,
			filename: `${entry.id}.webp`,
			title: entry.title,
			alt_text: entry.alt_text,
			proxied_mxc: `mxc://tenor.mau.dev/${id}`,
			https_url: entry.media_formats.webp.url,
			size: entry.media_formats.webp.size,
			width: entry.media_formats.webp.dims[0],
			height: entry.media_formats.webp.dims[1],
		}
	}).filter((entry: GIF | undefined): entry is GIF => !!entry)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function doRequest(url: URL, signal?: AbortSignal): Promise<any> {
	const resp = await fetch(url, { signal })
	if (resp.status !== 200) {
		throw new Error(`HTTP ${resp.status}: ${await resp.text()}`)
	}
	return await resp.json()
}

async function searchGiphy(signal: AbortSignal, query: string): Promise<GIF[]> {
	const url = new URL("https://api.giphy.com/v1/gifs/search")
	url.searchParams.set("api_key", GIPHY_API_KEY)
	url.searchParams.set("q", query)
	url.searchParams.set("limit", "50")
	return mapGiphyResults((await doRequest(url, signal)).data)
}

async function searchTenor(signal: AbortSignal, query: string): Promise<GIF[]> {
	const url = new URL("https://tenor.googleapis.com/v2/search")
	url.searchParams.set("key", TENOR_API_KEY)
	url.searchParams.set("media_filter", "webp")
	url.searchParams.set("q", query)
	url.searchParams.set("limit", "50")
	return mapTenorResults((await doRequest(url, signal)).results)
}

async function getGiphyTrending(): Promise<GIF[]> {
	const url = new URL("https://api.giphy.com/v1/gifs/trending")
	url.searchParams.set("api_key", GIPHY_API_KEY)
	url.searchParams.set("limit", "50")
	return mapGiphyResults((await doRequest(url)).data)
}

async function getTenorTrending(): Promise<GIF[]> {
	const url = new URL("https://tenor.googleapis.com/v2/featured")
	url.searchParams.set("key", TENOR_API_KEY)
	url.searchParams.set("media_filter", "webp")
	url.searchParams.set("limit", "50")
	return mapTenorResults((await doRequest(url)).results)
}

const searchFuncs = {
	giphy: searchGiphy,
	tenor: searchTenor,
}

const trendingFuncs = {
	giphy: getGiphyTrending,
	tenor: getTenorTrending,
}

export async function searchGIF(provider: GIFProvider, query: string, signal: AbortSignal): Promise<GIF[]> {
	return searchFuncs[provider](signal, query)
}

export async function getTrendingGIFs(provider: GIFProvider): Promise<GIF[]> {
	return trendingFuncs[provider]()
}
