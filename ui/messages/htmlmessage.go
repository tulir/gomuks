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
	"math"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"maunium.net/go/mautrix"
	"maunium.net/go/mauview"
	"maunium.net/go/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/ui/widget"
)

type HTMLMessage struct {
	BaseMessage

	Root HTMLEntity

	FocusedBackground tcell.Color

	focused bool
}

func NewHTMLMessage(id, sender, displayname string, msgtype mautrix.MessageType, root HTMLEntity, timestamp time.Time) UIMessage {
	return &HTMLMessage{
		BaseMessage: newBaseMessage(id, sender, displayname, msgtype, timestamp),
		Root:        root,
	}
}
func (hw *HTMLMessage) Draw(screen mauview.Screen) {
	if hw.focused {
		screen.SetStyle(tcell.StyleDefault.Background(hw.FocusedBackground))
	}
	screen.Clear()
	hw.Root.Draw(screen)
}

func (hw *HTMLMessage) Focus() {
	hw.focused = true
}

func (hw *HTMLMessage) Blur() {
	hw.focused = false
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
}

func (hw *HTMLMessage) Height() int {
	return hw.Root.Height()
}

func (hw *HTMLMessage) PlainText() string {
	return hw.Root.PlainText()
}

func (hw *HTMLMessage) NotificationContent() string {
	return hw.Root.PlainText()
}

type AdjustStyleFunc func(tcell.Style) tcell.Style

type HTMLEntity interface {
	AdjustStyle(AdjustStyleFunc) HTMLEntity
	Draw(screen mauview.Screen)
	IsBlock() bool
	GetTag() string
	PlainText() string
	String() string
	Height() int

	calculateBuffer(width, startX int, bare bool) int
	getStartX() int
}

type BlockquoteEntity struct {
	*BaseHTMLEntity
}

func NewBlockquoteEntity(children []HTMLEntity) *BlockquoteEntity {
	return &BlockquoteEntity{&BaseHTMLEntity{
		Tag:      "blockquote",
		Children: children,
		Block:    true,
		Indent:   2,
	}}
}

func (be *BlockquoteEntity) Draw(screen mauview.Screen) {
	be.BaseHTMLEntity.Draw(screen)
	for y := 0; y < be.height; y++ {
		screen.SetContent(0, y, '>', nil, be.Style)
	}
}

func (be *BlockquoteEntity) String() string {
	return fmt.Sprintf("&BlockquoteEntity{%s},\n", be.BaseHTMLEntity)
}

type ListEntity struct {
	*BaseHTMLEntity
	Ordered bool
	Start   int
}

func digits(num int) int {
	if num <= 0 {
		return 0
	}
	return int(math.Floor(math.Log10(float64(num))) + 1)
}

func NewListEntity(ordered bool, start int, children []HTMLEntity) *ListEntity {
	entity := &ListEntity{
		BaseHTMLEntity: &BaseHTMLEntity{
			Tag:      "ul",
			Children: children,
			Block:    true,
			Indent:   2,
		},
		Ordered: ordered,
		Start:   start,
	}
	if ordered {
		entity.Tag = "ol"
		entity.Indent += digits(start + len(children) - 1)
	}
	return entity
}

func (le *ListEntity) Draw(screen mauview.Screen) {
	width, _ := screen.Size()

	proxyScreen := &mauview.ProxyScreen{Parent: screen, OffsetX: le.Indent, Width: width - le.Indent, Style: le.Style}
	for i, entity := range le.Children {
		proxyScreen.Height = entity.Height()
		if le.Ordered {
			number := le.Start + i
			line := fmt.Sprintf("%d. %s", number, strings.Repeat(" ", le.Indent-2-digits(number)))
			widget.WriteLine(screen, mauview.AlignLeft, line, 0, proxyScreen.OffsetY, le.Indent, le.Style)
		} else {
			screen.SetContent(0, proxyScreen.OffsetY, '●', nil, le.Style)
		}
		entity.Draw(proxyScreen)
		proxyScreen.SetStyle(le.Style)
		proxyScreen.OffsetY += entity.Height()
	}
}

func (le *ListEntity) String() string {
	return fmt.Sprintf("&ListEntity{Ordered=%t, Start=%d, Base=%s},\n", le.Ordered, le.Start, le.BaseHTMLEntity)
}

type CodeBlockEntity struct {
	*BaseHTMLEntity
	Background tcell.Style
}

func NewCodeBlockEntity(children []HTMLEntity, background tcell.Style) *CodeBlockEntity {
	return &CodeBlockEntity{
		BaseHTMLEntity: &BaseHTMLEntity{
			Tag:      "pre",
			Block:    true,
			Children: children,
		},
		Background: background,
	}
}

func (ce *CodeBlockEntity) Draw(screen mauview.Screen) {
	screen.Fill(' ', ce.Background)
	ce.BaseHTMLEntity.Draw(screen)
}

func (ce *CodeBlockEntity) AdjustStyle(fn AdjustStyleFunc) HTMLEntity {
	return ce
}

type BreakEntity struct {
	*BaseHTMLEntity
}

func NewBreakEntity() *BreakEntity {
	return &BreakEntity{&BaseHTMLEntity{
		Tag:   "br",
		Block: true,
	}}
}

type BaseHTMLEntity struct {
	// Permanent variables
	Tag      string
	Text     string
	Style    tcell.Style
	Children []HTMLEntity
	Block    bool
	Indent   int

	DefaultHeight int

	// Non-permanent variables (calculated buffer data)
	buffer    []string
	prevWidth int
	startX    int
	height    int
}

func NewHTMLTextEntity(text string) *BaseHTMLEntity {
	return &BaseHTMLEntity{
		Tag:  "text",
		Text: text,
	}
}

func NewHTMLEntity(tag string, children []HTMLEntity, block bool) *BaseHTMLEntity {
	return &BaseHTMLEntity{
		Tag:      tag,
		Children: children,
		Block:    block,
	}
}

func (he *BaseHTMLEntity) AdjustStyle(fn AdjustStyleFunc) HTMLEntity {
	for _, child := range he.Children {
		child.AdjustStyle(fn)
	}
	he.Style = fn(he.Style)
	return he
}

func (he *BaseHTMLEntity) IsBlock() bool {
	return he.Block
}

func (he *BaseHTMLEntity) GetTag() string {
	return he.Tag
}

func (he *BaseHTMLEntity) Height() int {
	return he.height
}

func (he *BaseHTMLEntity) getStartX() int {
	return he.startX
}

func (he *BaseHTMLEntity) String() string {
	var buf strings.Builder
	buf.WriteString("&BaseHTMLEntity{\n")
	_, _ = fmt.Fprintf(&buf, `    Tag="%s", Style=%d, Block=%t, Indent=%d, startX=%d, height=%d,`,
		he.Tag, he.Style, he.Block, he.Indent, he.startX, he.height)
	buf.WriteRune('\n')
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

func (he *BaseHTMLEntity) PlainText() string {
	if len(he.Children) == 0 {
		return he.Text
	}
	var buf strings.Builder
	buf.WriteString(he.Text)
	newlined := false
	for _, child := range he.Children {
		if child.IsBlock() && !newlined {
			buf.WriteRune('\n')
		}
		newlined = false
		buf.WriteString(child.PlainText())
		if child.IsBlock() {
			buf.WriteRune('\n')
			newlined = true
		}
	}
	return buf.String()
}

func (he *BaseHTMLEntity) Draw(screen mauview.Screen) {
	width, _ := screen.Size()
	if len(he.buffer) > 0 {
		x := he.startX
		for y, line := range he.buffer {
			widget.WriteLine(screen, mauview.AlignLeft, line, x, y, width, he.Style)
			x = 0
		}
	}
	if len(he.Children) > 0 {
		prevBreak := false
		proxyScreen := &mauview.ProxyScreen{Parent: screen, OffsetX: he.Indent, Width: width - he.Indent, Style: he.Style}
		for i, entity := range he.Children {
			if i != 0 && entity.getStartX() == 0 {
				proxyScreen.OffsetY++
			}
			proxyScreen.Height = entity.Height()
			entity.Draw(proxyScreen)
			proxyScreen.SetStyle(he.Style)
			proxyScreen.OffsetY += entity.Height() - 1
			_, isBreak := entity.(*BreakEntity)
			if prevBreak && isBreak {
				proxyScreen.OffsetY++
			}
			prevBreak = isBreak
		}
	}
}

func (he *BaseHTMLEntity) calculateBuffer(width, startX int, bare bool) int {
	he.startX = startX
	if he.Block {
		he.startX = 0
	}
	he.height = 0
	if len(he.Children) > 0 {
		childStartX := he.startX
		prevBreak := false
		for _, entity := range he.Children {
			if entity.IsBlock() || childStartX == 0 || he.height == 0 {
				he.height++
			}
			childStartX = entity.calculateBuffer(width-he.Indent, childStartX, bare)
			he.height += entity.Height() - 1
			_, isBreak := entity.(*BreakEntity)
			if prevBreak && isBreak {
				he.height++
			}
			prevBreak = isBreak
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
			// TODO add option no wrap and character wrap options
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
	if len(he.Text) == 0 && len(he.Children) == 0 {
		he.height = he.DefaultHeight
	}
	return he.startX
}

func trim(extract, full string, bare bool) (string, bool) {
	if len(extract) == len(full) {
		return extract, true
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
				return extract, true
			}
		}
	}
	return extract, len(extract) > 0 && extract[len(extract)-1] == ' '
}
