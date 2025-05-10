// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
	_ "go.mau.fi/util/dbutil/litestream"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/exzerolog"
	"go.mau.fi/zeroconfig"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

var writerTypeReadline zeroconfig.WriterType = "hitest_readline"

func main() {
	hicli.InitialDeviceDisplayName = "mautrix hitest"
	rl := exerrors.Must(readline.New("> "))
	defer func() {
		_ = rl.Close()
	}()
	zeroconfig.RegisterWriter(writerTypeReadline, func(config *zeroconfig.WriterConfig) (io.Writer, error) {
		return rl.Stdout(), nil
	})
	debug := zerolog.DebugLevel
	log := exerrors.Must((&zeroconfig.Config{
		MinLevel: &debug,
		Writers: []zeroconfig.WriterConfig{{
			Type:   writerTypeReadline,
			Format: zeroconfig.LogFormatPrettyColored,
		}},
	}).Compile())
	exzerolog.SetupDefaults(log)

	rawDB := exerrors.Must(dbutil.NewWithDialect("hicli.db", "sqlite3-fk-wal"))
	ctx := log.WithContext(context.Background())
	cli := hicli.New(rawDB, nil, *log, []byte("meow"), func(a any) {
		_, _ = fmt.Fprintf(rl, "Received event of type %T\n", a)
		switch evt := a.(type) {
		case *jsoncmd.SyncComplete:
			for _, room := range evt.Rooms {
				name := "name unset"
				if room.Meta.Name != nil {
					name = *room.Meta.Name
				}
				_, _ = fmt.Fprintf(rl, "Room %s (%s) in sync:\n", name, room.Meta.ID)
				_, _ = fmt.Fprintf(rl, "  Preview: %d, sort: %v\n", room.Meta.PreviewEventRowID, room.Meta.SortingTimestamp)
				_, _ = fmt.Fprintf(rl, "  Timeline: +%d %v, reset: %t\n", len(room.Timeline), room.Timeline, room.Reset)
			}
		case *jsoncmd.EventsDecrypted:
			for _, decrypted := range evt.Events {
				_, _ = fmt.Fprintf(rl, "Delayed decryption of %s completed: %s / %s\n", decrypted.ID, decrypted.DecryptedType, decrypted.Decrypted)
			}
			if evt.PreviewEventRowID != 0 {
				_, _ = fmt.Fprintf(rl, "Room preview updated: %+v\n", evt.PreviewEventRowID)
			}
		case *jsoncmd.Typing:
			_, _ = fmt.Fprintf(rl, "Typing list in %s: %+v\n", evt.RoomID, evt.UserIDs)
		}
	})
	userID, _ := cli.DB.Account.GetFirstUserID(ctx)
	exerrors.PanicIfNotNil(cli.Start(ctx, userID, nil))
	if !cli.IsLoggedIn() {
		rl.SetPrompt("User ID: ")
		userID := id.UserID(exerrors.Must(rl.Readline()))
		_, serverName := exerrors.Must2(userID.Parse())
		discovery := exerrors.Must(mautrix.DiscoverClientAPI(ctx, serverName))
		password := exerrors.Must(rl.ReadPassword("Password: "))
		recoveryCode := exerrors.Must(rl.ReadPassword("Recovery code: "))
		exerrors.PanicIfNotNil(cli.LoginAndVerify(ctx, discovery.Homeserver.BaseURL, userID.String(), string(password), string(recoveryCode)))
	}
	rl.SetPrompt("> ")

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "send":
			resp, err := cli.Send(ctx, id.RoomID(fields[1]), event.EventMessage, &event.MessageEventContent{
				Body:    strings.Join(fields[2:], " "),
				MsgType: event.MsgText,
			}, false, false)
			_, _ = fmt.Fprintln(rl, err)
			_, _ = fmt.Fprintf(rl, "%+v\n", resp)
		}
	}
	cli.Stop()
}
