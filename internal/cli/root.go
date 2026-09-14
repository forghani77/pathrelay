// Package cli defines the pathrelay command tree.
package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"pathrelay/internal/version"
)

// Flag variables shared by root, install and uninstall commands.
var (
	flagListen string
	flagTarget string
	flagSocks  string
	flagHTTP   string
)

// Execute builds the command tree and runs it.
func Execute() error {
	return NewRootCmd().Execute()
}

// NewRootCmd returns the fully wired root command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "pathrelay",
		Version: version.Version,
		Short:   "HTTP reverse proxy that dials the upstream through a SOCKS5 or HTTP CONNECT proxy",
		Long: `pathrelay forwards incoming HTTP requests to a target base URL
through a SOCKS5 proxy (--socks) or an HTTP CONNECT proxy (--http).

Shell completion is built in: run "pathrelay completion --help".
"pathrelay install" installs a systemd service configured with the
flags you pass it; "pathrelay uninstall" removes it again.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(cmd)
		},
	}

	root.PersistentFlags().StringVarP(&flagListen, "listen", "l", ":80", "address to listen on")
	root.PersistentFlags().StringVarP(&flagTarget, "target", "t", "https://mydomain:2096", "upstream target base URL")
	root.PersistentFlags().StringVarP(&flagSocks, "socks", "s", "localhost:1080", "SOCKS5 proxy address (exclusive with --http)")
	root.PersistentFlags().StringVar(&flagHTTP, "http", "", "HTTP CONNECT proxy address (exclusive with --socks)")

	root.AddCommand(NewInstallCmd(), NewUninstallCmd())
	return root
}

// effectiveDialerArgs resolves the --socks/--http pair, which are mutually
// exclusive. Since --socks has a non-empty default, an explicit --socks flag
// is only detected via the Changed bit.
func effectiveDialerArgs(cmd *cobra.Command) (socks, httpProxy string, err error) {
	socks, httpProxy = flagSocks, flagHTTP
	if httpProxy != "" {
		if cmd.Flags().Changed("socks") {
			return "", "", errors.New("--socks and --http are mutually exclusive; pass only one")
		}
		socks = ""
	}
	return socks, httpProxy, nil
}
