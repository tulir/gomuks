// Licensed under the GNU Free Documentation License 1.2
// https://www.gnu.org/licenses/old-licenses/fdl-1.2.en.html
//
// Source: https://rosettacode.org/wiki/Longest_common_prefix#Go

package util

func LongestCommonPrefix(list []string) string {
	// Special cases first
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	}

	// LCP of min and max (lexigraphically)
	// is the LCP of the whole set.
	min, max := list[0], list[0]
	for _, s := range list[1:] {
		switch {
		case s < min:
			min = s
		case s > max:
			max = s
		}
	}

	for i := 0; i < len(min) && i < len(max); i++ {
		if min[i] != max[i] {
			return min[:i]
		}
	}

	// In the case where lengths are not equal but all bytes
	// are equal, min is the answer ("foo" < "foobar").
	return min
}
