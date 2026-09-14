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

// NewInstallCmd returns the `install` subcommand.
func NewInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the binary, shell completions and a systemd service from the current flags",
		Long: `Install copies the pathrelay binary to /usr/local/bin, writes shell
completion scripts, and creates a systemd service configured with the
flags given to install:

  sudo pathrelay install --listen :8080 --target https://mydomain:2096 --socks localhost:1080
  sudo pathrelay install --http 127.0.0.1:3128

Requires root and systemd.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			socks, httpProxy, err := effectiveDialerArgs(cmd)
			if err != nil {
				return err
			}
			return runInstall(cmd.Root(), socks, httpProxy)
		},
	}
}

func runInstall(root *cobra.Command, socks, httpProxy string) error {
	if runtime.GOOS == "windows" {
		return errors.New("install is not supported on windows; build pathrelay.exe and place it on your PATH manually")
	}
	if os.Geteuid() != 0 {
		return errors.New("install requires root; try: sudo pathrelay install ...")
	}

	// 1. Binary
	dest, err := sysd.InstallBinary()
	if err != nil {
		return err
	}
	fmt.Printf("installed binary:      %s\n", dest)

	// 2. Completions (non-fatal detail report)
	installedComps, skipped, err := shellcomp.Install(root)
	if err != nil {
		return err
	}
	for _, path := range installedComps {
		fmt.Printf("installed completion:  %s\n", path)
	}
	for _, shell := range skipped {
		fmt.Printf("skipped completion:    %s (no completion dir found)\n", shell)
	}

	// 3. Systemd service
	if !sysd.Available() {
		fmt.Println("skipped service:       systemd not available on this system")
		return nil
	}
	if err := sysd.Install(flagListen, flagTarget, socks, httpProxy); err != nil {
		return fmt.Errorf("installing systemd service: %w", err)
	}
	fmt.Printf("installed service:     %s (enabled and started)\n", sysd.UnitPath)
	fmt.Printf("service status:        systemctl status %s\n", sysd.UnitName)
	fmt.Printf("service logs:          journalctl -u %s -f\n", sysd.UnitName)
	return nil
}
