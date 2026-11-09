//go:build !windows

package tokencipher

import (
	"errors"
	"testing"
)

func TestDPAPI_HorsWindowsLeConstructeurRefuse(t *testing.T) {
	c, err := NewDPAPI()
	if c != nil || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("NewDPAPI hors Windows = %v, %v", c, err)
	}
}
