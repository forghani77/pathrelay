package dialer

import (
	"strings"
	"testing"
)

func TestSelectDialerExclusive(t *testing.T) {
	if _, _, err := Select("h:1080", "h:3128"); err == nil {
		t.Fatal("expected error when both proxies are set")
	}
}

func TestSelectDialerHTTP(t *testing.T) {
	d, via, err := Select("", "h:3128")
	if err != nil || d == nil || !strings.Contains(via, "h:3128") {
		t.Fatalf("http dialer: d=%v via=%q err=%v", d, via, err)
	}
}

func TestSelectDialerSocks(t *testing.T) {
	d, via, err := Select("h:1080", "")
	if err != nil || d == nil || !strings.Contains(via, "h:1080") {
		t.Fatalf("socks dialer: d=%v via=%q err=%v", d, via, err)
	}
}

func TestSelectDialerDirect(t *testing.T) {
	d, via, err := Select("", "")
	if err != nil || d != nil || via != "" {
		t.Fatalf("direct: d=%v via=%q err=%v", d, via, err)
	}
}
