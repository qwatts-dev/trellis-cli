//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

// execPassthrough replaces the current process with the plugin binary via the
// exec syscall; on success it never returns.
func execPassthrough(bin string, args []string) int {
	if err := syscall.Exec(bin, append([]string{bin}, args...), os.Environ()); err != nil {
		return 1
	}
	return 0
}
