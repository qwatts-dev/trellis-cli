//go:build !windows

package plugin

import "os"

// isExecutableFile checks the POSIX executable mode bits.
func isExecutableFile(_ string, info os.FileInfo) bool {
	return info.Mode()&0111 != 0
}
