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
	"fmt"
	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	"maunium.net/go/gomuks/interface"
	"strings"
)

type gomuksPointerContainer struct {
	MainView *MainView
	UI       *GomuksUI
	Matrix   ifc.MatrixContainer
	Config   *config.Config
	Gomuks   ifc.Gomuks
}

type Command struct {
	gomuksPointerContainer
	Handler *CommandProcessor

	Room        *RoomView
	Command     string
	OrigCommand string
	Args        []string
	OrigText    string
}

func (cmd *Command) Reply(message string, args ...interface{}) {
	cmd.Room.AddServiceMessage(fmt.Sprintf(message, args...))
}

type Alias struct {
	NewCommand string
}

func (alias *Alias) Process(cmd *Command) *Command {
	cmd.Command = alias.NewCommand
	return cmd
}

type CommandHandler func(cmd *Command)

type CommandProcessor struct {
	gomuksPointerContainer

	aliases  map[string]*Alias
	commands map[string]CommandHandler
}

func NewCommandProcessor(parent *MainView) *CommandProcessor {
	return &CommandProcessor{
		gomuksPointerContainer: gomuksPointerContainer{
			MainView: parent,
			UI:       parent.parent,
			Matrix:   parent.matrix,
			Config:   parent.config,
			Gomuks:   parent.gmx,
		},
		aliases: map[string]*Alias{
			"part":  {"leave"},
			"send":  {"sendevent"},
			"state": {"setstate"},
			"rb":    {"rainbow"},
		},
		commands: map[string]CommandHandler{
			"unknown-command": cmdUnknownCommand,
			"help":            cmdHelp,
			"me":              cmdMe,
			"quit":            cmdQuit,
			"clearcache":      cmdClearCache,
			"leave":           cmdLeave,
			"join":            cmdJoin,
			"uitoggle":        cmdUIToggle,
			"logout":          cmdLogout,
			"sendevent":       cmdSendEvent,
			"setstate":        cmdSetState,
			"rainbow":         cmdRainbow,
		},
	}
}

func (ch *CommandProcessor) ParseCommand(roomView *RoomView, text string) *Command {
	if text[0] != '/' || len(text) < 2 {
		return nil
	}
	text = text[1:]
	split := strings.SplitN(text, " ", -1)
	return &Command{
		gomuksPointerContainer: ch.gomuksPointerContainer,
		Handler:                ch,

		Room:        roomView,
		Command:     strings.ToLower(split[0]),
		OrigCommand: split[0],
		Args:        split[1:],
		OrigText:    text,
	}
}

func (ch *CommandProcessor) HandleCommand(cmd *Command) {
	defer debug.Recover()
	if cmd == nil {
		return
	}
	if alias, ok := ch.aliases[cmd.Command]; ok {
		cmd = alias.Process(cmd)
	}
	if cmd == nil {
		return
	}
	if handler, ok := ch.commands[cmd.Command]; ok {
		handler(cmd)
		return
	}
	cmdUnknownCommand(cmd)
}
