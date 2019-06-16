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
	"io"
	"os"
	"runtime"
	dbg "runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/lucasb-eyer/go-colorful"

	"maunium.net/go/mautrix"

	"maunium.net/go/gomuks/debug"
)

func cmdMe(cmd *Command) {
	text := strings.Join(cmd.Args, " ")
	go cmd.Room.SendMessage(mautrix.MsgEmote, text)
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
	{colorful.LinearRgb(1, 0, 0), 0 / 11.0},
	{colorful.LinearRgb(1, 0.5, 0), 1 / 11.0},
	{colorful.LinearRgb(1, 1, 0), 2 / 11.0},
	{colorful.LinearRgb(0.5, 1, 0), 3 / 11.0},
	{colorful.LinearRgb(0, 1, 0), 4 / 11.0},
	{colorful.LinearRgb(0, 1, 0.5), 5 / 11.0},
	{colorful.LinearRgb(0, 1, 1), 6 / 11.0},
	{colorful.LinearRgb(0, 0.5, 1), 7 / 11.0},
	{colorful.LinearRgb(0, 0, 1), 8 / 11.0},
	{colorful.LinearRgb(0.5, 0, 1), 9 / 11.0},
	{colorful.LinearRgb(1, 0, 1), 10 / 11.0},
	{colorful.LinearRgb(1, 0, 0.5), 11 / 11.0},
}

// TODO this command definitely belongs in a plugin once we have a plugin system.
func makeRainbow(cmd *Command, msgtype mautrix.MessageType) {
	text := strings.Join(cmd.Args, " ")
	var html strings.Builder
	for i, char := range text {
		if unicode.IsSpace(char) {
			html.WriteRune(char)
			continue
		}
		color := rainbow.GetInterpolatedColorFor(float64(i) / float64(len(text))).Hex()
		_, _ = fmt.Fprintf(&html, "<font color=\"%s\">%c</font>", color, char)
	}
	go cmd.Room.SendMessage(msgtype, html.String())
	cmd.UI.Render()
}

func cmdRainbow(cmd *Command) {
	makeRainbow(cmd, mautrix.MsgText)
}

func cmdRainbowMe(cmd *Command) {
	makeRainbow(cmd, mautrix.MsgEmote)
}

func cmdNotice(cmd *Command) {
	go cmd.Room.SendMessage(mautrix.MsgNotice, strings.Join(cmd.Args, " "))
	cmd.UI.Render()
}

func cmdRoomNick(cmd *Command) {
	room := cmd.Room.MxRoom()
	member := room.GetMember(room.SessionUserID)
	member.Displayname = strings.Join(cmd.Args, " ")
	_, err := cmd.Matrix.Client().SendStateEvent(room.ID, mautrix.StateMember, room.SessionUserID, member)
	if err != nil {
		cmd.Reply("Failed to set room nick:", err)
	}
}

func cmdHeapProfile(cmd *Command) {
	runtime.GC()
	dbg.FreeOSMemory()
	memProfile, err := os.Create("gomuks.heap.prof")
	if err != nil {
		debug.Print("Failed to open gomuks.heap.prof:", err)
		return
	}
	defer func() {
		err := memProfile.Close()
		if err != nil {
			debug.Print("Failed to close gomuks.heap.prof:", err)
		}
	}()
	if err := pprof.WriteHeapProfile(memProfile); err != nil {
		debug.Print("Heap profile error:", err)
	}
}

func runTimedProfile(cmd *Command, start func(writer io.Writer) error, stop func(), task, file string) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /%s <seconds>", cmd.Command)
	} else if dur, err := strconv.Atoi(cmd.Args[0]); err != nil || dur < 0 {
		cmd.Reply("Usage: /%s <seconds>", cmd.Command)
	} else if cpuProfile, err := os.Create(file); err != nil {
		debug.Printf("Failed to open %s: %v", file, err)
	} else if err = start(cpuProfile); err != nil {
		_ = cpuProfile.Close()
		debug.Print(task, "error:", err)
	} else {
		cmd.Reply("Started %s for %d seconds", task, dur)
		go func() {
			time.Sleep(time.Duration(dur) * time.Second)
			stop()
			cmd.Reply("%s finished.", task)

			err := cpuProfile.Close()
			if err != nil {
				debug.Print("Failed to close gomuks.cpu.prof:", err)
			}
		}()
	}
}

func cmdCPUProfile(cmd *Command) {
	runTimedProfile(cmd, pprof.StartCPUProfile, pprof.StopCPUProfile, "CPU profiling", "gomuks.cpu.prof")
}

func cmdTrace(cmd *Command) {
	runTimedProfile(cmd, trace.Start, trace.Stop, "Call tracing", "gomuks.trace")
}

func cmdQuit(cmd *Command) {
	cmd.Gomuks.Stop(true)
}

func cmdClearCache(cmd *Command) {
	cmd.Config.Clear()
	cmd.Gomuks.Stop(false)
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

/create [room name]  - Create a room.
/pm <user id> <...>  - Create a private chat with the given user(s).
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
	_, err := cmd.Matrix.Client().InviteUser(cmd.Room.MxRoom().ID, &mautrix.ReqInviteUser{UserID: cmd.Args[0]})
	if err != nil {
		debug.Print("Error in invite call:", err)
		cmd.Reply("Failed to invite user: %v", err)
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
	_, err := cmd.Matrix.Client().BanUser(cmd.Room.MxRoom().ID, &mautrix.ReqBanUser{Reason: reason, UserID: cmd.Args[0]})
	if err != nil {
		debug.Print("Error in ban call:", err)
		cmd.Reply("Failed to ban user: %v", err)
	}

}

func cmdUnban(cmd *Command) {
	if len(cmd.Args) != 1 {
		cmd.Reply("Usage: /unban <user>")
		return
	}
	_, err := cmd.Matrix.Client().UnbanUser(cmd.Room.MxRoom().ID, &mautrix.ReqUnbanUser{UserID: cmd.Args[0]})
	if err != nil {
		debug.Print("Error in unban call:", err)
		cmd.Reply("Failed to unban user: %v", err)
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
	_, err := cmd.Matrix.Client().KickUser(cmd.Room.MxRoom().ID, &mautrix.ReqKickUser{Reason: reason, UserID: cmd.Args[0]})
	if err != nil {
		debug.Print("Error in kick call:", err)
		debug.Print("Failed to kick user:", err)
	}
}

func cmdCreateRoom(cmd *Command) {
	req := &mautrix.ReqCreateRoom{}
	if len(cmd.Args) > 0 {
		req.Name = strings.Join(cmd.Args, " ")
	}
	room, err := cmd.Matrix.CreateRoom(req)
	if err != nil {
		cmd.Reply("Failed to create room:", err)
		return
	}
	cmd.MainView.SwitchRoom("", room)
}

func cmdPrivateMessage(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /pm <user id> [more user ids...]")
	}
	req := &mautrix.ReqCreateRoom{
		Preset: "trusted_private_chat",
		Invite: cmd.Args,
	}
	room, err := cmd.Matrix.CreateRoom(req)
	if err != nil {
		cmd.Reply("Failed to create room:", err)
		return
	}
	cmd.MainView.SwitchRoom("", room)
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
