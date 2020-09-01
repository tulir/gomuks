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

package matrix

import (
	"context"
	"fmt"
	"image"
	"os"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/pkg/errors"
	"gopkg.in/vansante/go-ffprobe.v2"

	"maunium.net/go/gomuks/debug"
	"maunium.net/go/mautrix/event"
)

func getImageInfo(path string) (event.FileInfo, error) {
	var info event.FileInfo
	file, err := os.Open(path)
	if err != nil {
		return info, errors.Wrap(err, "failed to open image to get info")
	}
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return info, errors.Wrap(err, "failed to get image info")
	}
	info.Width = cfg.Width
	info.Height = cfg.Height
	return info, nil
}

func getFFProbeInfo(mimeClass, path string) (msgtype event.MessageType, info event.FileInfo, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	var probedInfo *ffprobe.ProbeData
	probedInfo, err = ffprobe.ProbeURL(ctx, path)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("failed to get %s info with ffprobe", mimeClass))
		return
	}
	if mimeClass == "audio" {
		msgtype = event.MsgAudio
		stream := probedInfo.FirstAudioStream()
		if stream != nil {
			info.Duration = int(stream.DurationTs)
		}
	} else {
		msgtype = event.MsgVideo
		stream := probedInfo.FirstVideoStream()
		if stream != nil {
			info.Duration = int(stream.DurationTs)
			info.Width = stream.Width
			info.Height = stream.Height
		}
	}
	return
}

func getMediaInfo(path string) (msgtype event.MessageType, info event.FileInfo, err error) {
	var mime *mimetype.MIME
	mime, err = mimetype.DetectFile(path)
	if err != nil {
		err = errors.Wrap(err, "failed to get content type")
		return
	}

	mimeClass := strings.SplitN(mime.String(), "/", 2)[0]
	switch mimeClass {
	case "image":
		msgtype = event.MsgImage
		info, err = getImageInfo(path)
		if err != nil {
			debug.Printf("Failed to get image info for %s: %v", err)
			err = nil
		}
	case "audio", "video":
		msgtype, info, err = getFFProbeInfo(mimeClass, path)
		if err != nil {
			debug.Printf("Failed to get ffprobe info for %s: %v", err)
			err = nil
		}
	default:
		msgtype = event.MsgFile
	}
	info.MimeType = mime.String()

	return
}
