// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2019 Tulir Asokan
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

package messages

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/ui/widget"
	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"
)

type HTMLMessage struct {
	BaseMessage

	Root *HTMLEntity
}

func NewHTMLMessage(id, sender, displayname string, msgtype mautrix.MessageType, root *HTMLEntity, timestamp time.Time) UIMessage {
	return &HTMLMessage{
		BaseMessage: newBaseMessage(id, sender, displayname, msgtype, timestamp),
		Root:        root,
	}
}
func (hw *HTMLMessage) Draw(screen mauview.Screen) {
	hw.Root.Draw(screen)
}

func (hw *HTMLMessage) OnKeyEvent(event mauview.KeyEvent) bool {
	return false
}

func (hw *HTMLMessage) OnMouseEvent(event mauview.MouseEvent) bool {
	return false
}

func (hw *HTMLMessage) OnPasteEvent(event mauview.PasteEvent) bool {
	return false
}

func (hw *HTMLMessage) CalculateBuffer(preferences config.UserPreferences, width int) {
	if width <= 0 {
		panic("Negative width in CalculateBuffer")
	}
	// TODO account for bare messages in initial startX
	startX := 0
	hw.Root.calculateBuffer(width, startX, preferences.BareMessageView)
	//debug.Print(hw.Root.String())
}

func (hw *HTMLMessage) Height() int {
	return hw.Root.height
}

func (hw *HTMLMessage) PlainText() string {
	return "Plaintext unavailable"
}

func (hw *HTMLMessage) NotificationContent() string {
	return "Notification content unavailable"
}

type HTMLEntity struct {
	// Permanent variables
	Tag      string
	Text     string
	Style    tcell.Style
	Children []*HTMLEntity
	Block    bool
	Indent   int

	// Non-permanent variables (calculated buffer data)
	buffer    []string
	prevWidth int
	startX    int
	height    int
}

func (he *HTMLEntity) AdjustStyle(fn func(tcell.Style) tcell.Style) *HTMLEntity {
	for _, child := range he.Children {
		child.AdjustStyle(fn)
	}
	he.Style = fn(he.Style)
	return he
}

func (he *HTMLEntity) Draw(screen mauview.Screen) {
	width, _ := screen.Size()
	if len(he.buffer) > 0 {
		x := he.startX
		for y, line := range he.buffer {
			widget.WriteLine(screen, mauview.AlignLeft, line, x, y, width, he.Style)
			x = 0
		}
	}
	if len(he.Children) > 0 {
		proxyScreen := &mauview.ProxyScreen{Parent: screen, OffsetX: he.Indent, Width: width - he.Indent}
		for i, entity := range he.Children {
			if i != 0 && entity.startX == 0 {
				proxyScreen.OffsetY++
			}
			proxyScreen.Height = entity.height
			entity.Draw(proxyScreen)
			proxyScreen.OffsetY += entity.height - 1
		}
	}
}

func (he *HTMLEntity) String() string {
	var buf strings.Builder
	buf.WriteString("&HTMLEntity{\n")
	_, _ = fmt.Fprintf(&buf, `    Tag="%s", Style=%d, Block=%t, Indent=%d, startX=%d, height=%d,\n`,
		he.Tag, he.Style, he.Block, he.Indent, he.startX, he.height)
	_, _ = fmt.Fprintf(&buf, `    Buffer=["%s"]`, strings.Join(he.buffer, "\", \""))
	if len(he.Text) > 0 {
		buf.WriteString(",\n")
		_, _ = fmt.Fprintf(&buf, `    Text="%s"`, he.Text)
	}
	if len(he.Children) > 0 {
		buf.WriteString(",\n")
		buf.WriteString("    Children={")
		for _, child := range he.Children {
			buf.WriteString("\n        ")
			buf.WriteString(strings.Join(strings.Split(strings.TrimRight(child.String(), "\n"), "\n"), "\n        "))
		}
		buf.WriteString("\n    },")
	}
	buf.WriteString("\n},\n")
	return buf.String()
}

func (he *HTMLEntity) calculateBuffer(width, startX int, bare bool) int {
	he.startX = startX
	if he.Block {
		he.startX = 0
	}
	he.height = 0
	if len(he.Children) > 0 {
		childStartX := he.startX
		for _, entity := range he.Children {
			if entity.Block || childStartX == 0 || he.height == 0 {
				he.height++
			}
			childStartX = entity.calculateBuffer(width-he.Indent, childStartX, bare)
			he.height += entity.height - 1
		}
		if len(he.Text) == 0 && !he.Block {
			return childStartX
		}
	}
	if len(he.Text) > 0 {
		he.prevWidth = width
		if he.buffer == nil {
			he.buffer = []string{}
		}
		bufPtr := 0
		text := he.Text
		textStartX := he.startX
		for {
			extract := runewidth.Truncate(text, width-textStartX, "")
			extract, wordWrapped := trim(extract, text, bare)
			if !wordWrapped && textStartX > 0 {
				if bufPtr < len(he.buffer) {
					he.buffer[bufPtr] = ""
				} else {
					he.buffer = append(he.buffer, "")
				}
				bufPtr++
				textStartX = 0
				continue
			}
			if bufPtr < len(he.buffer) {
				he.buffer[bufPtr] = extract
			} else {
				he.buffer = append(he.buffer, extract)
			}
			bufPtr++
			text = text[len(extract):]
			if len(text) == 0 {
				he.buffer = he.buffer[:bufPtr]
				he.height += len(he.buffer)
				// This entity is over, return the startX for the next entity
				if he.Block {
					// ...except if it's a block entity
					return 0
				}
				return textStartX + runewidth.StringWidth(extract)
			}
			textStartX = 0
		}
	}
	return 0
}

// Regular expressions used to split lines when calculating the buffer.
/*var (
	boundaryPattern     = regexp.MustCompile(`([[:punct:]]\s*|\s+)`)
	bareBoundaryPattern = regexp.MustCompile(`(\s+)`)
	spacePattern        = regexp.MustCompile(`\s+`)
)*/

func trim(extract, full string, bare bool) (string, bool) {
	if len(extract) == len(full) {
		return extract, true
	}
	if spaces := spacePattern.FindStringIndex(full[len(extract):]); spaces != nil && spaces[0] == 0 {
		extract = full[:len(extract)+spaces[1]]
		//return extract, true
	}
	regex := boundaryPattern
	if bare {
		regex = bareBoundaryPattern
	}
	matches := regex.FindAllStringIndex(extract, -1)
	if len(matches) > 0 {
		if match := matches[len(matches)-1]; len(match) >= 2 {
			if until := match[1]; until < len(extract) {
				extract = extract[:until]
				return extract, true
			}
		}
	}
	return extract, len(extract) > 0 && extract[len(extract)-1] == ' '
}
