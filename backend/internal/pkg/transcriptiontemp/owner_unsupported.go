//go:build !linux && !darwin && !windows

package transcriptiontemp

import (
	"errors"
	"os"
)

func verifyWorkspaceOwnership(_ os.FileInfo) error {
	return errors.New("transcription temporary directory ownership cannot be verified on this platform")
}
