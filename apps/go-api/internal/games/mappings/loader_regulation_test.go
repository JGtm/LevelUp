package mappings

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestLoadRegulationTOMLsFromRepo — smoke test sur les VRAIS fichiers du repo :
// halo_infinite doit porter les 9 variantes mesurées à 720 s, halo_5 doit être
// vide (aucun temps réglementaire mesuré → aucun flag « Prolongation »).
func TestLoadRegulationTOMLsFromRepo(t *testing.T) {
	t.Parallel()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")

	hi, err := LoadRegulationFromFile(filepath.Join(repoRoot, "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("halo_infinite regulation.toml: %v", err)
	}
	if n := len(hi.SecondsMap()); n != 9 {
		t.Errorf("halo_infinite : %d variantes, want 9", n)
	}
	for _, variant := range []string{"CTF:Arena", "Slayer:Arena", "Strongholds:Arena", "Arena:Team Slayer"} {
		if secs, ok := hi.Seconds(variant); !ok || secs != 720 {
			t.Errorf("halo_infinite %q = (%d, %v), want (720, true)", variant, secs, ok)
		}
	}

	// Les cibles de victoire mesurées (2026-08-24) : sondage sur les plateaux les plus massifs.
	for variant, want := range map[string]int{
		"Slayer:Arena Super Fiesta": 50,
		"BTB:Slayer":                100,
		"CTF:Arena":                 3,
		"CTF:Arena Neutral Flag":    5,
		"Ranked:Strongholds":        250,
	} {
		if target, ok := hi.ScoreTarget(variant); !ok || target != want {
			t.Errorf("halo_infinite cible %q = (%d, %v), want (%d, true)", variant, target, ok, want)
		}
	}
	// Oddball et KOTH sont VOLONTAIREMENT absents (modes à manches, cf. le TOML).
	for _, variant := range []string{"Ranked:Oddball", "KOTH:Arena"} {
		if _, ok := hi.ScoreTarget(variant); ok {
			t.Errorf("halo_infinite : %q ne doit pas avoir de cible (mode à manches)", variant)
		}
	}

	h5, err := LoadRegulationFromFile(filepath.Join(repoRoot, "config", "titles", "halo_5", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("halo_5 regulation.toml: %v", err)
	}
	if n := len(h5.SecondsMap()); n != 0 {
		t.Errorf("halo_5 : %d variantes, want 0 (aucune mesure)", n)
	}
	if _, ok := h5.ScoreTarget("CTF:Arena"); ok {
		t.Error("halo_5 : aucune cible de victoire mesurée attendue")
	}
}

func TestLoadRegulation_Valid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[regulation_seconds]
"CTF:Arena"         = 720
"Team Slayer:Arena" = 720
`)
	set, err := LoadRegulationFromBytes("regulation.toml", raw)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("title_slug: got %q", set.TitleSlug())
	}
	if set.SchemaVersion() != 1 {
		t.Errorf("schema_version: got %d", set.SchemaVersion())
	}
	if secs, ok := set.Seconds("CTF:Arena"); !ok || secs != 720 {
		t.Errorf("CTF:Arena: got (%d, %v), want (720, true)", secs, ok)
	}
	// Variante inconnue → jamais de flag (dégradation sûre).
	if secs, ok := set.Seconds("BTB:Slayer"); ok || secs != 0 {
		t.Errorf("variante inconnue: got (%d, %v), want (0, false)", secs, ok)
	}
	if n := len(set.SecondsMap()); n != 2 {
		t.Errorf("SecondsMap len: got %d, want 2", n)
	}
}

// Table VIDE = valide (titre sans temps réglementaire mesuré, ex. Halo 5).
func TestLoadRegulation_EmptyTableIsValid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_5"
schema_version = 1

[regulation_seconds]
`)
	set, err := LoadRegulationFromBytes("regulation.toml", raw)
	if err != nil {
		t.Fatalf("une table vide doit être valide : %v", err)
	}
	if n := len(set.SecondsMap()); n != 0 {
		t.Errorf("SecondsMap len: got %d, want 0", n)
	}
	if _, ok := set.Seconds("CTF:Arena"); ok {
		t.Error("aucune variante ne doit être connue sur une table vide")
	}
}

// La section [score_target] est OPTIONNELLE (un TOML antérieur au schéma 2 n'en a pas)
// et se valide comme les secondes : clé non vide, valeur > 0.
func TestLoadRegulation_ScoreTargets(t *testing.T) {
	set, err := LoadRegulationFromBytes("regulation.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 2

[regulation_seconds]
"CTF:Arena" = 720

[score_target]
"CTF:Arena"    = 3
"Slayer:Arena" = 50
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if target, ok := set.ScoreTarget("CTF:Arena"); !ok || target != 3 {
		t.Errorf("CTF:Arena: got (%d, %v), want (3, true)", target, ok)
	}
	if _, ok := set.ScoreTarget("Ranked:Oddball"); ok {
		t.Error("variante inconnue : aucune cible attendue")
	}
	if _, err := LoadRegulationFromBytes("t.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 2
[score_target]
"CTF:Arena" = 0
`)); err == nil {
		t.Error("cible a zero : attendu une erreur")
	}
}

func TestRegulationSet_NilIsSafe(t *testing.T) {
	var set *RegulationSet
	if secs, ok := set.Seconds("CTF:Arena"); ok || secs != 0 {
		t.Errorf("nil Seconds: got (%d, %v), want (0, false)", secs, ok)
	}
	if target, ok := set.ScoreTarget("CTF:Arena"); ok || target != 0 {
		t.Errorf("nil ScoreTarget: got (%d, %v), want (0, false)", target, ok)
	}
	if n := len(set.SecondsMap()); n != 0 {
		t.Errorf("nil SecondsMap must be empty: got %d", n)
	}
	if set.TitleSlug() != "" || set.SchemaVersion() != 0 {
		t.Error("nil accessors must return zero values")
	}
}

func TestLoadRegulation_Invalid(t *testing.T) {
	cases := map[string]string{
		"meta manquant": `
[regulation_seconds]
"CTF:Arena" = 720
`,
		"schema_version zero": `
[meta]
title_slug = "halo_infinite"
[regulation_seconds]
"CTF:Arena" = 720
`,
		"temps negatif": `
[meta]
title_slug     = "halo_infinite"
schema_version = 1
[regulation_seconds]
"CTF:Arena" = -1
`,
		"temps zero": `
[meta]
title_slug     = "halo_infinite"
schema_version = 1
[regulation_seconds]
"CTF:Arena" = 0
`,
	}
	for name, raw := range cases {
		if _, err := LoadRegulationFromBytes("t.toml", []byte(raw)); err == nil {
			t.Errorf("%s: attendu une erreur", name)
		}
	}
}

// La section [rounds_decide] (schéma 3) déclare les variantes dont le RÉSULTAT se lit en
// manches. Elle est optionnelle, ses clés doivent être non vides, et une entrée `false` est
// REFUSÉE : l'absence de clé est déjà le « non », deux façons de dire non se contrediraient
// un jour.
func TestLoadRegulation_RoundsDecide(t *testing.T) {
	set, err := LoadRegulationFromBytes("regulation.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 3

[rounds_decide]
"Arena:Oddball" = true
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !set.RoundsDecide("Arena:Oddball") {
		t.Error("Arena:Oddball doit se lire en manches")
	}
	if set.RoundsDecide("  Arena:Oddball  ") != true {
		t.Error("la clé doit être comparée trimée, comme les deux autres tables")
	}
	if set.RoundsDecide("CTF:Arena") {
		t.Error("variante non déclarée : on garde les points")
	}
	if _, err := LoadRegulationFromBytes("t.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 3
[rounds_decide]
"CTF:Arena" = false
`)); err == nil {
		t.Error("une entrée à false doit être refusée (retirer la ligne)")
	}
}

func TestRegulationSet_NilRoundsDecide(t *testing.T) {
	var set *RegulationSet
	if set.RoundsDecide("Arena:Oddball") {
		t.Error("nil RoundsDecide doit rendre false")
	}
}

// TestRegulationReelle_OddballDeclare épingle le CONTENU livré : les trois variantes Oddball
// mesurées le 2026-08-29 sont déclarées, et le CTF d'arène (deux mi-temps) ne l'est PAS.
// Sans ce test, un nettoyage de config retirerait la table sans que rien ne casse.
func TestRegulationReelle_OddballDeclare(t *testing.T) {
	set, err := LoadRegulationFromFile("../../../../../config/titles/halo_infinite/mappings/regulation.toml")
	if err != nil {
		t.Fatalf("lecture de la config livrée : %v", err)
	}
	for _, v := range []string{"Arena:Oddball", "Ranked:Oddball", "Oddball:Arena"} {
		if !set.RoundsDecide(v) {
			t.Errorf("%q doit être déclarée dans [rounds_decide] (mesure du 2026-08-29)", v)
		}
	}
	for _, v := range []string{"CTF:Arena", "Ranked:CTF", "Arena:One Flag CTF", "Slayer:Arena"} {
		if set.RoundsDecide(v) {
			t.Errorf("%q ne doit PAS être déclarée : son score est déjà le bon (cf. rapport §2.1)", v)
		}
	}
}
