package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"pathrelay/internal/sysd"
)

// NewUninstallCmd returns the `uninstall` subcommand.
func NewUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Stop and remove the systemd service (binary and completions are kept)",
		Long: `Uninstall stops and disables the pathrelay systemd service and
removes its unit file. The binary and shell completions are left in
place — only the service is removed.

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

	if !sysd.Available() {
		return errors.New("systemd is not available on this system; nothing to uninstall")
	}

	removed, err := sysd.Uninstall()
	switch {
	case err != nil:
		return fmt.Errorf("removing systemd service: %w", err)
	case removed:
		fmt.Printf("removed service:    %s\n", sysd.UnitPath)
		fmt.Printf("binary kept:        %s\n", sysd.BinPath)
		fmt.Println("completions kept (see 'pathrelay completion --help')")
	default:
		fmt.Printf("no service:         %s not present\n", sysd.UnitPath)
	}
	return nil
}
