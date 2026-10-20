package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	brewFormula = "alialjaffer/tap/s2t"
	goPackage   = "github.com/alialjaffer/s2t@latest"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade s2t to the latest release",
	Long: `upgrade detects how s2t was installed and upgrades it the same way:
Homebrew installs run "brew upgrade", "go install" builds run "go install ...@latest".
Other installs (make install, a downloaded binary) get instructions instead.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		switch {
		case isHomebrewPath(exe):
			return runPassthrough("brew", "upgrade", brewFormula)
		case isGoInstallPath(exe):
			return runPassthrough("go", "install", goPackage)
		default:
			return fmt.Errorf("%s was not installed with Homebrew or go install; to upgrade, run \"git pull && make install\" in your clone, or download the latest binary from https://github.com/alialjaffer/s2t/releases", exe)
		}
	},
}

func runPassthrough(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

func isHomebrewPath(p string) bool {
	return strings.Contains(filepath.ToSlash(p), "/Cellar/")
}

// isGoInstallPath reports whether exe sits in the directory "go install"
// writes to: $GOBIN, or the first $GOPATH entry's bin.
func isGoInstallPath(exe string) bool {
	out, err := exec.Command("go", "env", "GOBIN", "GOPATH").Output()
	if err != nil {
		return false
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	dir := ""
	if len(lines) > 0 {
		dir = strings.TrimSpace(lines[0])
	}
	if dir == "" && len(lines) > 1 {
		dir = filepath.Join(filepath.SplitList(strings.TrimSpace(lines[1]))[0], "bin")
	}
	return dir != "" && filepath.Dir(exe) == filepath.Clean(dir)
}
