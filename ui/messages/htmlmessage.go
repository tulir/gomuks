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
	// TODO account for bare messages in initial startX
	startX := 0
	hw.Root.calculateBuffer(width, startX, preferences.BareMessageView)
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
		for _, entity := range he.Children {
			if entity.Block {
				proxyScreen.OffsetY++
			}
			proxyScreen.Height = entity.height
			entity.Draw(proxyScreen)
			proxyScreen.OffsetY += entity.height - 1
		}
	}
}

func (he *HTMLEntity) calculateBuffer(width, startX int, bare bool) int {
	if len(he.Children) > 0 {
		childStartX := 0
		for _, entity := range he.Children {
			childStartX = entity.calculateBuffer(width-he.Indent, childStartX, bare)
			he.height += entity.height - 1
		}
	}
	if len(he.Text) > 0 && width != he.prevWidth {
		he.prevWidth = width
		he.buffer = make([]string, 0, 1)
		text := he.Text
		if !he.Block {
			he.startX = startX
		} else {
			startX = 0
		}
		for {
			extract := runewidth.Truncate(text, width-startX, "")
			extract = trim(extract, text, bare)
			he.buffer = append(he.buffer, extract)
			text = text[len(extract):]
			startX = 0
			if len(text) == 0 {
				he.height += len(he.buffer)
				// This entity is over, return the startX for the next entity
				if he.Block {
					// ...except if it's a block entity
					return 0
				}
				return runewidth.StringWidth(extract)
			}
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

func trim(extract, full string, bare bool) string {
	if len(extract) == len(full) {
		return extract
	}
	if spaces := spacePattern.FindStringIndex(full[len(extract):]); spaces != nil && spaces[0] == 0 {
		extract = full[:len(extract)+spaces[1]]
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
			}
		}
	}
	return extract
}
