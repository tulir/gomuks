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

package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"unicode"

	"github.com/lucasb-eyer/go-colorful"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/format"

	"maunium.net/go/gomuks/debug"
)

func cmdMe(cmd *Command) {
	text := strings.Join(cmd.Args, " ")
	tempMessage := cmd.Room.NewTempMessage("m.emote", text)
	go cmd.MainView.sendTempMessage(cmd.Room, tempMessage, text)
	cmd.UI.Render()
}

// GradientTable from https://github.com/lucasb-eyer/go-colorful/blob/master/doc/gradientgen/gradientgen.go
type GradientTable []struct {
	Col colorful.Color
	Pos float64
}

func (gt GradientTable) GetInterpolatedColorFor(t float64) colorful.Color {
	for i := 0; i < len(gt)-1; i++ {
		c1 := gt[i]
		c2 := gt[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendHcl(c2.Col, t).Clamped()
		}
	}
	return gt[len(gt)-1].Col
}

var rainbow = GradientTable{
	{colorful.LinearRgb(1, 0, 0), 0.0},
	{colorful.LinearRgb(1, 0.5, 0), 0.1},
	{colorful.LinearRgb(0.5, 0.5, 0), 0.2}, // Yellow is 0.5, 0.5 instead of 1, 1 to make it readable on light themes
	{colorful.LinearRgb(0.5, 1, 0), 0.3},
	{colorful.LinearRgb(0, 1, 0), 0.4},
	{colorful.LinearRgb(0, 1, 0.5), 0.5},
	{colorful.LinearRgb(0, 1, 1), 0.6},
	{colorful.LinearRgb(0, 0.5, 1), 0.7},
	{colorful.LinearRgb(0.5, 0, 1), 0.8},
	{colorful.LinearRgb(1, 0, 1), 0.9},
	{colorful.LinearRgb(1, 0, 0.5), 1},
}

func cmdHeapProfile(cmd *Command) {
	runtime.GC()
	memProfile, err := os.Create("gomuks.prof")
	if err != nil {
		debug.Print(err)
	}
	defer memProfile.Close()
	if err := pprof.WriteHeapProfile(memProfile); err != nil {
		debug.Print(err)
	}
}

// TODO this command definitely belongs in a plugin once we have a plugin system.
func cmdRainbow(cmd *Command) {
	text := strings.Join(cmd.Args, " ")
	var html strings.Builder
	fmt.Fprint(&html, "**🌈** ")
	for i, char := range text {
		if unicode.IsSpace(char) {
			html.WriteRune(char)
			continue
		}
		color := rainbow.GetInterpolatedColorFor(float64(i) / float64(len(text))).Hex()
		fmt.Fprintf(&html, "<font color=\"%s\">%c</font>", color, char)
	}
	tempMessage := cmd.Room.NewTempMessage("m.text", format.HTMLToText(html.String()))
	go cmd.MainView.sendTempMessage(cmd.Room, tempMessage, html.String())
	cmd.UI.Render()
}

func cmdQuit(cmd *Command) {
	cmd.Gomuks.Stop()
}

func cmdClearCache(cmd *Command) {
	cmd.Config.Clear()
	cmd.Gomuks.Stop()
}

func cmdUnknownCommand(cmd *Command) {
	cmd.Reply("Unknown command \"%s\". Try \"/help\" for help.", cmd.Command)
}

func cmdHelp(cmd *Command) {
	cmd.Reply(`/help - Show the temporary help message.

/quit       - Quit gomuks.
/clearcache - Clear cache and quit gomuks.
/logout     - Log out of Matrix.

/me <message>      - Send an emote message.
/rainbow <message> - Send a rainbow message (markdown not supported).

/join <room address> - Join a room.
/leave               - Leave the current room.

/invite <user id>          - Invite a user.
/kick   <user id> [reason] - Kick a user.
/ban    <user id> [reason] - Ban a user.
/unban  <user id>          - Unban a user.

/send     <room id> <type>         <json> - Send a custom event to the given room.
/msend              <type>         <json> - Send a custom event to the current room.
/setstate <room id> <type> <key/-> <json> - Send a custom event to the given room.
/msetstate          <type> <key/-> <json> - Send a custom event to the current room.

/toggle <thing> - Temporary command to toggle various UI features.`)
}

func cmdLeave(cmd *Command) {
	err := cmd.Matrix.LeaveRoom(cmd.Room.MxRoom().ID)
	debug.Print("Leave room error:", err)
	if err == nil {
		cmd.MainView.RemoveRoom(cmd.Room.MxRoom())
	}
}

func cmdInvite(cmd *Command) {
	if len(cmd.Args) != 1 {
		cmd.Reply("Usage: /invite <user id>")
		return
	}
	_, err := cmd.Matrix.Client().InviteUser(cmd.Room.MxRoom().ID, &mautrix.ReqInviteUser{cmd.Args[0]})
	if err != nil {
		debug.Print("Error in invite call:", err)
		cmd.Reply("Failed to invite user:", err)
	}
}

func cmdBan(cmd *Command) {
	if len(cmd.Args) < 1 {
		cmd.Reply("Usage: /ban <user> [reason]")
		return
	}
	reason := "you are the weakest link, goodbye!"
	if len(cmd.Args) >= 2 {
		reason = strings.Join(cmd.Args[1:], " ")
	}
	_, err := cmd.Matrix.Client().BanUser(cmd.Room.MxRoom().ID, &mautrix.ReqBanUser{reason, cmd.Args[0]})
	if err != nil {
		debug.Print("Error in ban call:", err)
		cmd.Reply("Failed to ban user:", err)
	}

}

func cmdUnban(cmd *Command) {
	if len(cmd.Args) != 1 {
		cmd.Reply("Usage: /unban <user>")
		return
	}
	_, err := cmd.Matrix.Client().UnbanUser(cmd.Room.MxRoom().ID, &mautrix.ReqUnbanUser{cmd.Args[0]})
	if err != nil {
		debug.Print("Error in unban call:", err)
		cmd.Reply("Failed to unban user:", err)
	}
}

func cmdKick(cmd *Command) {
	if len(cmd.Args) < 1 {
		cmd.Reply("Usage: /kick <user> [reason]")
		return
	}
	reason := "you are the weakest link, goodbye!"
	if len(cmd.Args) >= 2 {
		reason = strings.Join(cmd.Args[1:], " ")
	}
	_, err := cmd.Matrix.Client().KickUser(cmd.Room.MxRoom().ID, &mautrix.ReqKickUser{reason, cmd.Args[0]})
	if err != nil {
		debug.Print("Error in kick call:", err)
		debug.Print("Failed to kick user:", err)
	}

}

func cmdJoin(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /join <room>")
		return
	}
	identifer := cmd.Args[0]
	server := ""
	if len(cmd.Args) > 1 {
		server = cmd.Args[1]
	}
	room, err := cmd.Matrix.JoinRoom(identifer, server)
	debug.Print("Join room error:", err)
	if err == nil {
		cmd.MainView.AddRoom(room)
	}
}

func cmdMSendEvent(cmd *Command) {
	if len(cmd.Args) < 2 {
		cmd.Reply("Usage: /msend <event type> <content>")
		return
	}
	cmd.Args = append([]string{cmd.Room.MxRoom().ID}, cmd.Args...)
	cmdSendEvent(cmd)
}

func cmdSendEvent(cmd *Command) {
	debug.Print(cmd.Command, cmd.Args, len(cmd.Args))
	if len(cmd.Args) < 3 {
		cmd.Reply("Usage: /send <room id> <event type> <content>")
		return
	}
	roomID := cmd.Args[0]
	eventType := mautrix.NewEventType(cmd.Args[1])
	rawContent := strings.Join(cmd.Args[2:], " ")
	debug.Print(roomID, eventType, rawContent)

	var content interface{}
	err := json.Unmarshal([]byte(rawContent), &content)
	debug.Print(err)
	if err != nil {
		cmd.Reply("Failed to parse content: %v", err)
		return
	}
	debug.Print("Sending event to", roomID, eventType, content)

	resp, err := cmd.Matrix.Client().SendMessageEvent(roomID, eventType, content)
	debug.Print(resp, err)
	if err != nil {
		cmd.Reply("Error from server: %v", err)
	} else {
		cmd.Reply("Event sent, ID: %s", resp.EventID)
	}
}

func cmdMSetState(cmd *Command) {
	if len(cmd.Args) < 2 {
		cmd.Reply("Usage: /msetstate <event type> <state key> <content>")
		return
	}
	cmd.Args = append([]string{cmd.Room.MxRoom().ID}, cmd.Args...)
	cmdSetState(cmd)
}

func cmdSetState(cmd *Command) {
	if len(cmd.Args) < 4 {
		cmd.Reply("Usage: /setstate <room id> <event type> <state key/`-`> <content>")
		return
	}

	roomID := cmd.Args[0]
	eventType := mautrix.NewEventType(cmd.Args[1])
	stateKey := cmd.Args[2]
	if stateKey == "-" {
		stateKey = ""
	}
	rawContent := strings.Join(cmd.Args[3:], " ")

	var content interface{}
	err := json.Unmarshal([]byte(rawContent), &content)
	if err != nil {
		cmd.Reply("Failed to parse content: %v", err)
		return
	}
	debug.Print("Sending state event to", roomID, eventType, stateKey, content)
	resp, err := cmd.Matrix.Client().SendStateEvent(roomID, eventType, stateKey, content)
	if err != nil {
		cmd.Reply("Error from server: %v", err)
	} else {
		cmd.Reply("State event sent, ID: %s", resp.EventID)
	}
}

func cmdToggle(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /toggle <rooms/users/baremessages/images/typingnotif/emojis>")
		return
	}
	switch cmd.Args[0] {
	case "rooms":
		cmd.Config.Preferences.HideRoomList = !cmd.Config.Preferences.HideRoomList
	case "users":
		cmd.Config.Preferences.HideUserList = !cmd.Config.Preferences.HideUserList
	case "baremessages":
		cmd.Config.Preferences.BareMessageView = !cmd.Config.Preferences.BareMessageView
	case "images":
		cmd.Config.Preferences.DisableImages = !cmd.Config.Preferences.DisableImages
	case "typingnotif":
		cmd.Config.Preferences.DisableTypingNotifs = !cmd.Config.Preferences.DisableTypingNotifs
	case "emojis":
		cmd.Config.Preferences.DisableEmojis = !cmd.Config.Preferences.DisableEmojis
	default:
		cmd.Reply("Usage: /toggle <rooms/users/baremessages/images/typingnotif/emojis>")
		return
	}
	// is there a reason this is called twice?
	// cmd.UI.Render()
	cmd.UI.Render()
	go cmd.Matrix.SendPreferencesToMatrix()
}

func cmdLogout(cmd *Command) {
	cmd.Matrix.Logout()
}
