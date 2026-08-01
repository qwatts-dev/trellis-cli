package wsl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/roots/trellis-cli/app_paths"
	"github.com/roots/trellis-cli/command"
)

// WindowsHostsResolver implements vm.HostsResolver for Windows.
//
// It manages entries in C:\Windows\System32\drivers\etc\hosts,
// using the same trellis marker format as the upstream HostsFileResolver.
// Writing requires admin privileges; the resolver attempts a direct write
// first, then falls back to UAC elevation via PowerShell.
type WindowsHostsResolver struct {
	Hosts        []string
	hostsPath    string
	tmpHostsPath string
}

func NewWindowsHostsResolver(hosts []string) *WindowsHostsResolver {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}

	return &WindowsHostsResolver{
		Hosts:        hosts,
		hostsPath:    filepath.Join(systemRoot, "System32", "drivers", "etc", "hosts"),
		tmpHostsPath: filepath.Join(app_paths.DataDir(), "hosts"),
	}
}

func (h *WindowsHostsResolver) AddHosts(name string, ip string) error {
	// Skip the write (and UAC prompt) if this project's exact entry is already
	// present -- regardless of its position in the file. Comparing only this
	// block (not the whole file) avoids a needless admin elevation when other
	// trellis projects' blocks happen to sit after this one.
	current, readErr := os.ReadFile(h.hostsPath)
	if readErr == nil && bytes.Contains(current, h.hostsBlock(name, ip)) {
		return nil
	}

	content, err := h.addHostsContent(name, ip)
	if err != nil {
		return fmt.Errorf("error updating hosts file: %v", err)
	}

	return h.writeHostsFile(content)
}

func (h *WindowsHostsResolver) RemoveHosts(name string) error {
	content, err := h.removeHostsContent(name)
	if err != nil {
		return fmt.Errorf("error removing hosts entry: %v", err)
	}
	return h.writeHostsFile(content)
}

func (h *WindowsHostsResolver) addHostsContent(name string, ip string) ([]byte, error) {
	content, err := h.removeHostsContent(name)
	if err != nil {
		return nil, err
	}

	// Ensure blank line separation from existing hosts content.
	trimmed := bytes.TrimRight(content, "\r\n\t ")
	entry := append([]byte("\n\n"), h.hostsBlock(name, ip)...)
	return append(trimmed, entry...), nil
}

// hostsBlock returns the marker-delimited block for a project's hosts entry.
// The exact bytes are used both to write the entry and to detect an existing
// identical one (so AddHosts can skip a needless UAC prompt).
func (h *WindowsHostsResolver) hostsBlock(name string, ip string) []byte {
	return []byte(fmt.Sprintf(
		"## trellis-start-%s\n# Added by qwatts-dev/trellis-cli (https://github.com/qwatts-dev/trellis-cli)\n# To remove: trellis vm delete\n%s %s\n## trellis-end-%s\n",
		name, ip, strings.Join(h.Hosts, " "), name))
}

func (h *WindowsHostsResolver) removeHostsContent(name string) ([]byte, error) {
	header := fmt.Sprintf("## trellis-start-%s", name)
	footer := fmt.Sprintf("## trellis-end-%s", name)

	re := regexp.MustCompile(fmt.Sprintf(`%s([\s\S]*)%s\n`, header, footer))
	content, err := os.ReadFile(h.hostsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", h.hostsPath, err)
	}

	content = re.ReplaceAll(content, []byte{})
	return content, nil
}

func (h *WindowsHostsResolver) writeHostsFile(content []byte) error {
	if err := os.MkdirAll(filepath.Dir(h.tmpHostsPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(h.tmpHostsPath, content, 0644); err != nil {
		return err
	}

	// Try direct write (succeeds if process has admin rights).
	if err := os.WriteFile(h.hostsPath, content, 0644); err == nil {
		return nil
	}

	fmt.Printf("\r\nUpdating %s (admin privileges required -- a UAC prompt will appear)\r\n", h.hostsPath)

	// Elevate via UAC: launch PowerShell as admin to copy the temp file.
	// Use double quotes for the inner Copy-Item paths to avoid nested
	// single-quote escaping issues in PowerShell's -ArgumentList.
	copyCmd := fmt.Sprintf(
		`Copy-Item -LiteralPath \"%s\" -Destination \"%s\" -Force`,
		h.tmpHostsPath, h.hostsPath,
	)

	return command.Cmd("powershell", []string{
		"-Command",
		fmt.Sprintf(
			"Start-Process powershell.exe -Verb RunAs -Wait -ArgumentList '-NoProfile','-Command','%s'",
			copyCmd,
		),
	}).Run()
}
