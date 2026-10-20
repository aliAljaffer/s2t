package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const brewFormula = "alialjaffer/tap/s2t"

var upgradeCmd = &cobra.Command{
	Use:           "upgrade",
	Short:         "Upgrade s2t to the latest release (Homebrew installs only)",
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
		if !isHomebrewPath(exe) {
			return fmt.Errorf("%s is not a Homebrew install; upgrade it the way you installed it (see the README)", exe)
		}
		brew := exec.Command("brew", "upgrade", brewFormula)
		brew.Stdout, brew.Stderr = os.Stdout, os.Stderr
		return brew.Run()
	},
}

func isHomebrewPath(p string) bool {
	return strings.Contains(filepath.ToSlash(p), "/Cellar/")
}
