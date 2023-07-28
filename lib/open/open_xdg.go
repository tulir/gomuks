//go:build !windows && !darwin

package open

const Command = "xdg-open"

var Args []string
