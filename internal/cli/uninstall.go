package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"pathrelay/internal/shellcomp"
	"pathrelay/internal/sysd"
)

// NewUninstallCmd returns the `uninstall` subcommand.
func NewUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Stop and remove the systemd service, completion scripts and binary",
		Long: `Uninstall reverses everything install did: stops and disables the
systemd service, removes the unit file, deletes the completion scripts
and removes the binary from /usr/local/bin.

  sudo pathrelay uninstall

Requires root.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall()
		},
	}
}

func runUninstall() error {
	if runtime.GOOS == "windows" {
		return errors.New("uninstall is not supported on windows")
	}
	if os.Geteuid() != 0 {
		return errors.New("uninstall requires root; try: sudo pathrelay uninstall")
	}

	warns := 0
	warnf := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
		warns++
	}

	// 1. Systemd service
	if sysd.Available() {
		removed, err := sysd.Uninstall()
		switch {
		case err != nil:
			warnf("removing systemd service: %v", err)
		case removed:
			fmt.Printf("removed service:       %s\n", sysd.UnitPath)
		default:
			fmt.Printf("no service:            %s not present\n", sysd.UnitPath)
		}
	} else {
		fmt.Println("skipped service:       systemd not available on this system")
	}

	// 2. Completions
	removedComps, missing, err := shellcomp.Remove()
	switch {
	case err != nil:
		warnf("removing completions: %v", err)
	default:
		for _, path := range removedComps {
			fmt.Printf("removed completion:    %s\n", path)
		}
		for _, path := range missing {
			fmt.Printf("no completion:         %s not present\n", path)
		}
	}

	// 3. Binary
	removedBin, err := sysd.RemoveBinary()
	switch {
	case err != nil:
		warnf("removing binary: %v", err)
	case removedBin:
		fmt.Printf("removed binary:        %s\n", sysd.BinPath)
	default:
		fmt.Printf("no binary:             %s not present\n", sysd.BinPath)
	}

	if warns > 0 {
		return fmt.Errorf("uninstall finished with %d warning(s)", warns)
	}
	return nil
}
