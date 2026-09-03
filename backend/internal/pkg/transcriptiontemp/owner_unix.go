//go:build linux || darwin

package transcriptiontemp

import (
	"errors"
	"os"
	"syscall"
)

func verifyWorkspaceOwnership(info os.FileInfo) error {
	return verifyWorkspaceOwnershipForUID(info, uint32(os.Geteuid()))
}

func verifyWorkspaceOwnershipForUID(info os.FileInfo, expectedUID uint32) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != expectedUID {
		return errors.New("transcription temporary directory is not owned by the service user")
	}
	return nil
}
