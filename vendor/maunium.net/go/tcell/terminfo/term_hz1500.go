// Generated automatically.  DO NOT HAND-EDIT.

package terminfo

func init() {
	// hazeltine 1500
	AddTerminfo(&Terminfo{
		Name:          "hz1500",
		Columns:       80,
		Lines:         24,
		Bell:          "\a",
		Clear:         "~\x1c",
		Italic:        "\x1b[3m",
		Strikethrough: "\x1b[9m",
		PadChar:       "\x00",
		SetCursor:     "~\x11%p2%p2%?%{30}%>%t%' '%+%;%'`'%+%c%p1%'`'%+%c",
		CursorBack1:   "\b",
		CursorUp1:     "~\f",
		KeyUp:         "~\f",
		KeyDown:       "\n",
		KeyRight:      "\x10",
		KeyLeft:       "\b",
		KeyHome:       "~\x12",
	})
}
