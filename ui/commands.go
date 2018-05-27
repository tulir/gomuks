// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2018 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ui

import (
	"maunium.net/go/gomuks/debug"
	"strings"
)

func cmdMe(cmd *Command) {
	text := strings.Join(cmd.Args, " ")
	tempMessage := cmd.Room.NewTempMessage("m.emote", text)
	go cmd.MainView.sendTempMessage(cmd.Room, tempMessage, text)
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
	cmd.Reply("Known command. Don't try \"/help\" for help.")
}

func cmdLeave(cmd *Command) {
	err := cmd.Matrix.LeaveRoom(cmd.Room.MxRoom().ID)
	debug.Print("Leave room error:", err)
	if err == nil {
		cmd.MainView.RemoveRoom(cmd.Room.MxRoom())
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

func cmdUIToggle(cmd *Command) {
	if len(cmd.Args) == 0 {
		cmd.Reply("Usage: /uitoggle <rooms/users/baremessages>")
		return
	}
	switch cmd.Args[0] {
	case "rooms":
		cmd.MainView.hideRoomList = !cmd.MainView.hideRoomList
		cmd.Config.Preferences.HideRoomList = cmd.MainView.hideRoomList
	case "users":
		cmd.MainView.hideUserList = !cmd.MainView.hideUserList
		cmd.Config.Preferences.HideUserList = cmd.MainView.hideUserList
	case "baremessages":
		cmd.MainView.bareMessages = !cmd.MainView.bareMessages
		cmd.Config.Preferences.BareMessageView = cmd.MainView.bareMessages
	default:
		cmd.Reply("Usage: /uitoggle <rooms/users/baremessages>")
		return
	}
	cmd.UI.Render()
	cmd.UI.Render()
	go cmd.Matrix.SendPreferencesToMatrix()
}

func cmdLogout(cmd *Command) {
	cmd.Matrix.Logout()
}
