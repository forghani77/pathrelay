// Package sysd installs and removes the pathrelay systemd service.
package sysd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

const (
	UnitName = "pathrelay.service"
	UnitPath = "/etc/systemd/system/" + UnitName
	BinPath  = "/usr/local/bin/pathrelay"
)

// systemShellQuote quotes s for use inside a systemd unit string, where the
// value is ultimately parsed with /bin/sh -c.
func systemShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// UnitContent renders the systemd unit for the given configuration.
func UnitContent(listen, target, socks, httpProxy string) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", errors.New("--target must not be empty")
	}

	args := []string{
		"--listen", listen,
		"--target", target,
	}
	switch {
	case socks != "" && httpProxy != "":
		return "", errors.New("--socks and --http are mutually exclusive; pass only one")
	case httpProxy != "":
		args = append(args, "--http", httpProxy)
	default:
		args = append(args, "--socks", socks)
	}

	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = systemShellQuote(a)
	}

	tmpl := template.Must(template.New("unit").Parse(`[Unit]
Description=pathrelay HTTP reverse proxy
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{.ExecStart}}
Restart=on-failure
RestartSec=5
# Hardening
DynamicUser=yes
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
PrivateDevices=yes
RestrictAddressFamilies=AF_INET AF_INET6
RestrictNamespaces=yes
LockPersonality=yes
MemoryDenyWriteExecute=yes
SystemCallArchitectures=native

[Install]
WantedBy=multi-user.target
`))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"ExecStart": BinPath + " " + strings.Join(quoted, " "),
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Available reports whether the system boots with systemd.
func Available() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// systemctl runs systemctl with a timeout and returns trimmed combined output.
func systemctl(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "systemctl", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// InstallBinary copies the currently running executable to BinPath.
func InstallBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating current binary: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving current binary path: %w", err)
	}

	if exe == BinPath {
		return BinPath, nil // already installed at the destination
	}

	data, err := os.ReadFile(exe)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", exe, err)
	}
	if err := os.MkdirAll(filepath.Dir(BinPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(BinPath, data, 0o755); err != nil {
		return "", fmt.Errorf("writing %s: %w", BinPath, err)
	}
	return BinPath, nil
}

// RemoveBinary deletes the installed binary if present.
func RemoveBinary() (removed bool, err error) {
	if _, statErr := os.Stat(BinPath); statErr != nil {
		return false, nil
	}
	if err := os.Remove(BinPath); err != nil {
		return false, fmt.Errorf("removing %s: %w", BinPath, err)
	}
	return true, nil
}

// Install writes the unit file, reloads systemd, and enables+starts the
// service.
func Install(listen, target, socks, httpProxy string) error {
	content, err := UnitContent(listen, target, socks, httpProxy)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(UnitPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(UnitPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", UnitPath, err)
	}

	ctx := context.Background()
	if out, err := systemctl(ctx, "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w\n%s", err, out)
	}
	if out, err := systemctl(ctx, "enable", "--now", UnitName); err != nil {
		return fmt.Errorf("systemctl enable --now %s: %w\n%s", UnitName, err, out)
	}
	return nil
}

// Uninstall stops and disables the service, removes the unit file and
// reloads systemd. Missing unit is reported as (false, nil).
func Uninstall() (removed bool, err error) {
	if _, statErr := os.Stat(UnitPath); statErr != nil {
		return false, nil
	}

	ctx := context.Background()
	if out, err := systemctl(ctx, "disable", "--now", UnitName); err != nil {
		fmt.Fprintf(os.Stderr, "warning: systemctl disable --now %s: %v\n%s\n", UnitName, err, out)
	}
	if err := os.Remove(UnitPath); err != nil {
		return false, fmt.Errorf("removing %s: %w", UnitPath, err)
	}
	if out, err := systemctl(ctx, "daemon-reload"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: systemctl daemon-reload: %v\n%s\n", err, out)
	}
	return true, nil
}
