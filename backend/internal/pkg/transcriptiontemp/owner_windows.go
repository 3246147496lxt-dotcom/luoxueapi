//go:build windows

package transcriptiontemp

import "os"

func verifyWorkspaceOwnership(_ os.FileInfo) error {
	// os.TempDir resolves to the current Windows identity's temporary
	// directory. Windows enforces its ownership boundary through ACLs rather
	// than Unix UID/mode fields, so the parent ACL is the applicable guard.
	return nil
}
