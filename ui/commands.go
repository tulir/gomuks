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

package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	dbg "runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/yuin/goldmark"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/lib/filepicker"
)

func cmdMe(cmd *Command) {
	text := strings.Join(cmd.Args, " ")
	go cmd.Room.SendMessage(event.MsgEmote, text)
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

var rainbowMark = goldmark.New(format.Extensions, format.HTMLOptions, goldmark.WithExtensions(ExtensionRainbow))

// TODO this command definitely belongs in a plugin once we have a plugin system.
func makeRainbow(cmd *Command, msgtype event.MessageType) {
	text := strings.Join(cmd.Args, " ")

	var buf strings.Builder
	_ = rainbowMark.Convert([]byte(text), &buf)

	htmlBody := strings.TrimRight(buf.String(), "\n")
	htmlBody = format.AntiParagraphRegex.ReplaceAllString(htmlBody, "$1")
	text = format.HTMLToText(htmlBody)

	count := strings.Count(htmlBody, defaultRB.ColorID)
	i := -1
	htmlBody = regexp.MustCompile(defaultRB.ColorID).ReplaceAllStringFunc(htmlBody, func(match string) string {
		i++
		return rainbow.GetInterpolatedColorFor(float64(i) / float64(count)).Hex()
	})

	go cmd.Room.SendMessageHTML(msgtype, text, htmlBody)
}

func cmdRainbow(cmd *Command) {
	makeRainbow(cmd, event.MsgText)
}

func cmdRainbowMe(cmd *Command) {
	makeRainbow(cmd, event.MsgEmote)
}

func cmdRainbowNotice(cmd *Command) {
	makeRainbow(cmd, event.MsgNotice)
}

func cmdNotice(cmd *Command) {
	go cmd.Room.SendMessage(event.MsgNotice, strings.Join(cmd.Args, " "))
}

func cmdAccept(cmd *Command) {
	room := cmd.Room.MxRoom()
	if room.SessionMember.Membership != "invite" {
		cmd.Reply("/accept can only be used in rooms you're invited to")
		return
	}
	_, server, _ := room.SessionMember.Sender.Parse()
	_, err := cmd.Matrix.JoinRoom(room.ID, server)
	if err != nil {
		cmd.Reply("Failed to accept invite: %v", err)
	} else {
		cmd.Reply("Successfully accepted invite")
	}
	cmd.MainView.UpdateTags(room)
	go cmd.MainView.LoadHistory(room.ID)
}

func cmdReject(cmd *Command) {
	room := cmd.Room.MxRoom()
	if room.SessionMember.Membership != "invite" {
		cmd.Reply("/reject can only be used in rooms you're invited to")
		return
	}
	err := cmd.Matrix.LeaveRoom(room.ID)
	if err != nil {
		cmd.Reply("Failed to reject invite: %v", err)
	} else {
		cmd.Reply("Successfully rejected invite")
	}
	cmd.MainView.RemoveRoom(room)
}

func cmdID(cmd *Command) {
	cmd.Reply("The internal ID of this room is %s", cmd.Room.MxRoom().ID)
}

type SelectReason string

const (
	SelectReply    SelectReason = "reply to"
	SelectReact                 = "react to"
	SelectRedact                = "redact"
	SelectEdit                  = "edit"
	SelectDownload              = "download"
	SelectOpen                  = "open"
	SelectCopy                  = "copy"
)

func cmdReply(cmd *Command) {
	cmd.Room.StartSelecting(SelectReply, strings.Join(cmd.Args, " "))
}

func cmdEdit(cmd *Command) {
	cmd.Room.StartSelecting(SelectEdit, "")
}

func findEditorExecutable() (string, string, error) {
	if editor := os.Getenv("VISUAL"); len(editor) > 0 {
		if path, err := exec.LookPath(editor); err != nil {
			return "", "", fmt.Errorf("$VISUAL ('%s') not found in $PATH", editor)
		} else {
			return editor, path, nil
		}
	} else if editor = os.Getenv("EDITOR"); len(editor) > 0 {
		if path, err := exec.LookPath(editor); err != nil {
			return "", "", fmt.Errorf("$EDITOR ('%s') not found in $PATH", editor)
		} else {
			return editor, path, nil
		}
	} else if path, _ := exec.LookPath("nano"); len(path) > 0 {
		return "nano", path, nil
	} else if path, _ = exec.LookPath("vi"); len(path) > 0 {
		return "vi", path, nil
	} else {
		return "", "", fmt.Errorf("$VISUAL and $EDITOR not set, nano and vi not found in $PATH")
	}
}

func cmdExternalEditor(cmd *Command) {
	var file *os.File
	defer func() {
		if file != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}
	}()

	fileExtension := "md"
	if cmd.Config.Preferences.DisableMarkdown {
		if cmd.Config.Preferences.DisableHTML {
			fileExtension = "txt"
		} else {
			fileExtension = "html"
		}
	}

	if editorName, executablePath, err := findEditorExecutable(); err != nil {
		cmd.Reply("Couldn't find editor to use: %v", err)
		return
	} else if file, err = os.CreateTemp("", fmt.Sprintf("gomuks-draft-*.%s", fileExtension)); err != nil {
		cmd.Reply("Failed to create temp file: %v", err)
		return
	} else if _, err = file.WriteString(cmd.RawArgs); err != nil {
		cmd.Reply("Failed to write to temp file: %v", err)
	} else if err = file.Close(); err != nil {
		cmd.Reply("Failed to close temp file: %v", err)
	} else if err = cmd.UI.RunExternal(executablePath, file.Name()); err != nil {
		var exitErr *exec.ExitError
		if isExit := errors.As(err, &exitErr); isExit {
			cmd.Reply("%s exited with non-zero status %d", editorName, exitErr.ExitCode())
		} else {
			cmd.Reply("Failed to run %s: %v", editorName, err)
		}
	} else if data, err := os.ReadFile(file.Name()); err != nil {
		cmd.Reply("Failed to read temp file: %v", err)
	} else if len(bytes.TrimSpace(data)) > 0 {
		cmd.Room.InputSubmit(string(data))
	} else {
		cmd.Reply("Temp file was blank, sending cancelled")
		if cmd.Room.editing != nil {
			cmd.Room.SetEditing(nil)
		}
	}
}

func cmdRedact(cmd *Command) {
	cmd.Room.StartSelecting(SelectRedact, strings.Join(cmd.Args, " "))
}

func cmdDownload(cmd *Command) {
	cmd.Room.StartSelecting(SelectDownload, strings.Join(cmd.Args, " "))
}

func cmdUpload(cmd *Command) {
	var path string
	var err error
	if len(cmd.Args) == 0 {
		if filepicker.IsSupported() {
			path, err = filepicker.Open()
			if err != nil {
				cmd.Reply("Failed to open file picker: %v", err)
				return
			} else if len(path) == 0 {
				cmd.Reply("File picking cancelled")
				return
			}
		} else {
			cmd.Reply("Usage: /upload <file>")
			return
		}
	} else {
		path, err = filepath.Abs(cmd.RawArgs)
		if err != nil {
			cmd.Reply("Failed to get absolute path: %v", err)
			return
		}
	}

	go cmd.Room.SendMessageMedia(path)
}

func cmdOpen(cmd *Command) {
	cmd.Room.StartSelecting(SelectOpen, strings.Join(cmd.Args, " "))
}

func cmdCopy(cmd *Command) {
	register := strings.Join(cmd.Args, " ")
	if len(register) == 0 {
		register = "clipboard"
	}
	if register == "clipboard" || register == "primary" {
		cmd.Room.StartSelecting(SelectCopy, register)
	} else {
		cmd.Reply("Usage: /copy [register], where register is either \"clipboard\" or \"primary\". Defaults to \"clipboard\".")
	}
}

func cmdReact(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /react <reaction>")
		return
	}

	cmd.Room.StartSelecting(SelectReact, strings.Join(cmd.Args, " "))
}

func readRoomAlias(cmd *Command) (alias id.RoomAlias, err error) {
	param := strings.Join(cmd.Args[1:], " ")
	if strings.ContainsRune(param, ':') {
		if param[0] != '#' {
			return "", errors.New("full aliases must start with #")
		}

		alias = id.RoomAlias(param)
	} else {
		_, homeserver, _ := cmd.Matrix.Client().UserID.Parse()
		alias = id.NewRoomAlias(param, homeserver)
	}
	return
}

func cmdAlias(cmd *Command) {
	if len(cmd.Args) < 2 {
		cmd.Reply("Usage: /alias <add|remove> <localpart>")
		return
	}

	alias, err := readRoomAlias(cmd)
	if err != nil {
		cmd.Reply(err.Error())
		return
	}

	subcmd := strings.ToLower(cmd.Args[0])
	switch subcmd {
	case "add", "create":
		cmdAddAlias(cmd, alias)
	case "remove", "delete", "del", "rm":
		cmdRemoveAlias(cmd, alias)
	case "resolve", "get":
		cmdResolveAlias(cmd, alias)
	default:
		cmd.Reply("Usage: /alias <add|remove|resolve> <localpart>")
	}
}

func niceError(err error) string {
	httpErr, ok := err.(mautrix.HTTPError)
	if ok && httpErr.RespError != nil {
		return httpErr.RespError.Error()
	}
	return err.Error()
}

func cmdAddAlias(cmd *Command, alias id.RoomAlias) {
	_, err := cmd.Matrix.Client().CreateAlias(alias, cmd.Room.MxRoom().ID)
	if err != nil {
		cmd.Reply("Failed to create alias: %v", niceError(err))
	} else {
		cmd.Reply("Created alias %s", alias)
	}
}

func cmdRemoveAlias(cmd *Command, alias id.RoomAlias) {
	_, err := cmd.Matrix.Client().DeleteAlias(alias)
	if err != nil {
		cmd.Reply("Failed to delete alias: %v", niceError(err))
	} else {
		cmd.Reply("Deleted alias %s", alias)
	}
}

func cmdResolveAlias(cmd *Command, alias id.RoomAlias) {
	resp, err := cmd.Matrix.Client().ResolveAlias(alias)
	if err != nil {
		cmd.Reply("Failed to resolve alias: %v", niceError(err))
	} else {
		roomIDText := string(resp.RoomID)
		if resp.RoomID == cmd.Room.MxRoom().ID {
			roomIDText += " (this room)"
		}
		cmd.Reply("Alias %s points to room %s\nThere are %d servers in the room.", alias, roomIDText, len(resp.Servers))
	}
}

func cmdTags(cmd *Command) {
	tags := cmd.Room.MxRoom().RawTags
	if len(cmd.Args) > 0 && cmd.Args[0] == "--internal" {
		tags = cmd.Room.MxRoom().Tags()
	}
	if len(tags) == 0 {
		if cmd.Room.MxRoom().IsDirect {
			cmd.Reply("This room has no tags, but it's marked as a direct chat.")
		} else {
			cmd.Reply("This room has no tags.")
		}
		return
	}
	var resp strings.Builder
	resp.WriteString("Tags in this room:\n")
	for _, tag := range tags {
		if tag.Order != "" {
			_, _ = fmt.Fprintf(&resp, "%s (order: %s)\n", tag.Tag, tag.Order)
		} else {
			_, _ = fmt.Fprintf(&resp, "%s (no order)\n", tag.Tag)
		}
	}
	cmd.Reply(strings.TrimSpace(resp.String()))
}

func cmdTag(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /tag <tag> [order]")
		return
	}
	order := math.NaN()
	if len(cmd.Args) > 1 {
		var err error
		order, err = strconv.ParseFloat(cmd.Args[1], 64)
		if err != nil {
			cmd.Reply("%s is not a valid order: %v", cmd.Args[1], err)
			return
		}
	}
	var err error
	if len(cmd.Args) > 2 && cmd.Args[2] == "--reset" {
		tags := event.Tags{
			cmd.Args[0]: {Order: json.Number(fmt.Sprintf("%f", order))},
		}
		for _, tag := range cmd.Room.MxRoom().RawTags {
			tags[tag.Tag] = event.Tag{Order: tag.Order}
		}
		err = cmd.Matrix.Client().SetTags(cmd.Room.MxRoom().ID, tags)
	} else {
		err = cmd.Matrix.Client().AddTag(cmd.Room.MxRoom().ID, cmd.Args[0], order)
	}
	if err != nil {
		cmd.Reply("Failed to add tag: %v", err)
	}
}

func cmdUntag(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /untag <tag>")
		return
	}
	err := cmd.Matrix.Client().RemoveTag(cmd.Room.MxRoom().ID, cmd.Args[0])
	if err != nil {
		cmd.Reply("Failed to remove tag: %v", err)
	}
}

func cmdRoomNick(cmd *Command) {
	room := cmd.Room.MxRoom()
	member := room.GetMember(room.SessionUserID)
	member.Displayname = strings.Join(cmd.Args, " ")
	_, err := cmd.Matrix.Client().SendStateEvent(room.ID, event.StateMember, string(room.SessionUserID), member)
	if err != nil {
		cmd.Reply("Failed to set room nick: %v", err)
	}
}

func cmdFingerprint(cmd *Command) {
	c := cmd.Matrix.Crypto()
	if c == nil {
		cmd.Reply("Encryption support is not enabled")
	} else {
		cmd.Reply("Device ID: %s\nFingerprint: %s", cmd.Matrix.Client().DeviceID, c.Fingerprint())
	}
}

func cmdHeapProfile(cmd *Command) {
	if len(cmd.Args) == 0 || cmd.Args[0] != "nogc" {
		runtime.GC()
		dbg.FreeOSMemory()
	}
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
	cmd.Reply(`Unknown command "/%s". Try "/help" for help.`, cmd.Command)
}

func cmdHelp(cmd *Command) {
	view := cmd.MainView
	view.ShowModal(NewHelpModal(view))
}

func cmdKeys(cmd *Command) {
	view := cmd.MainView
	view.ShowModal(NewKeysModal(view))
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
	_, err := cmd.Matrix.Client().InviteUser(cmd.Room.MxRoom().ID, &mautrix.ReqInviteUser{UserID: id.UserID(cmd.Args[0])})
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
	_, err := cmd.Matrix.Client().BanUser(cmd.Room.MxRoom().ID, &mautrix.ReqBanUser{Reason: reason, UserID: id.UserID(cmd.Args[0])})
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
	_, err := cmd.Matrix.Client().UnbanUser(cmd.Room.MxRoom().ID, &mautrix.ReqUnbanUser{UserID: id.UserID(cmd.Args[0])})
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
	_, err := cmd.Matrix.Client().KickUser(cmd.Room.MxRoom().ID, &mautrix.ReqKickUser{Reason: reason, UserID: id.UserID(cmd.Args[0])})
	if err != nil {
		debug.Print("Error in kick call:", err)
		debug.Print("Failed to kick user:", err)
	}
}

func formatPowerLevels(pl *event.PowerLevelsEventContent) string {
	var buf strings.Builder
	buf.WriteString("Membership actions:\n")
	_, _ = fmt.Fprintf(&buf, "  Invite: %d\n", pl.Invite())
	_, _ = fmt.Fprintf(&buf, "  Kick: %d\n", pl.Kick())
	_, _ = fmt.Fprintf(&buf, "  Ban: %d\n", pl.Ban())
	buf.WriteString("Events:\n")
	_, _ = fmt.Fprintf(&buf, "  Redact: %d\n", pl.Redact())
	_, _ = fmt.Fprintf(&buf, "  State default: %d\n", pl.StateDefault())
	_, _ = fmt.Fprintf(&buf, "  Event default: %d\n", pl.EventsDefault)
	for evtType, level := range pl.Events {
		_, _ = fmt.Fprintf(&buf, "  %s: %d\n", evtType, level)
	}
	buf.WriteString("Users:\n")
	_, _ = fmt.Fprintf(&buf, "  Default: %d\n", pl.UsersDefault)
	for userID, level := range pl.Users {
		_, _ = fmt.Fprintf(&buf, "  %s: %d\n", userID, level)
	}
	return strings.TrimSpace(buf.String())
}

func copyPtr(ptr *int) *int {
	if ptr == nil {
		return nil
	}
	val := *ptr
	return &val
}

func copyMap[Key comparable](m map[Key]int) map[Key]int {
	if m == nil {
		return nil
	}
	copied := make(map[Key]int, len(m))
	for k, v := range m {
		copied[k] = v
	}
	return copied
}

func copyPowerLevels(pl *event.PowerLevelsEventContent) *event.PowerLevelsEventContent {
	return &event.PowerLevelsEventContent{
		Users:           copyMap(pl.Users),
		Events:          copyMap(pl.Events),
		InvitePtr:       copyPtr(pl.InvitePtr),
		KickPtr:         copyPtr(pl.KickPtr),
		BanPtr:          copyPtr(pl.BanPtr),
		RedactPtr:       copyPtr(pl.RedactPtr),
		StateDefaultPtr: copyPtr(pl.StateDefaultPtr),
		EventsDefault:   pl.EventsDefault,
		UsersDefault:    pl.UsersDefault,
	}
}

var things = `
[thing] can be one of the following

Literals:
* invite, kick, ban, redact - special moderation action levels
* state_default, events_default - default level for state and non-state events
* users_default - default level for users

Patterns:
* user ID - specific user level
* event type - specific event type level

The default levels are 0 for users, 50 for moderators and 100 for admins.`

func cmdPowerLevel(cmd *Command) {
	evt := cmd.Room.MxRoom().GetStateEvent(event.StatePowerLevels, "")
	pl := copyPowerLevels(evt.Content.AsPowerLevels())
	if len(cmd.Args) == 0 {
		// TODO open in modal?
		cmd.Reply(formatPowerLevels(pl))
		return
	} else if len(cmd.Args) < 2 {
		cmd.Reply("Usage: /%s [thing] [level]\n%s", cmd.Command, things)
		return
	}

	value, err := strconv.Atoi(cmd.Args[1])
	if err != nil {
		cmd.Reply("Invalid power level %q: %v", cmd.Args[1], err)
		return
	}

	ownLevel := pl.GetUserLevel(cmd.Matrix.Client().UserID)
	plChangeLevel := pl.GetEventLevel(event.StatePowerLevels)
	if ownLevel < plChangeLevel {
		cmd.Reply("Can't modify power levels (own level is %d, modifying requires %d)", ownLevel, plChangeLevel)
		return
	} else if value > ownLevel {
		cmd.Reply("Can't set level to be higher than own level (%d > %d)", value, ownLevel)
		return
	}

	var oldValue int
	var thing string
	switch cmd.Args[0] {
	case "invite":
		oldValue = pl.Invite()
		pl.InvitePtr = &value
		thing = "invite level"
	case "kick":
		oldValue = pl.Kick()
		pl.KickPtr = &value
		thing = "kick level"
	case "ban":
		oldValue = pl.Ban()
		pl.BanPtr = &value
		thing = "ban level"
	case "redact":
		oldValue = pl.Redact()
		pl.RedactPtr = &value
		thing = "level for redacting other users' events"
	case "state_default":
		oldValue = pl.StateDefault()
		pl.StateDefaultPtr = &value
		thing = "default level for state events"
	case "events_default":
		oldValue = pl.EventsDefault
		pl.EventsDefault = value
		thing = "default level for normal events"
	case "users_default":
		oldValue = pl.UsersDefault
		pl.UsersDefault = value
		thing = "default level for users"
	default:
		userID := id.UserID(cmd.Args[0])
		if _, _, err = userID.Parse(); err == nil {
			if pl.Users == nil {
				pl.Users = make(map[id.UserID]int)
			}
			oldValue = pl.Users[userID]
			if oldValue == ownLevel && userID != cmd.Matrix.Client().UserID {
				cmd.Reply("Can't change level of another user which is equal to own level (%d)", ownLevel)
				return
			}
			pl.Users[userID] = value
			thing = fmt.Sprintf("level of user %s", userID)
		} else {
			if pl.Events == nil {
				pl.Events = make(map[string]int)
			}
			oldValue = pl.Events[cmd.Args[0]]
			pl.Events[cmd.Args[0]] = value
			thing = fmt.Sprintf("level for event %s", cmd.Args[0])
		}
	}

	if oldValue == value {
		cmd.Reply("%s is already %d", strings.ToUpper(thing[0:1])+thing[1:], value)
	} else if oldValue > ownLevel {
		cmd.Reply("Can't change level which is higher than own level (%d > %d)", oldValue, ownLevel)
	} else if resp, err := cmd.Matrix.Client().SendStateEvent(cmd.Room.MxRoom().ID, event.StatePowerLevels, "", pl); err != nil {
		if httpErr, ok := err.(mautrix.HTTPError); ok && httpErr.RespError != nil {
			err = httpErr.RespError
		}
		cmd.Reply("Failed to set %s to %d: %v", thing, value, err)
	} else {
		cmd.Reply("Successfully set %s to %d\n(event ID: %s)", thing, value, resp.EventID)
	}
}

func cmdCreateRoom(cmd *Command) {
	req := &mautrix.ReqCreateRoom{}
	if len(cmd.Args) > 0 {
		req.Name = strings.Join(cmd.Args, " ")
	}
	room, err := cmd.Matrix.CreateRoom(req)
	if err != nil {
		cmd.Reply("Failed to create room: %v", err)
		return
	}
	cmd.MainView.SwitchRoom("", room)
}

func cmdPrivateMessage(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /pm <user id> [more user ids...]")
	}
	invites := make([]id.UserID, len(cmd.Args))
	for i, userID := range cmd.Args {
		invites[i] = id.UserID(userID)
		_, _, err := invites[i].Parse()
		if err != nil {
			cmd.Reply("%s isn't a valid user ID", userID)
			return
		}
	}
	req := &mautrix.ReqCreateRoom{
		Preset:   "trusted_private_chat",
		Invite:   invites,
		IsDirect: true,
	}
	room, err := cmd.Matrix.CreateRoom(req)
	if err != nil {
		cmd.Reply("Failed to create room: %v", err)
		return
	}
	cmd.MainView.SwitchRoom("", room)
}

func cmdJoin(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /join <room>")
		return
	}
	identifer := id.RoomID(cmd.Args[0])
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
	cmd.Args = append([]string{string(cmd.Room.MxRoom().ID)}, cmd.Args...)
	cmdSendEvent(cmd)
}

func cmdSendEvent(cmd *Command) {
	if len(cmd.Args) < 3 {
		cmd.Reply("Usage: /send <room id> <event type> <content>")
		return
	}
	roomID := id.RoomID(cmd.Args[0])
	eventType := event.NewEventType(cmd.Args[1])
	rawContent := strings.Join(cmd.Args[2:], " ")

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
	cmd.Args = append([]string{string(cmd.Room.MxRoom().ID)}, cmd.Args...)
	cmdSetState(cmd)
}

func cmdSetState(cmd *Command) {
	if len(cmd.Args) < 4 {
		cmd.Reply("Usage: /setstate <room id> <event type> <state key/`-`> <content>")
		return
	}

	roomID := id.RoomID(cmd.Args[0])
	eventType := event.NewEventType(cmd.Args[1])
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

type ToggleMessage interface {
	Name() string
	Format(state bool) string
}

type HideMessage string

func (hm HideMessage) Format(state bool) string {
	if state {
		return string(hm) + " is now hidden"
	} else {
		return string(hm) + " is now visible"
	}
}

func (hm HideMessage) Name() string {
	return string(hm)
}

type SimpleToggleMessage string

func (stm SimpleToggleMessage) Format(state bool) string {
	if state {
		return "Disabled " + string(stm)
	} else {
		return "Enabled " + string(stm)
	}
}

func (stm SimpleToggleMessage) Name() string {
	return string(unicode.ToUpper(rune(stm[0]))) + string(stm[1:])
}

type InvertedToggleMessage string

func (itm InvertedToggleMessage) Format(state bool) string {
	if state {
		return "Enabled " + string(itm)
	} else {
		return "Disabled " + string(itm)
	}
}

func (itm InvertedToggleMessage) Name() string {
	return string(unicode.ToUpper(rune(itm[0]))) + string(itm[1:])
}

var toggleMsg = map[string]ToggleMessage{
	"rooms":         HideMessage("Room list sidebar"),
	"users":         HideMessage("User list sidebar"),
	"timestamps":    HideMessage("message timestamps"),
	"baremessages":  InvertedToggleMessage("bare message view"),
	"images":        SimpleToggleMessage("image rendering"),
	"typingnotif":   SimpleToggleMessage("typing notifications"),
	"emojis":        SimpleToggleMessage("emoji shortcode conversion"),
	"html":          SimpleToggleMessage("HTML input"),
	"markdown":      SimpleToggleMessage("markdown input"),
	"downloads":     SimpleToggleMessage("automatic downloads"),
	"notifications": SimpleToggleMessage("desktop notifications"),
	"unverified":    SimpleToggleMessage("sending messages to unverified devices"),
	"showurls":      SimpleToggleMessage("show URLs in text format"),
	"inlineurls":    InvertedToggleMessage("use fancy terminal features to render URLs inside text"),
}

func makeUsage() string {
	var buf strings.Builder
	buf.WriteString("Usage: /toggle <things...>\n\n")
	buf.WriteString("List of Things:\n")
	for key, value := range toggleMsg {
		_, _ = fmt.Fprintf(&buf, "* %s - %s\n", key, value.Name())
	}
	return buf.String()[:buf.Len()-1]
}

func cmdToggle(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply(makeUsage())
		return
	}
	for _, thing := range cmd.Args {
		var val *bool
		switch thing {
		case "rooms":
			val = &cmd.Config.Preferences.HideRoomList
		case "users":
			val = &cmd.Config.Preferences.HideUserList
		case "timestamps":
			val = &cmd.Config.Preferences.HideTimestamp
		case "baremessages":
			val = &cmd.Config.Preferences.BareMessageView
		case "images":
			val = &cmd.Config.Preferences.DisableImages
		case "typingnotif":
			val = &cmd.Config.Preferences.DisableTypingNotifs
		case "emojis":
			val = &cmd.Config.Preferences.DisableEmojis
		case "html":
			val = &cmd.Config.Preferences.DisableHTML
		case "markdown":
			val = &cmd.Config.Preferences.DisableMarkdown
		case "downloads":
			val = &cmd.Config.Preferences.DisableDownloads
		case "notifications":
			val = &cmd.Config.Preferences.DisableNotifications
		case "unverified":
			val = &cmd.Config.SendToVerifiedOnly
		case "showurls":
			val = &cmd.Config.Preferences.DisableShowURLs
		case "inlineurls":
			switch cmd.Config.Preferences.InlineURLMode {
			case "enable":
				cmd.Config.Preferences.InlineURLMode = "disable"
				cmd.Reply("Force-disabled using fancy terminal features to render URLs inside text. Restart gomuks to apply changes.")
			default:
				cmd.Config.Preferences.InlineURLMode = "enable"
				cmd.Reply("Force-enabled using fancy terminal features to render URLs inside text. Restart gomuks to apply changes.")
			}
			continue
		default:
			cmd.Reply("Unknown toggle %s. Use /toggle without arguments for a list of togglable things.", thing)
			return
		}
		*val = !(*val)
		debug.Print(thing, *val)
		cmd.Reply(toggleMsg[thing].Format(*val))
		if thing == "rooms" {
			// Update topic string to include or not include room name
			cmd.Room.Update()
		}
	}
	cmd.UI.Render()
	go cmd.Matrix.SendPreferencesToMatrix()
}

func cmdLogout(cmd *Command) {
	cmd.Matrix.Logout()
}
