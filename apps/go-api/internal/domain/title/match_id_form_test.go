package title

import (
	"strings"
	"testing"
)

func TestIsValidMatchID(t *testing.T) {
	for _, ok := range []string{
		"11111111-aaaa-bbbb-cccc-000000000001", "000d5950", "ABCDEF-0123", "abc",
		strings.Repeat("a", MaxMatchIDLength),
	} {
		if !IsValidMatchID(ok) {
			t.Errorf("IsValidMatchID(%q) = false, attendu true", ok)
		}
	}
	for _, bad := range []string{
		"", "../../x", "a/b", `a\b`, "abc xyz", "ghij", strings.Repeat("a", MaxMatchIDLength+1),
		"a.b", "..", ".", "' OR '1'='1", "GGGGGGGG-GGGG",
	} {
		if IsValidMatchID(bad) {
			t.Errorf("IsValidMatchID(%q) = true, attendu false", bad)
		}
	}
}
