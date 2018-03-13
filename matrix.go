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

package main

import (
	"fmt"

	"github.com/matrix-org/gomatrix"
)

type MatrixContainer struct {
	lient   *gomatrix.Client
	config  *Config
	running bool
	stop    chan bool
}

func (c *MatrixContainer) Initialized() bool {
	return c.lient != nil
}

func (c *MatrixContainer) Init(config *Config) error {
	c.config = config

	if c.lient != nil {
		c.lient.StopSync()
	}

	if len(c.config.HS) == 0 {
		return fmt.Errorf("no homeserver in config")
	}

	var mxid, accessToken string
	if c.config.Session != nil {
		accessToken = c.config.Session.AccessToken
		mxid = c.config.MXID
	}

	var err error
	c.lient, err = gomatrix.NewClient(c.config.HS, mxid, accessToken)
	if err != nil {
		return err
	}

	c.stop = make(chan bool, 1)

	if c.config.Session != nil {
		go c.Start()
	}
	return nil
}

func (c *MatrixContainer) Login(user, password string) error {
	resp, err := c.lient.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     user,
		Password: password,
	})
	if err != nil {
		return err
	}
	c.lient.SetCredentials(resp.UserID, resp.AccessToken)
	c.config.MXID = resp.UserID
	c.config.Save()

	c.config.Session = c.config.NewSession(resp.UserID)
	c.config.Session.AccessToken = resp.AccessToken
	c.config.Session.Save()

	go c.Start()

	return nil
}

func (c *MatrixContainer) Stop() {
	c.stop <- true
	c.lient.StopSync()
}

func (c *MatrixContainer) Start() {
	debug.Print("Starting sync...")
	c.running = true
	c.lient.Store = c.config.Session

	syncer := c.lient.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", c.HandleMessage)

	for {
		select {
		case <-c.stop:
			debug.Print("Stopping sync...")
			c.running = false
			return
		default:
			if err := c.lient.Sync(); err != nil {
				debug.Print("Sync() errored", err)
			}
		}
	}
}

func (c *MatrixContainer) HandleMessage(evt *gomatrix.Event) {
	debug.Print("Message received")
}
