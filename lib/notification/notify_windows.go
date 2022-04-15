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

package notification

import (
	"gopkg.in/toast.v1"
)

func Send(title, text string, critical, sound bool) error {
	notification := toast.Notification{
		AppID:    "gomuks",
		Title:    title,
		Message:  text,
		Audio:    toast.Silent,
		Duration: toast.Short,
		// 		Icon: ...,
	}
	if sound {
		notification.Audio = toast.IM
	}
	if critical {
		notification.Duration = toast.Long
	}
	return notification.Push()
}
