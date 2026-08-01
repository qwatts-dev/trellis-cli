package cmd

import (
	"flag"
	"fmt"
	"runtime"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/roots/trellis-cli/pkg/trust"
	"github.com/roots/trellis-cli/pkg/wsl"
	"github.com/roots/trellis-cli/trellis"
)

type VmTrustPathsCommand struct {
	UI      cli.Ui
	Trellis *trellis.Trellis
	flags   *flag.FlagSet
	site    string
}

func NewVmTrustPathsCommand(ui cli.Ui, trellis *trellis.Trellis) *VmTrustPathsCommand {
	c := &VmTrustPathsCommand{UI: ui, Trellis: trellis}
	c.init()
	return c
}

func (c *VmTrustPathsCommand) init() {
	c.flags = flag.NewFlagSet("", flag.ContinueOnError)
	c.flags.Usage = func() { c.UI.Info(c.Help()) }
	c.flags.StringVar(&c.site, "site", "", "Show only the named site.")
}

func (c *VmTrustPathsCommand) Run(args []string) int {
	if err := c.Trellis.LoadProject(); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.Trellis.CheckVirtualenv(c.UI)

	if windowsHostRequired(c.Trellis, c.UI, "vm trust paths") {
		return 1
	}

	if err := c.flags.Parse(args); err != nil {
		return 1
	}

	if err := (&CommandArgumentValidator{required: 0, optional: 0}).validate(c.flags.Args()); err != nil {
		c.UI.Error(err.Error())
		c.UI.Output(c.Help())
		return 1
	}

	// On Windows the fork trusts certs in the Windows Trusted Root store via
	// certutil (not pkg/trust), so report from there instead.
	if runtime.GOOS == "windows" {
		manager, err := newVmManager(c.Trellis, c.UI)
		if err != nil {
			c.UI.Error("Error: " + err.Error())
			return 1
		}
		wslManager, ok := manager.(*wsl.Manager)
		if !ok {
			c.UI.Error("'trellis vm trust paths' requires the WSL backend on Windows.")
			return 1
		}

		entries := wslManager.TrustPaths(c.site)
		if len(entries) == 0 {
			c.UI.Info("No SSL-enabled sites in the development environment.")
			return 0
		}
		for i, e := range entries {
			if i > 0 {
				c.UI.Info("")
			}
			status := "not trusted (run `trellis vm trust`)"
			if e.Trusted {
				status = "trusted (Windows Trusted Root store)"
			}
			c.UI.Info(e.Site)
			c.UI.Info("  cert:   " + e.CertPath)
			if e.Thumbprint != "" {
				c.UI.Info("  sha1:   " + e.Thumbprint)
			}
			c.UI.Info("  status: " + status)
		}
		return 0
	}

	state, err := trust.Load()
	if err != nil {
		c.UI.Error("Error reading trust state: " + err.Error())
		return 1
	}

	project := c.Trellis.Path
	entries := state.EntriesForProject(project)
	if c.site != "" {
		filtered := entries[:0]
		for _, e := range entries {
			if e.Site == c.site {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if len(entries) == 0 {
		c.UI.Info(fmt.Sprintf("no trust entries for %s. Run `trellis vm trust` first.", project))
		return 0
	}

	for i, entry := range entries {
		if i > 0 {
			c.UI.Info("")
		}
		key := entry.KeyPath
		if key == "" {
			key = "<not exported>"
		}
		c.UI.Info(entry.Site)
		c.UI.Info("  cert: " + entry.CertPath)
		c.UI.Info("  key:  " + key)
	}

	return 0
}

func (c *VmTrustPathsCommand) Synopsis() string {
	return "Prints the host paths of the exported SSL cert and key for each trusted site."
}

func (c *VmTrustPathsCommand) Help() string {
	helpText := `
Usage: trellis vm trust paths [options]

Prints one line per trusted site for the current project, showing where the
exported cert and private key live on the host. Use these paths to wire up
host-side tooling (Vite, Playwright, curl) against the same cert the VM
serves.

Options:
      --site  Show only the named site.
  -h, --help  Show this help.
`
	return strings.TrimSpace(helpText)
}
