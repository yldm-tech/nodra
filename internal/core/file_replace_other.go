//go:build !windows

package core

import (
	"context"
	"os"
)

// replaceFile renames from over to. Rename on these systems replaces the destination atomically whatever its mode and whoever holds it open, so there is nothing to retry.
func replaceFile(_ context.Context, from, to string) error {
	return os.Rename(from, to)
}
