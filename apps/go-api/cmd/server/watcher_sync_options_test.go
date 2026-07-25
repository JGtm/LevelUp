package main

// watcher_sync_options_test.go — GARDE-RAIL durable du chemin LIVE (watcher).
//
// Invariant protégé : les options de sync du watcher DÉRIVENT de
// domain.DefaultSyncOptions() et n'en divergent QUE sur des réglages de débit
// explicitement allowlistés. Tout flag d'extraction (présent ou futur) est donc
// hérité automatiquement.
//
// Ce que ça empêche de se reproduire : WithHighlightEvents (2026-06-04) puis
// WithObjectiveStats (contre-revue V7.2, 2026-07-25) — deux flags oubliés dans
// un literal indépendant, sans rattrapage possible (le delta du scheduler
// s'arrête au premier match déjà connu, le bit n'est jamais reposé).

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// watcherSyncOptionOverrides : les SEULS champs de domain.SyncOptions que le
// watcher a le droit de faire diverger de DefaultSyncOptions(). Ce sont des
// réglages de DÉBIT, jamais des flags d'extraction. Ajouter une entrée ici =
// décision explicite à justifier (cf. watcher_sync_options.go).
var watcherSyncOptionOverrides = map[string]string{
	"MaxMatches":        "fenêtre live courte (session en cours, pas de rattrapage d'historique)",
	"RequestsPerSecond": "quota Microsoft par-token partagé avec le scheduler (anti-429)",
}

// TestWatcherSyncOptions_InheritsExtractionFlags compare champ par champ les
// options du watcher à DefaultSyncOptions(). Tout champ hors allowlist DOIT être
// identique — c'est ce qui rend impossible l'oubli silencieux d'un futur flag.
func TestWatcherSyncOptions_InheritsExtractionFlags(t *testing.T) {
	def := reflect.ValueOf(domain.DefaultSyncOptions())
	got := reflect.ValueOf(watcherSyncOptions())
	typ := def.Type()

	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		defV := def.Field(i).Interface()
		gotV := got.Field(i).Interface()
		reason, allowed := watcherSyncOptionOverrides[name]

		if !allowed {
			if !reflect.DeepEqual(defV, gotV) {
				t.Errorf("SyncOptions.%s diverge du défaut (watcher=%v, DefaultSyncOptions=%v) : "+
					"le watcher DOIT hériter ce champ. Si la divergence est voulue, l'ajouter à "+
					"watcherSyncOptionOverrides avec sa justification", name, gotV, defV)
			}
			continue
		}
		// L'allowlist doit rester HONNÊTE : un champ allowlisté qui ne diverge
		// plus est une entrée périmée à retirer (sinon elle masquerait un futur
		// oubli sur ce champ).
		if reflect.DeepEqual(defV, gotV) {
			t.Errorf("SyncOptions.%s est allowlisté (%s) mais ne diverge plus du défaut (%v) : "+
				"retirer l'entrée de watcherSyncOptionOverrides", name, reason, gotV)
		}
	}
}

// TestWatcherSyncOptions_ExtractionFlagsOn verrouille nommément les deux flags
// dont l'oubli a causé un incident, et valide les options (fail-fast engine).
func TestWatcherSyncOptions_ExtractionFlagsOn(t *testing.T) {
	opts := watcherSyncOptions()
	if !opts.WithObjectiveStats {
		t.Error("WithObjectiveStats=false : les matchs du watcher n'auraient AUCUNE stat objectif " +
			"(CTF/Zones/Oddball) et le delta scheduler ne rattrape jamais — incident contre-revue V7.2")
	}
	if !opts.WithHighlightEvents {
		t.Error("WithHighlightEvents=false : highlight_events → killer_victim_pairs → weapon_kills " +
			"resteraient vides — incident 2026-06-04")
	}
	if !opts.WithParticipants || !opts.WithMedals {
		t.Errorf("WithParticipants=%v WithMedals=%v : attendus true (hérités du défaut)",
			opts.WithParticipants, opts.WithMedals)
	}
	if err := opts.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

// syncOptionsLiteralRE détecte la construction d'un literal composite
// domain.SyncOptions{. PAS de `\s*` avant l'accolade : cela matcherait aussi la
// signature `func watcherSyncOptions() domain.SyncOptions {` (gofmt impose une
// espace avant le corps d'une fonction et AUCUNE avant l'accolade d'un literal —
// la distinction est donc fiable sur du code gofmt-clean, ce que la CI garantit).
var syncOptionsLiteralRE = regexp.MustCompile(`domain\.SyncOptions\{`)

// TestWatcherSyncOptions_NoIndependentLiteral : garde-rail de FORME. Aucun fichier
// du binaire serveur ne doit reconstruire un literal domain.SyncOptions — c'est
// exactement le pattern qui a produit les deux oublis. Le seul point de
// construction autorisé est watcherSyncOptions() (dérivation).
func TestWatcherSyncOptions_NoIndependentLiteral(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du répertoire package: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("lecture %s: %v", name, rerr)
		}
		if syncOptionsLiteralRE.Match(src) {
			t.Errorf("%s construit un literal domain.SyncOptions : utiliser watcherSyncOptions() "+
				"(dérivation de DefaultSyncOptions) — un literal indépendant perd les futurs flags d'extraction", name)
		}
	}
}
