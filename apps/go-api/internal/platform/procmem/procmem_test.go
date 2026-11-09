package procmem

import "testing"

func TestRSSMBPositive(t *testing.T) {
	if got := RSSMB(); got <= 0 {
		t.Fatalf("RSSMB() = %d, une mémoire résidente positive est attendue", got)
	}
}
