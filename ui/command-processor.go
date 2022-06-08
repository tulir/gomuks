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
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"maunium.net/go/gomuks/config"
	"maunium.net/go/gomuks/debug"
	ifc "maunium.net/go/gomuks/interface"
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
	RawArgs     string
	OrigText    string
}

type CommandAutocomplete Command

func (cmd *Command) Reply(message string, args ...interface{}) {
	cmd.Room.AddServiceMessage(fmt.Sprintf(message, args...))
	cmd.UI.Render()
}

type Alias struct {
	NewCommand string
}

func (alias *Alias) Process(cmd *Command) *Command {
	cmd.Command = alias.NewCommand
	return cmd
}

type CommandHandler func(cmd *Command)
type CommandAutocompleter func(cmd *CommandAutocomplete) (completions []string, newText string)

type CommandProcessor struct {
	gomuksPointerContainer

	aliases  map[string]*Alias
	commands map[string]CommandHandler

	autocompleters map[string]CommandAutocompleter
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
			"part":       {"leave"},
			"send":       {"sendevent"},
			"msend":      {"msendevent"},
			"state":      {"setstate"},
			"mstate":     {"msetstate"},
			"rb":         {"rainbow"},
			"rbme":       {"rainbowme"},
			"rbn":        {"rainbownotice"},
			"myroomnick": {"roomnick"},
			"createroom": {"create"},
			"dm":         {"pm"},
			"query":      {"pm"},
			"r":          {"reply"},
			"delete":     {"redact"},
			"remove":     {"redact"},
			"rm":         {"redact"},
			"del":        {"redact"},
			"e":          {"edit"},
			"dl":         {"download"},
			"o":          {"open"},
			"4s":         {"ssss"},
			"s4":         {"ssss"},
			"cs":         {"cross-signing"},
		},
		autocompleters: map[string]CommandAutocompleter{
			"devices":       autocompleteUser,
			"device":        autocompleteDevice,
			"verify":        autocompleteUser,
			"verify-device": autocompleteDevice,
			"unverify":      autocompleteDevice,
			"blacklist":     autocompleteDevice,
			"upload":        autocompleteFile,
			"download":      autocompleteFile,
			"open":          autocompleteFile,
			"import":        autocompleteFile,
			"export":        autocompleteFile,
			"export-room":   autocompleteFile,
			"toggle":        autocompleteToggle,
		},
		commands: map[string]CommandHandler{
			"unknown-command": cmdUnknownCommand,

			"id":         cmdID,
			"help":       cmdHelp,
			"keys":       cmdKeys,
			"me":         cmdMe,
			"quit":       cmdQuit,
			"clearcache": cmdClearCache,
			"leave":      cmdLeave,
			"create":     cmdCreateRoom,
			"pm":         cmdPrivateMessage,
			"join":       cmdJoin,
			"kick":       cmdKick,
			"ban":        cmdBan,
			"unban":      cmdUnban,
			"toggle":     cmdToggle,
			"logout":     cmdLogout,
			"accept":     cmdAccept,
			"reject":     cmdReject,
			"reply":      cmdReply,
			"redact":     cmdRedact,
			"react":      cmdReact,
			"edit":       cmdEdit,
			"external":   cmdExternalEditor,
			"download":   cmdDownload,
			"upload":     cmdUpload,
			"open":       cmdOpen,
			"copy":       cmdCopy,
			"sendevent":  cmdSendEvent,
			"msendevent": cmdMSendEvent,
			"setstate":   cmdSetState,
			"msetstate":  cmdMSetState,
			"roomnick":   cmdRoomNick,
			"rainbow":    cmdRainbow,
			"rainbowme":  cmdRainbowMe,
			"notice":     cmdNotice,
			"alias":      cmdAlias,
			"tags":       cmdTags,
			"tag":        cmdTag,
			"untag":      cmdUntag,
			"invite":     cmdInvite,
			"hprof":      cmdHeapProfile,
			"cprof":      cmdCPUProfile,
			"trace":      cmdTrace,
			"panic": func(cmd *Command) {
				panic("hello world")
			},

			"rainbownotice": cmdRainbowNotice,

			"fingerprint":   cmdFingerprint,
			"devices":       cmdDevices,
			"verify-device": cmdVerifyDevice,
			"verify":        cmdVerify,
			"device":        cmdDevice,
			"unverify":      cmdUnverify,
			"blacklist":     cmdBlacklist,
			"reset-session": cmdResetSession,
			"import":        cmdImportKeys,
			"export":        cmdExportKeys,
			"export-room":   cmdExportRoomKeys,
			"ssss":          cmdSSSS,
			"cross-signing": cmdCrossSigning,
		},
	}
}

func (ch *CommandProcessor) ParseCommand(roomView *RoomView, text string) *Command {
	if text[0] != '/' || len(text) < 2 {
		return nil
	}
	text = text[1:]
	split := strings.Fields(text)
	command := split[0]
	args := split[1:]
	var rawArgs string
	if len(text) > len(command)+1 {
		rawArgs = text[len(command)+1:]
	}
	return &Command{
		gomuksPointerContainer: ch.gomuksPointerContainer,
		Handler:                ch,

		Room:        roomView,
		Command:     strings.ToLower(command),
		OrigCommand: command,
		Args:        args,
		RawArgs:     rawArgs,
		OrigText:    text,
	}
}

func (ch *CommandProcessor) Autocomplete(roomView *RoomView, text string, cursorOffset int) ([]string, string, bool) {
	var completions []string
	if cursorOffset != runewidth.StringWidth(text) {
		return completions, text, false
	}

	var cmd *Command
	if cmd = ch.ParseCommand(roomView, text); cmd == nil {
		return completions, text, false
	} else if alias, ok := ch.aliases[cmd.Command]; ok {
		cmd = alias.Process(cmd)
	}

	handler, ok := ch.autocompleters[cmd.Command]
	if ok {
		var newText string
		completions, newText = handler((*CommandAutocomplete)(cmd))
		if newText != "" {
			text = newText
		}
	}
	return completions, text, ok
}

func (ch *CommandProcessor) AutocompleteCommand(word string) (completions []string) {
	if word[0] != '/' {
		return
	}
	word = word[1:]
	for alias := range ch.aliases {
		if alias == word {
			return []string{"/" + alias}
		}
		if strings.HasPrefix(alias, word) {
			completions = append(completions, "/"+alias)
		}
	}
	for command := range ch.commands {
		if command == word {
			return []string{"/" + command}
		}
		if strings.HasPrefix(command, word) {
			completions = append(completions, "/"+command)
		}
	}
	return
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
