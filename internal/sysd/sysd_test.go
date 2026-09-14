package sysd

import (
	"strings"
	"testing"
)

func TestUnitContentSocks(t *testing.T) {
	got, err := UnitContent(":8080", "https://example.com:2096", "127.0.0.1:1080", "")
	if err != nil {
		t.Fatalf("UnitContent: %v", err)
	}
	for _, want := range []string{
		"ExecStart=" + BinPath,
		"'--listen'", "':8080'",
		"'--target'", "'https://example.com:2096'",
		"'--socks'", "'127.0.0.1:1080'",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("unit file missing %q\n---\n%s", want, got)
		}
	}
	if strings.Contains(got, "--http") {
		t.Errorf("socks unit must not contain --http:\n%s", got)
	}
}

func TestUnitContentHTTP(t *testing.T) {
	got, err := UnitContent(":80", "https://example.com", "", "10.0.0.1:3128")
	if err != nil {
		t.Fatalf("UnitContent: %v", err)
	}
	if !strings.Contains(got, "'--http'") || !strings.Contains(got, "'10.0.0.1:3128'") {
		t.Errorf("unit file missing --http flags:\n%s", got)
	}
	if strings.Contains(got, "--socks") {
		t.Errorf("http unit must not contain --socks:\n%s", got)
	}
}

func TestUnitContentBothProxiesRejected(t *testing.T) {
	if _, err := UnitContent(":80", "https://x", "h:1080", "h:3128"); err == nil {
		t.Fatal("expected error when both --socks and --http are set")
	}
}

func TestUnitContentEmptyTargetRejected(t *testing.T) {
	if _, err := UnitContent(":80", "  ", "h:1080", ""); err == nil {
		t.Fatal("expected error when target is empty")
	}
}

func TestSystemShellQuote(t *testing.T) {
	cases := map[string]string{
		"plain": "'plain'",
		":8080": "':8080'",
		"it's":  `'it'\''s'`,
		"a b":   "'a b'",
		"$HOME": "'$HOME'",
	}
	for in, want := range cases {
		if got := systemShellQuote(in); got != want {
			t.Errorf("systemShellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}
