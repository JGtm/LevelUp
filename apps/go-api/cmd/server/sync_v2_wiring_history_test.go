// Package main — sync_v2_wiring_history_test.go : le cycle V2 range la fin d'historique et le
// marque-page d'un joueur par l'ouverture en écriture SOUS LE BAIL du joueur, la même que le
// post-sync. Sans cet écrivain, aucun joueur ne sort du rattrapage de l'historique (le cycle
// journalise « sans écrivain câblé » et laisse l'état en l'état) : ratchet sur la source du câblage.
package main

import (
	"os"
	"strings"
	"testing"
)

func TestSyncV2WiringWiresHistoryWriterUnderPlayerLease(t *testing.T) {
	const src = "sync_v2_wiring.go"
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("lecture %s: %v", src, err)
	}
	text := string(b)
	if !strings.Contains(text, "WithHistoryWriter(playerDBOpenerRW)") {
		t.Error("le câblage du cycle V2 ne pose pas WithHistoryWriter(playerDBOpenerRW) : la fin d'historique ne serait jamais rangée")
	}
	if !strings.Contains(text, "playerDBOpenerRW := postSyncPlayerOpener(") {
		t.Error("playerDBOpenerRW n'est plus l'ouverture sous bail joueur (postSyncPlayerOpener) : une écriture de sync_meta " +
			"hors du bail casserait la règle d'un seul écrivain par base joueur")
	}
}
