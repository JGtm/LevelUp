package replaybuild

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// TestLePasDuRejeuEstCeluiDeLaPage : sans réglage, la cuisson se fait au pas que la page lit, jamais au
// défaut de construction du décodeur ; un réglage explicite gagne.
func TestLePasDuRejeuEstCeluiDeLaPage(t *testing.T) {
	b := &Builder{}
	if got := b.pasDuRejeu(); got != replay.HistoricalFrameIntervalMS {
		t.Fatalf("pas sans réglage = %d ms, attendu %d ms", got, replay.HistoricalFrameIntervalMS)
	}
	if replay.DefaultFrameIntervalMS == replay.HistoricalFrameIntervalMS {
		t.Fatalf("le défaut du décodeur vaut le pas de la page : ce test ne distingue plus rien")
	}
	b.WithFrameInterval(replay.DefaultFrameIntervalMS)
	if got := b.pasDuRejeu(); got != replay.DefaultFrameIntervalMS {
		t.Fatalf("pas réglé = %d ms, attendu %d ms", got, replay.DefaultFrameIntervalMS)
	}
}
