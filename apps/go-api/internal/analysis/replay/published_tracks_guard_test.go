package replay

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// GARDE-RAIL du helper `keepOfPublishedTracks` (lot 3.1).
//
// La règle n°6 du dépôt : une factorisation SANS garde-rail re-diverge. La leçon est
// chiffrée (prédicat bot passé de 8 à 36 copies après centralisation). Ici la divergence
// serait INVISIBLE — un calque garderait des éléments qu'un autre écarte, sur la même
// fiche, sans qu'aucun compteur ne bouge.
//
// Ce test échoue si un second endroit du paquet reconstruit un ensemble de slots publiés
// au lieu d'appeler le helper.

// slotSetPattern reconnaît la construction d'un ensemble de slots : `<map>[<x>.Slot] = true`.
var slotSetPattern = regexp.MustCompile(`\w+\[\w+\.Slot\]\s*=\s*true`)

// publishedSlotsOwner est le SEUL fichier de production autorisé à construire cet ensemble.
const publishedSlotsOwner = "published_tracks.go"

func TestUnSeulConstructeurDEnsembleDeSlotsPublies(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet: %v", err)
	}
	var offenders []string
	seenOwner := false
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("lecture %s: %v", name, err)
		}
		if !slotSetPattern.Match(raw) {
			continue
		}
		if name == publishedSlotsOwner {
			seenOwner = true
			continue
		}
		offenders = append(offenders, name)
	}

	// Le garde-rail doit pouvoir ÉCHOUER : si le motif ne se trouve même plus chez son
	// propriétaire, c'est le test qui ne garde plus rien (leçon J4.0 — un garde qui ne
	// peut pas échouer ne garde rien).
	if !seenOwner {
		t.Fatalf("motif introuvable dans %s : le garde-rail ne vérifie plus rien "+
			"(le helper a-t-il été renommé/déplacé ?)", publishedSlotsOwner)
	}
	if len(offenders) > 0 {
		t.Errorf("ensemble de slots publiés reconstruit hors de %s : %v — "+
			"appeler keepOfPublishedTracks(...) au lieu de recopier la boucle",
			publishedSlotsOwner, offenders)
	}
}

// TestKeepOfPublishedTracks_ContratPreserve fige le contrat commun aux quatre calques :
// entrée vide -> nil, sortie vide -> nil (jamais un tableau vide), filtrage effectif.
func TestKeepOfPublishedTracks_ContratPreserve(t *testing.T) {
	tracks := []Track{{Slot: 512}}
	garde := func(s Shot, published map[uint32]bool) bool { return published[s.Slot] }

	if got := keepOfPublishedTracks(nil, tracks, garde); got != nil {
		t.Errorf("entrée vide : attendu nil, obtenu %v", got)
	}
	if got := keepOfPublishedTracks([]Shot{{Slot: 999}}, tracks, garde); got != nil {
		t.Errorf("tout filtré : attendu nil (et non un tableau vide), obtenu %v", got)
	}
	got := keepOfPublishedTracks([]Shot{{Slot: 512}, {Slot: 999}}, tracks, garde)
	if len(got) != 1 || got[0].Slot != 512 {
		t.Errorf("filtrage : attendu le seul slot publié, obtenu %v", got)
	}
}
