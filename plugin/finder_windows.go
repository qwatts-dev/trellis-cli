package plugin

import (
	"os"
	"path/filepath"
	"strings"
)

// isExecutableFile treats files with a known Windows executable extension as
// executable, since Windows has no POSIX executable mode bit.
func isExecutableFile(fullPath string, _ os.FileInfo) bool {
	switch strings.ToLower(filepath.Ext(fullPath)) {
	case ".bat", ".cmd", ".com", ".exe", ".ps1":
		return true
	}
	return false
}
