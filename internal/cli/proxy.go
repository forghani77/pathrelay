package cli

import (
	"github.com/spf13/cobra"

	"pathrelay/internal/app"
)

// runProxy starts the reverse proxy in the foreground (root command).
func runProxy(cmd *cobra.Command) error {
	socks, httpProxy, err := effectiveDialerArgs(cmd)
	if err != nil {
		return err
	}

	cfg := app.Config{
		Listen: flagListen,
		Target: flagTarget,
		Socks:  socks,
		HTTP:   httpProxy,
	}
	return cfg.Run()
}
