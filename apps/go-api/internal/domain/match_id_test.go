package domain

import (
	"strings"
	"testing"
)

func TestIsValidMatchID(t *testing.T) {
	for _, ok := range []string{"11111111-aaaa-bbbb-cccc-000000000001", "ABCDEF-0123", strings.Repeat("a", 64)} {
		if !IsValidMatchID(ok) {
			t.Errorf("IsValidMatchID(%q) = false, attendu true", ok)
		}
	}
	for _, bad := range []string{"", "../../x", "a/b", `a\b`, "abc xyz", "ghij", strings.Repeat("a", 65), "a.b"} {
		if IsValidMatchID(bad) {
			t.Errorf("IsValidMatchID(%q) = true, attendu false", bad)
		}
	}
}
