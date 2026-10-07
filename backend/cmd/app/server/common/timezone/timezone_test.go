package timezone

import "testing"

func TestValid(t *testing.T) {
	for _, name := range []string{"UTC", "Asia/Shanghai", "Etc/GMT-8", "America/New_York"} {
		if !Valid(name) {
			t.Errorf("Valid(%q) = false", name)
		}
	}
	for _, name := range []string{"", "Local", "Mars/Base", "../etc/passwd"} {
		if Valid(name) {
			t.Errorf("Valid(%q) = true", name)
		}
	}
}

func TestServerReturnsValidZone(t *testing.T) {
	if name := Server(); !Valid(name) {
		t.Fatalf("Server() = %q 不是合法时区", name)
	}
}

func TestFromZoneinfoPath(t *testing.T) {
	cases := map[string]string{
		"/usr/share/zoneinfo/Asia/Shanghai":       "Asia/Shanghai",
		"../usr/share/zoneinfo/Etc/UTC":           "Etc/UTC",
		"/var/db/timezone/zoneinfo/Europe/Berlin": "Europe/Berlin",
		"/etc/localtime":                          "",
	}
	for path, want := range cases {
		if got := fromZoneinfoPath(path); got != want {
			t.Errorf("fromZoneinfoPath(%q) = %q, want %q", path, got, want)
		}
	}
}
