package wsl

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/roots/trellis-cli/command"
)

// WindowsTrustEntry describes a development site's self-signed certificate and
// whether it is currently present in the Windows Trusted Root store.
type WindowsTrustEntry struct {
	Site       string
	CertPath   string
	Exists     bool
	Thumbprint string
	Trusted    bool
}

// certDir is where TrustSslCerts caches certs extracted from the distro.
func (m *Manager) certDir() string {
	return filepath.Join(m.ConfigPath, "certs")
}

// TrustPaths reports the cached cert path and Windows trust-store status for
// each SSL-enabled development site. If site is non-empty, only that site is
// returned. It does not require the distro to be running.
func (m *Manager) TrustPaths(site string) []WindowsTrustEntry {
	var entries []WindowsTrustEntry
	for siteName, s := range m.Sites {
		if !s.SslEnabled() {
			continue
		}
		if site != "" && siteName != site {
			continue
		}

		certPath := filepath.Join(m.certDir(), siteName+".crt")
		entry := WindowsTrustEntry{Site: siteName, CertPath: certPath}
		if _, err := os.Stat(certPath); err == nil {
			entry.Exists = true
			entry.Thumbprint = certThumbprint(certPath)
			entry.Trusted = entry.Thumbprint != "" && rootStoreHas(entry.Thumbprint)
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Site < entries[j].Site })
	return entries
}

// Untrust removes the cached site certs from the Windows Trusted Root store
// (via an elevated certutil -delstore) and deletes the local cert copies.
// If site is non-empty, only that site is removed. Returns the number removed.
func (m *Manager) Untrust(site string) (int, error) {
	entries := m.TrustPaths(site)

	var thumbs []string
	var certFiles []string
	for _, e := range entries {
		if e.Thumbprint == "" {
			continue
		}
		thumbs = append(thumbs, e.Thumbprint)
		certFiles = append(certFiles, e.CertPath)
	}

	if len(thumbs) == 0 {
		return 0, nil
	}

	var cmds []string
	for _, t := range thumbs {
		cmds = append(cmds, fmt.Sprintf("certutil -delstore Root %s", t))
	}
	script := strings.Join(cmds, "; ")

	printStatus(m.ui, "Admin privileges required to remove certificates -- a UAC prompt will appear.")

	if err := command.Cmd("powershell", []string{
		"-Command",
		fmt.Sprintf(
			"Start-Process powershell.exe -Verb RunAs -Wait -ArgumentList '-NoProfile','-Command','%s'",
			script,
		),
	}).Run(); err != nil {
		return 0, err
	}

	for _, f := range certFiles {
		_ = os.Remove(f)
	}

	return len(thumbs), nil
}

// certThumbprint returns the SHA-1 thumbprint of a PEM/DER certificate file,
// or "" if it cannot be read.
func certThumbprint(path string) string {
	out, err := command.Cmd("powershell", []string{
		"-NoProfile", "-Command",
		fmt.Sprintf("(New-Object System.Security.Cryptography.X509Certificates.X509Certificate2('%s')).Thumbprint", path),
	}).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// rootStoreHas reports whether a certificate with the given thumbprint is in
// the LocalMachine Trusted Root store (where certutil -addstore Root writes).
func rootStoreHas(thumbprint string) bool {
	if thumbprint == "" {
		return false
	}
	out, err := command.Cmd("powershell", []string{
		"-NoProfile", "-Command",
		fmt.Sprintf("if (Test-Path 'Cert:\\LocalMachine\\Root\\%s') { 'yes' } else { 'no' }", thumbprint),
	}).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "yes"
}
