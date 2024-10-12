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

// From matrix-react-sdk, Copyright 2024 The Matrix.org Foundation C.I.C.
// Originally licensed under the Apache License, Version 2.0
// https://github.com/matrix-org/matrix-react-sdk/blob/develop/src/Linkify.tsx#L245
import sanitizeHtml from "sanitize-html"
import { getMediaURL } from "../api/media.ts"
import { calculateMediaSize } from "./mediasize.ts"

const COLOR_REGEX = /^#[0-9a-fA-F]{6}$/

export const PERMITTED_URL_SCHEMES = [
	"bitcoin",
	"ftp",
	"geo",
	"http",
	"https",
	"im",
	"irc",
	"ircs",
	"magnet",
	"mailto",
	"matrix",
	"mms",
	"news",
	"nntp",
	"openpgp4fpr",
	"sip",
	"sftp",
	"sms",
	"smsto",
	"ssh",
	"tel",
	"urn",
	"webcal",
	"wtai",
	"xmpp",
]

export const transformTags: NonNullable<sanitizeHtml.IOptions["transformTags"]> = {
	"a": function(tagName: string, attribs: sanitizeHtml.Attributes) {
		if (attribs.href) {
			attribs.target = "_blank"
		} else {
			// Delete the href attrib if it is falsy
			delete attribs.href
		}

		attribs.rel = "noreferrer noopener" // https://mathiasbynens.github.io/rel-noopener/
		return { tagName, attribs }
	},
	"img": function(tagName: string, attribs: sanitizeHtml.Attributes) {
		const src = attribs.src
		if (!src.startsWith("mxc://")) {
			return {
				tagName,
				attribs: {},
			}
		}

		const requestedWidth = Number(attribs.width)
		const requestedHeight = Number(attribs.height)
		if (requestedHeight && requestedHeight <= 48) {
			attribs.style = `height: ${requestedHeight}px; width: auto; max-width: ${2 * requestedHeight}px;`
		}
		const style = calculateMediaSize(requestedWidth, requestedHeight)
		if (style.media.aspectRatio) {
			attribs.style = `width: ${style.container.width}; height: ${style.container.height};`
		} else {
			attribs.style = `height: 24px; width: auto; max-width: 48px;`
		}

		attribs.src = getMediaURL(src)!
		attribs.loading = "lazy"
		return { tagName, attribs }
	},
	"code": function(tagName: string, attribs: sanitizeHtml.Attributes) {
		if (typeof attribs.class !== "undefined") {
			// Filter out all classes other than ones starting with language- for syntax highlighting.
			const classes = attribs.class.split(/\s/).filter(function(cl) {
				return cl.startsWith("language-") && !cl.startsWith("language-_")
			})
			attribs.class = classes.join(" ")
		}
		return { tagName, attribs }
	},
	"*": function(tagName: string, attribs: sanitizeHtml.Attributes) {
		// Delete any style previously assigned, style is an allowedTag for font, span & img,
		// because attributes are stripped after transforming.
		// For img this is trusted as it is generated wholly within the img transformation method.
		if (tagName !== "img") {
			delete attribs.style
		}

		if (tagName === "span") {
			attribs.title = attribs["data-mx-spoiler"]
		}

		// Sanitise and transform data-mx-color and data-mx-bg-color to their CSS
		// equivalents
		const customCSSMapper: Record<string, string> = {
			"data-mx-color": "color",
			"data-mx-bg-color": "background-color",
			// $customAttributeKey: $cssAttributeKey
		}

		let style = ""
		for (const [customAttributeKey, cssAttributeKey] of Object.entries(customCSSMapper)) {
			const customAttributeValue = attribs[customAttributeKey]
			if (
				customAttributeValue &&
				typeof customAttributeValue === "string" &&
				COLOR_REGEX.test(customAttributeValue)
			) {
				style += cssAttributeKey + ":" + customAttributeValue + ";"
				delete attribs[customAttributeKey]
			}
		}

		if (style) {
			attribs.style = style + (attribs.style || "")
		}

		return { tagName, attribs }
	},
}

export const sanitizeHtmlParams: sanitizeHtml.IOptions = {
	allowedTags: [
		// These tags are suggested by the spec https://spec.matrix.org/v1.12/client-server-api/#mroommessage-msgtypes
		"font",
		"del",
		"s",
		"h1",
		"h2",
		"h3",
		"h4",
		"h5",
		"h6",
		"blockquote",
		"p",
		"a",
		"ul",
		"ol",
		"sup",
		"sub",
		"nl",
		"li",
		"b",
		"i",
		"u",
		"strong",
		"em",
		"strike",
		"code",
		"hr",
		"br",
		"div",
		"table",
		"thead",
		"caption",
		"tbody",
		"tr",
		"th",
		"td",
		"pre",
		"span",
		"img",
		"details",
		"summary",
	],
	allowedAttributes: {
		// attribute sanitization happens after transformations, so we have to accept `style` for font, span & img
		// but strip during the transformation.
		// custom ones first:
		font: ["color", "data-mx-bg-color", "data-mx-color", "style"], // custom to matrix
		span: [
			"data-mx-maths", "data-mx-bg-color", "data-mx-color", "data-mx-spoiler", "style", "title",
		], // custom to matrix
		div: ["data-mx-maths"],
		// eslint-disable-next-line id-length
		a: ["href", "name", "target", "rel"], // remote target: custom to matrix
		// img tags also accept width/height, we just map those to max-width & max-height during transformation
		img: ["src", "alt", "title", "style", "loading", "data-mx-emoticon"],
		ol: ["start"],
		code: ["class"], // We don't actually allow all classes, we filter them in transformTags
	},
	// Lots of these won't come up by default because we don't allow them
	selfClosing: ["img", "br", "hr", "area", "base", "basefont", "input", "link", "meta"],
	// URL schemes we permit
	allowedSchemes: PERMITTED_URL_SCHEMES,
	allowProtocolRelative: false,
	transformTags,
	// 50 levels deep "should be enough for anyone"
	nestingLimit: 50,
}
