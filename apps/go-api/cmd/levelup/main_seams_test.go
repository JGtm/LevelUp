package main

// main_seams_test.go — la CLI câble les seams title-owned à son démarrage.
//
// POURQUOI (2026-09-16). `levelup sync-full --gamertag …` rendait
// `post-sync: PANIC récupéré … classifier LUSR non câblé` puis
// `perf_scores=0 lusr=0 citations=0 dominance=0` : le binaire ne posait que le provider
// d'étapes de migration, pas les classifiers. Ce test exerce le MÊME chemin de démarrage que
// main() (wireStartupSeams) après avoir dé-câblé le classifier par défaut : sans l'appel à
// titleseams.RegisterAll, GetLUSRChain panique et le test échoue.

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/games/titleseams"
	syncskill "levelup/go-api/internal/sync/skill"
)

func TestWireStartupSeamsCableLeClassifierLUSR(t *testing.T) {
	// Dé-câble le classifier par défaut pour que l'absence de câblage se voie
	// (le paquet est global au process : on le remet en sortie).
	syncskill.SetLUSRChainClassifier(nil)
	t.Cleanup(func() { titleseams.RegisterAll("") })

	if err := syncskill.ValidateLUSRChainClassifierWired(); err == nil {
		t.Fatal("préparation du test : le classifier LUSR est encore câblé après SetLUSRChainClassifier(nil)")
	}

	wireStartupSeams(&config.AppConfig{RepoRoot: t.TempDir()})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetLUSRChain a paniqué après le démarrage de la CLI : %v — les seams "+
				"title-owned ne sont pas câblés par wireStartupSeams", r)
		}
	}()
	if got := syncskill.GetLUSRChain("BTB:Slayer"); got == "" {
		t.Error("GetLUSRChain(\"BTB:Slayer\") rend une chaîne vide après le démarrage de la CLI")
	}
	if err := syncskill.ValidateLUSRChainClassifierWired(); err != nil {
		t.Errorf("ValidateLUSRChainClassifierWired après wireStartupSeams : %v", err)
	}
}
