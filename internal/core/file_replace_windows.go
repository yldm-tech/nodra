//go:build windows

package core

import (
	"context"
	"errors"
	"os"
	"syscall"
)

// Windows error codes that package syscall does not name.
const (
	errorSharingViolation syscall.Errno = 32
	errorLockViolation    syscall.Errno = 33
)

// replaceRetryHook, when a test sets it, is called with each replacement failure that replaceFile classifies as contention, before the delay that follows it.
var replaceRetryHook func(error)

// replaceFile renames from over to. os.Rename is MoveFileEx with MOVEFILE_REPLACE_EXISTING, which cannot replace a file that another handle opened without delete sharing: an antivirus or indexer scan, a sync client, or a reader in this or another process (os.Open and os.ReadFile open files that way). It fails with access denied or a sharing or lock violation until that handle closes, so those failures are retried for up to replaceRetryBudget or until ctx is done.
//
// MoveFileEx also refuses to replace a destination with the read-only attribute, which a FileStorage.Perm without the owner-write bit gives every file it writes, while rename on other systems ignores the destination's mode. replaceFile therefore clears the attribute on a read-only destination first and restores it if the replacement fails; the file that replaces it carries the attribute from its own Perm.
func replaceFile(ctx context.Context, from, to string) error {
	var readOnly os.FileMode
	if info, err := os.Lstat(to); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0200 == 0 {
		readOnly = info.Mode().Perm()
		if err := os.Chmod(to, readOnly|0200); err != nil {
			return err
		}
	}
	err := retryTransient(ctx, replaceRetryBudget, func() error { return os.Rename(from, to) }, func(err error) bool {
		if !isReplaceContention(err) {
			return false
		}
		if replaceRetryHook != nil {
			replaceRetryHook(err)
		}
		return true
	})
	if err != nil && readOnly != 0 {
		_ = os.Chmod(to, readOnly)
	}
	return err
}

// isReplaceContention reports whether err is a failure that MoveFileEx returns while another handle holds the source or destination open.
func isReplaceContention(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == syscall.ERROR_ACCESS_DENIED || errno == errorSharingViolation || errno == errorLockViolation
}
