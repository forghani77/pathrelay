package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestEffectiveDialerArgsDefault(t *testing.T) {
	// No flags changed: default socks is used, http empty.
	root := NewRootCmd()
	socks, httpProxy, err := effectiveDialerArgs(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if socks != "localhost:1080" || httpProxy != "" {
		t.Fatalf("got socks=%q http=%q", socks, httpProxy)
	}
}

func TestEffectiveDialerArgsBothSet(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"--socks", "a:1080", "--http", "b:3128"})
	// Bind args so the Changed bits are set without executing RunE.
	if err := root.ParseFlags([]string{"--socks", "a:1080", "--http", "b:3128"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if _, _, err := effectiveDialerArgs(root); err == nil {
		t.Fatal("expected error when both --socks and --http are set")
	}
}

func TestEffectiveDialerArgsHTTPOnly(t *testing.T) {
	root := NewRootCmd()
	if err := root.ParseFlags([]string{"--http", "b:3128"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	socks, httpProxy, err := effectiveDialerArgs(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if socks != "" || httpProxy != "b:3128" {
		t.Fatalf("got socks=%q http=%q", socks, httpProxy)
	}
}

func TestCommandTreeHelp(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"install", "uninstall", "--listen", "--target", "--socks", "--http"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q", want)
		}
	}
}

func TestUninstallHelp(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"uninstall", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "uninstall") {
		t.Error("uninstall help output unexpected")
	}
}

// Ensure the completion command tree works (cobra built-in, added on Execute).
func TestCompletionCommandPresent(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"completion"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"bash", "zsh", "fish", "powershell"} {
		if !strings.Contains(out, want) {
			t.Errorf("completion help missing %q", want)
		}
	}
}
