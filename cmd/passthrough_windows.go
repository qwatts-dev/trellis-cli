package cmd

import (
	"os"
	"os/exec"
)

// execPassthrough runs the plugin binary via exec.Command because Windows
// does not support the exec syscall used on POSIX systems.
func execPassthrough(bin string, args []string) int {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	if cmd.Run() == nil {
		return 0
	}
	return 1
}
