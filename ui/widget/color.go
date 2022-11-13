// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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

package widget

import (
	"fmt"
	"hash/fnv"

	"go.mau.fi/tcell"

	"maunium.net/go/mautrix/id"
)

var colorNames = []string{
	"maroon",
	"green",
	"olive",
	"navy",
	"purple",
	"teal",
	"silver",
	"gray",
	"red",
	"lime",
	"yellow",
	"blue",
	"fuchsia",
	"aqua",
	"white",
	"aliceblue",
	"antiquewhite",
	"aquamarine",
	"azure",
	"beige",
	"bisque",
	"blanchedalmond",
	"blueviolet",
	"brown",
	"burlywood",
	"cadetblue",
	"chartreuse",
	"chocolate",
	"coral",
	"cornflowerblue",
	"cornsilk",
	"crimson",
	"darkblue",
	"darkcyan",
	"darkgoldenrod",
	"darkgray",
	"darkgreen",
	"darkkhaki",
	"darkmagenta",
	"darkolivegreen",
	"darkorange",
	"darkorchid",
	"darkred",
	"darksalmon",
	"darkseagreen",
	"darkslateblue",
	"darkslategray",
	"darkturquoise",
	"darkviolet",
	"deeppink",
	"deepskyblue",
	"dimgray",
	"dodgerblue",
	"firebrick",
	"floralwhite",
	"forestgreen",
	"gainsboro",
	"ghostwhite",
	"gold",
	"goldenrod",
	"greenyellow",
	"honeydew",
	"hotpink",
	"indianred",
	"indigo",
	"ivory",
	"khaki",
	"lavender",
	"lavenderblush",
	"lawngreen",
	"lemonchiffon",
	"lightblue",
	"lightcoral",
	"lightcyan",
	"lightgoldenrodyellow",
	"lightgray",
	"lightgreen",
	"lightpink",
	"lightsalmon",
	"lightseagreen",
	"lightskyblue",
	"lightslategray",
	"lightsteelblue",
	"lightyellow",
	"limegreen",
	"linen",
	"mediumaquamarine",
	"mediumblue",
	"mediumorchid",
	"mediumpurple",
	"mediumseagreen",
	"mediumslateblue",
	"mediumspringgreen",
	"mediumturquoise",
	"mediumvioletred",
	"midnightblue",
	"mintcream",
	"mistyrose",
	"moccasin",
	"navajowhite",
	"oldlace",
	"olivedrab",
	"orange",
	"orangered",
	"orchid",
	"palegoldenrod",
	"palegreen",
	"paleturquoise",
	"palevioletred",
	"papayawhip",
	"peachpuff",
	"peru",
	"pink",
	"plum",
	"powderblue",
	"rebeccapurple",
	"rosybrown",
	"royalblue",
	"saddlebrown",
	"salmon",
	"sandybrown",
	"seagreen",
	"seashell",
	"sienna",
	"skyblue",
	"slateblue",
	"slategray",
	"snow",
	"springgreen",
	"steelblue",
	"tan",
	"thistle",
	"tomato",
	"turquoise",
	"violet",
	"wheat",
	"whitesmoke",
	"yellowgreen",
	"grey",
	"dimgrey",
	"darkgrey",
	"darkslategrey",
	"lightgrey",
	"lightslategrey",
	"slategrey",
}

// GetHashColorName gets a color name for the given string based on its FNV-1 hash.
//
// The array of possible color names are the alphabetically ordered color
// names specified in tcell.ColorNames.
//
// The algorithm to get the color is as follows:
//
//	colorNames[ FNV1(string) % len(colorNames) ]
//
// With the exception of the three special cases:
//
//	--> = green
//	<-- = red
//	--- = yellow
func GetHashColorName(s string) string {
	switch s {
	case "-->":
		return "green"
	case "<--":
		return "red"
	case "---":
		return "yellow"
	default:
		h := fnv.New32a()
		_, _ = h.Write([]byte(s))
		return colorNames[h.Sum32()%uint32(len(colorNames))]
	}
}

// GetHashColor gets the tcell Color value for the given string.
//
// GetHashColor calls GetHashColorName() and gets the Color value from the tcell.ColorNames map.
func GetHashColor(val interface{}) tcell.Color {
	switch str := val.(type) {
	case string:
		return tcell.ColorNames[GetHashColorName(str)]
	case *string:
		return tcell.ColorNames[GetHashColorName(*str)]
	case id.UserID:
		return tcell.ColorNames[GetHashColorName(string(str))]
	default:
		return tcell.ColorNames["red"]
	}
}

// AddColor adds tview color tags to the given string.
func AddColor(s, color string) string {
	return fmt.Sprintf("[%s]%s[white]", color, s)
}
