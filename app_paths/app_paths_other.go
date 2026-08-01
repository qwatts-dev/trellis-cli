//go:build !windows

package app_paths

func platformConfigDir() string { return "" }
func platformCacheDir() string  { return "" }
func platformDataDir() string   { return "" }
