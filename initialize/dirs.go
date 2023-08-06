package initialize

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func getRootDir(subdir string) string {
	rootDir := os.Getenv("GOMUKS_ROOT")
	if rootDir == "" {
		return ""
	}
	return filepath.Join(rootDir, subdir)
}

func UserCacheDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_CACHE_HOME")
	if dir == "" {
		dir = getRootDir("cache")
	}
	if dir == "" {
		dir, err = os.UserCacheDir()
		dir = filepath.Join(dir, "gomuks")
	}
	return
}

func UserDataDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_DATA_HOME")
	if dir != "" {
		return
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return UserConfigDir()
	}
	dir = getRootDir("data")
	if dir == "" {
		dir = os.Getenv("XDG_DATA_HOME")
	}
	if dir == "" {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("neither $XDG_DATA_HOME nor $HOME are defined")
		}
		dir = filepath.Join(dir, ".local", "share")
	}
	dir = filepath.Join(dir, "gomuks")
	return
}

func getXDGUserDir(name string) (dir string, err error) {
	cmd := exec.Command("xdg-user-dir", name)
	var out strings.Builder
	cmd.Stdout = &out
	err = cmd.Run()
	dir = strings.TrimSpace(out.String())
	return
}

func UserDownloadDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_DOWNLOAD_HOME")
	if dir != "" {
		return
	}
	dir, _ = getXDGUserDir("DOWNLOAD")
	if dir != "" {
		return
	}
	dir, err = os.UserHomeDir()
	dir = filepath.Join(dir, "Downloads")
	return
}

func UserConfigDir() (dir string, err error) {
	dir = os.Getenv("GOMUKS_CONFIG_HOME")
	if dir == "" {
		dir = getRootDir("config")
	}
	if dir == "" {
		dir, err = os.UserConfigDir()
		dir = filepath.Join(dir, "gomuks")
	}
	return
}
