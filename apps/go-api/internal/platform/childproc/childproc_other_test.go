//go:build !windows

package childproc

import (
	"context"
	"testing"
)

func TestCommandSansEffetHorsWindows(t *testing.T) {
	if cmd := Command(context.Background(), "sh"); cmd.SysProcAttr != nil {
		t.Fatalf("SysProcAttr doit rester nil hors Windows : %+v", cmd.SysProcAttr)
	}
}
