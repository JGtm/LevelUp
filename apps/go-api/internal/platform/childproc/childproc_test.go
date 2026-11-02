package childproc

import (
	"context"
	"testing"
)

func TestCommandNeFabriquePasUneAutreCommande(t *testing.T) {
	cmd := Command(context.Background(), "un-binaire", "a", "b")
	if cmd.Path == "" || len(cmd.Args) != 3 || cmd.Args[1] != "a" || cmd.Args[2] != "b" {
		t.Fatalf("commande alteree : path=%q args=%v", cmd.Path, cmd.Args)
	}
}
