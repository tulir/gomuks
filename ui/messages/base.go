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

package messages

import (
	"fmt"
	"sort"
	"time"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/mauview"
	"go.mau.fi/tcell"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/matrix/muksevt"

	"maunium.net/go/gomuks/ui/widget"
)

type MessageRenderer interface {
	Draw(screen mauview.Screen, msg *UIMessage)
	NotificationContent() string
	PlainText() string
	CalculateBuffer(prefs config.UserPreferences, width int, msg *UIMessage)
	Height() int
	Clone() MessageRenderer
	String() string
}

type ReactionItem struct {
	Key   string
	Count int
}

func (ri ReactionItem) String() string {
	return fmt.Sprintf("%d×%s", ri.Count, ri.Key)
}

type ReactionSlice []ReactionItem

func (rs ReactionSlice) Len() int {
	return len(rs)
}

func (rs ReactionSlice) Less(i, j int) bool {
	return rs[i].Key < rs[j].Key
}

func (rs ReactionSlice) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

type UIMessage struct {
	EventID            id.EventID
	TxnID              string
	Relation           event.RelatesTo
	Type               event.MessageType
	SenderID           id.UserID
	SenderName         string
	DefaultSenderColor tcell.Color
	Timestamp          time.Time
	State              muksevt.OutgoingState
	IsHighlight        bool
	IsService          bool
	IsSelected         bool
	Edited             bool
	Event              *muksevt.Event
	ReplyTo            *UIMessage
	Reactions          ReactionSlice
	Renderer           MessageRenderer
}

func (msg *UIMessage) GetEvent() *muksevt.Event {
	if msg == nil {
		return nil
	}
	return msg.Event
}

const DateFormat = "January _2, 2006"
const TimeFormat = "15:04:05"

func newUIMessage(evt *muksevt.Event, displayname string, renderer MessageRenderer) *UIMessage {
	msgContent := evt.Content.AsMessage()
	msgtype := msgContent.MsgType
	if len(msgtype) == 0 {
		msgtype = event.MessageType(evt.Type.String())
	}

	reactions := make(ReactionSlice, 0, len(evt.Unsigned.Relations.Annotations.Map))
	for key, count := range evt.Unsigned.Relations.Annotations.Map {
		reactions = append(reactions, ReactionItem{
			Key:   key,
			Count: count,
		})
	}
	sort.Sort(reactions)

	return &UIMessage{
		SenderID:           evt.Sender,
		SenderName:         displayname,
		Timestamp:          unixToTime(evt.Timestamp),
		DefaultSenderColor: widget.GetHashColor(evt.Sender),
		Type:               msgtype,
		EventID:            evt.ID,
		TxnID:              evt.Unsigned.TransactionID,
		Relation:           *msgContent.GetRelatesTo(),
		State:              evt.Gomuks.OutgoingState,
		IsHighlight:        false,
		IsService:          false,
		Edited:             len(evt.Gomuks.Edits) > 0,
		Reactions:          reactions,
		Event:              evt,
		Renderer:           renderer,
	}
}

func (msg *UIMessage) AddReaction(key string) {
	found := false
	for i, rs := range msg.Reactions {
		if rs.Key == key {
			rs.Count++
			msg.Reactions[i] = rs
			found = true
			break
		}
	}
	if !found {
		msg.Reactions = append(msg.Reactions, ReactionItem{
			Key:   key,
			Count: 1,
		})
	}
	sort.Sort(msg.Reactions)
}

func unixToTime(unix int64) time.Time {
	timestamp := time.Now()
	if unix != 0 {
		timestamp = time.Unix(unix/1000, unix%1000*1000)
	}
	return timestamp
}

// Sender gets the string that should be displayed as the sender of this message.
//
// If the message is being sent, the sender is "Sending...".
// If sending has failed, the sender is "Error".
// If the message is an emote, the sender is blank.
// In any other case, the sender is the display name of the user who sent the message.
func (msg *UIMessage) Sender() string {
	switch msg.State {
	case muksevt.StateLocalEcho:
		return "Sending..."
	case muksevt.StateSendFail:
		return "Error"
	}
	switch msg.Type {
	case "m.emote":
		// Emotes don't show a separate sender, it's included in the buffer.
		return ""
	default:
		return msg.SenderName
	}
}

func (msg *UIMessage) NotificationSenderName() string {
	return msg.SenderName
}

func (msg *UIMessage) NotificationContent() string {
	return msg.Renderer.NotificationContent()
}

func (msg *UIMessage) getStateSpecificColor() tcell.Color {
	switch msg.State {
	case muksevt.StateLocalEcho:
		return tcell.ColorGray
	case muksevt.StateSendFail:
		return tcell.ColorRed
	case muksevt.StateDefault:
		fallthrough
	default:
		return tcell.ColorDefault
	}
}

// SenderColor returns the color the name of the sender should be shown in.
//
// If the message is being sent, the color is gray.
// If sending has failed, the color is red.
//
// In any other case, the color is whatever is specified in the Message struct.
// Usually that means it is the hash-based color of the sender (see ui/widget/color.go)
func (msg *UIMessage) SenderColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	case msg.Type == "m.room.member":
		return widget.GetHashColor(msg.SenderName)
	case msg.IsService:
		return tcell.ColorGray
	default:
		return msg.DefaultSenderColor
	}
}

// TextColor returns the color the actual content of the message should be shown in.
func (msg *UIMessage) TextColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	case msg.IsService, msg.Type == "m.notice":
		return tcell.ColorGray
	case msg.IsHighlight:
		return tcell.ColorYellow
	case msg.Type == "m.room.member":
		return tcell.ColorGreen
	default:
		return tcell.ColorDefault
	}
}

// TimestampColor returns the color the timestamp should be shown in.
//
// As with SenderColor(), messages being sent and messages that failed to be sent are
// gray and red respectively.
//
// However, other messages are the default color instead of a color stored in the struct.
func (msg *UIMessage) TimestampColor() tcell.Color {
	if msg.IsService {
		return tcell.ColorGray
	}
	return msg.getStateSpecificColor()
}

func (msg *UIMessage) ReplyHeight() int {
	if msg.ReplyTo != nil {
		return 1 + msg.ReplyTo.Height()
	}
	return 0
}

func (msg *UIMessage) ReactionHeight() int {
	if len(msg.Reactions) > 0 {
		return 1
	}
	return 0
}

// Height returns the number of rows in the computed buffer (see Buffer()).
func (msg *UIMessage) Height() int {
	return msg.ReplyHeight() + msg.Renderer.Height() + msg.ReactionHeight()
}

func (msg *UIMessage) Time() time.Time {
	return msg.Timestamp
}

// FormatTime returns the formatted time when the message was sent.
func (msg *UIMessage) FormatTime() string {
	return msg.Timestamp.Format(TimeFormat)
}

// FormatDate returns the formatted date when the message was sent.
func (msg *UIMessage) FormatDate() string {
	return msg.Timestamp.Format(DateFormat)
}

func (msg *UIMessage) SameDate(message *UIMessage) bool {
	year1, month1, day1 := msg.Timestamp.Date()
	year2, month2, day2 := message.Timestamp.Date()
	return day1 == day2 && month1 == month2 && year1 == year2
}

func (msg *UIMessage) ID() id.EventID {
	if len(msg.EventID) == 0 {
		return id.EventID(msg.TxnID)
	}
	return msg.EventID
}

func (msg *UIMessage) SetID(id id.EventID) {
	msg.EventID = id
}

func (msg *UIMessage) SetIsHighlight(isHighlight bool) {
	msg.IsHighlight = isHighlight
}

func (msg *UIMessage) DrawReactions(screen mauview.Screen) {
	if len(msg.Reactions) == 0 {
		return
	}
	width, height := screen.Size()
	screen = mauview.NewProxyScreen(screen, 0, height-1, width, 1)

	x := 0
	for _, reaction := range msg.Reactions {
		_, drawn := mauview.PrintWithStyle(screen, reaction.String(), x, 0, width-x, mauview.AlignLeft, tcell.StyleDefault.Foreground(mauview.Styles.PrimaryTextColor).Background(tcell.ColorDarkGreen))
		x += drawn + 1
		if x >= width {
			break
		}
	}
}

func (msg *UIMessage) Draw(screen mauview.Screen) {
	proxyScreen := msg.DrawReply(screen)
	msg.Renderer.Draw(proxyScreen, msg)
	msg.DrawReactions(proxyScreen)
	if msg.IsSelected {
		w, h := screen.Size()
		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				mainc, combc, style, _ := screen.GetContent(x, y)
				_, bg, _ := style.Decompose()
				if bg == tcell.ColorDefault {
					screen.SetContent(x, y, mainc, combc, style.Background(tcell.ColorDarkGreen))
				}
			}
		}
	}
}

func (msg *UIMessage) Clone() *UIMessage {
	clone := *msg
	clone.ReplyTo = nil
	clone.Reactions = nil
	clone.Renderer = clone.Renderer.Clone()
	return &clone
}

func (msg *UIMessage) CalculateReplyBuffer(preferences config.UserPreferences, width int) {
	if msg.ReplyTo == nil {
		return
	}
	msg.ReplyTo.CalculateBuffer(preferences, width-1)
}

func (msg *UIMessage) CalculateBuffer(preferences config.UserPreferences, width int) {
	msg.Renderer.CalculateBuffer(preferences, width, msg)
	msg.CalculateReplyBuffer(preferences, width)
}

func (msg *UIMessage) DrawReply(screen mauview.Screen) mauview.Screen {
	if msg.ReplyTo == nil {
		return screen
	}
	width, height := screen.Size()
	replyHeight := msg.ReplyTo.Height()
	widget.WriteLineSimpleColor(screen, "In reply to", 1, 0, tcell.ColorGreen)
	widget.WriteLineSimpleColor(screen, msg.ReplyTo.SenderName, 13, 0, msg.ReplyTo.SenderColor())
	for y := 0; y < 1+replyHeight; y++ {
		screen.SetCell(0, y, tcell.StyleDefault, '▊')
	}
	replyScreen := mauview.NewProxyScreen(screen, 1, 1, width-1, replyHeight)
	msg.ReplyTo.Draw(replyScreen)
	return mauview.NewProxyScreen(screen, 0, replyHeight+1, width, height-replyHeight-1)
}

func (msg *UIMessage) String() string {
	return fmt.Sprintf(`&messages.UIMessage{
    ID="%s", TxnID="%s",
    Type="%s", Timestamp=%s,
    Sender={ID="%s", Name="%s", Color=#%X},
    IsService=%t, IsHighlight=%t,
    Renderer=%s,
}`,
		msg.EventID, msg.TxnID,
		msg.Type, msg.Timestamp.String(),
		msg.SenderID, msg.SenderName, msg.DefaultSenderColor.Hex(),
		msg.IsService, msg.IsHighlight, msg.Renderer.String())
}

func (msg *UIMessage) PlainText() string {
	return msg.Renderer.PlainText()
}
