package app_paths

import (
	"os"
	"path/filepath"
)

const (
	appData      = "AppData"
	localAppData = "LocalAppData"
)

func platformConfigDir() string {
	if c := os.Getenv(appData); c != "" {
		return filepath.Join(c, "Trellis CLI")
	}
	return ""
}

func platformCacheDir() string {
	if b := os.Getenv(localAppData); b != "" {
		return filepath.Join(b, "Trellis CLI")
	}
	return ""
}

func platformDataDir() string {
	if b := os.Getenv(localAppData); b != "" {
		return filepath.Join(b, "Trellis CLI")
	}
	return ""
}
