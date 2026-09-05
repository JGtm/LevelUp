package mappings

import (
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/testutil"
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
		// KOTH, mesure du 2026-08-30 : 3 en social (45 matchs sur 46), 4 en classé (3/3).
		// L'ancien motif d'exclusion (« l'oracle du film diffère de l'API ») est périmé —
		// le registre porte des collines depuis le backfill du 2026-08-24, cf. le TOML.
		"KOTH:Arena":              3,
		"Ranked:King of the Hill": 4,
	} {
		if target, ok := hi.ScoreTarget(variant); !ok || target != want {
			t.Errorf("halo_infinite cible %q = (%d, %v), want (%d, true)", variant, target, ok, want)
		}
	}
	// Oddball reste VOLONTAIREMENT absent (mode à manches : le total déborde le plateau
	// d'une manche). "Arena:King of the Hill" aussi, pour une autre raison : son plateau
	// n'est atteint que par UN match, sous la règle des >= 2. Cf. le TOML.
	for _, variant := range []string{"Ranked:Oddball", "Arena:King of the Hill"} {
		if _, ok := hi.ScoreTarget(variant); ok {
			t.Errorf("halo_infinite : %q ne doit pas avoir de cible (cf. commentaire du TOML)", variant)
		}
	}

	// Tics de garde par point : 35, mesure du 2026-08-30 (union des instants, 15 periodes sur 16).
	if secs, ok := hi.HoldTicksPerPoint("KOTH:Arena"); !ok || secs != 35 {
		t.Errorf("halo_infinite tics/point %q = (%d, %v), want (35, true)", "KOTH:Arena", secs, ok)
	}
	// Le KOTH CLASSÉ n'a pas de seuil : ses 3 matchs sont inexploitables (deux sans film en
	// cache, un sur une carte absente du catalogue de bornes). Lui recopier la valeur du
	// social serait la devinette que la table interdit — donc aucune jauge côté client.
	// Strongholds non plus : ses zones simultanées portent leur vraie jauge dans le film.
	for _, variant := range []string{"Ranked:King of the Hill", "Strongholds:Arena", "CTF:Arena"} {
		if _, ok := hi.HoldTicksPerPoint(variant); ok {
			t.Errorf("halo_infinite : %q ne doit pas avoir de tics de garde par point", variant)
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
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	set, err := LoadRegulationFromFile(
		filepath.Join(root, "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
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

// ---------------------------------------------------------------------------------------
// [score_timeline] — comment le score se montre dans le temps (schema_version 5).
// ---------------------------------------------------------------------------------------

// La table est indexée par JETON DE MODE, cherché comme MOT ENTIER dans le `pair_name` BRUT
// (suffixe de carte retiré) — et NON dans le libellé normalisé, qui mange le jeton sur toute
// une famille de pair_name (cf. le commentaire de la table).
func TestLoadRegulation_ScoreTimelineKinds(t *testing.T) {
	t.Parallel()
	set, err := LoadRegulationFromBytes("regulation.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 5

[score_timeline]
"Slayer"           = "hidden"
"CTF"              = "events"
"King of the Hill" = "events"
"Oddball"          = "curve"
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cases := map[string]string{
		// Jeton exact.
		"Slayer": ScoreTimelineHidden,
		"CTF":    ScoreTimelineEvents,
		// Mot ENTIER à l'intérieur du libellé normalisé — le cas nominal des variantes.
		"Team Slayer":      ScoreTimelineHidden,
		"Tactical Slayer":  ScoreTimelineHidden,
		"One Flag CTF":     ScoreTimelineEvents,
		"CTF 3 Captures":   ScoreTimelineEvents,
		"King of the Hill": ScoreTimelineEvents,
		"Oddball":          ScoreTimelineCurve,
		// Insensible à la casse.
		"team slayer": ScoreTimelineHidden,
		// MODE ABSENT DE LA TABLE = REPLI SÛR : la courbe, jamais un bloc effacé.
		"Strongholds": ScoreTimelineCurve,
		"Stockpile":   ScoreTimelineCurve,
		"":            ScoreTimelineCurve,
		// Pas un mot entier : "Slayers" ne doit pas attraper "Slayer".
		"Slayers": ScoreTimelineCurve,
	}
	for label, want := range cases {
		if got := set.ScoreTimelineKind(label); got != want {
			t.Errorf("ScoreTimelineKind(%q) = %q, want %q", label, got, want)
		}
	}
}

// Une valeur hors des trois lectures admises est une ERREUR DE CONFIGURATION au chargement,
// jamais un silence : une faute de frappe tomberait sinon sur le repli `curve` et se lirait
// à l'écran comme une décision produit.
func TestLoadRegulation_ScoreTimelineRejectsUnknownKind(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{`"Slayer" = "event"`, `"Slayer" = ""`, `"Slayer" = "bar"`} {
		_, err := LoadRegulationFromBytes("t.toml", []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 5

[score_timeline]
`+bad+`
`))
		if err == nil {
			t.Errorf("lecture inconnue (%s) : attendu une erreur de chargement", bad)
		}
	}
}

// Table absente (TOML antérieur au schéma 5) ou jeu de règles nil : tout retombe sur la
// courbe — le comportement d'avant la table.
func TestRegulationSet_ScoreTimelineFallsBackToCurve(t *testing.T) {
	t.Parallel()
	set, err := LoadRegulationFromBytes("regulation.toml", []byte(`
[meta]
title_slug     = "halo_5"
schema_version = 1
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := set.ScoreTimelineKind("Slayer"); got != ScoreTimelineCurve {
		t.Errorf("table absente : got %q, want %q", got, ScoreTimelineCurve)
	}
	var nilSet *RegulationSet
	if got := nilSet.ScoreTimelineKind("Slayer"); got != ScoreTimelineCurve {
		t.Errorf("nil : got %q, want %q", got, ScoreTimelineCurve)
	}
}

// Smoke test sur le VRAI fichier du repo, avec les `pair_name` RÉELS du registre local.
//
// LES NEUF FAMILLES QUI RATAIENT SONT ICI EN ASSERTION NOMINATIVE. Elles rendaient toutes
// `curve` tant que l'appariement passait par `NormalizeModeLabel` (460 matchs, dont les 429
// de « Super Fiesta:Slayer », le mode le plus joué du corpus). L'entrée passe désormais par
// le brut, suffixe de carte retiré — c'est ce que ce test verrouille.
func TestScoreTimelineFromRepo(t *testing.T) {
	t.Parallel()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	hi, err := LoadRegulationFromFile(filepath.Join(repoRoot, "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("halo_infinite regulation.toml: %v", err)
	}
	cases := map[string]string{
		// --- LES NEUF FAMILLES DU DÉFAUT, par ordre de volume au registre -----------------
		"Super Fiesta:Slayer":  ScoreTimelineHidden, // 429 matchs — normalisé : « Super Fiesta »
		"Super Husky Raid:CTF": ScoreTimelineEvents, //  11 matchs — « Super Husky Raid »
		"Arena:Team Snipers":   ScoreTimelineHidden, //   8 matchs — « Team Snipers »
		"Husky Raid:CTF":       ScoreTimelineEvents, //   3 matchs — « Husky Raid »
		"Team Slayer:Arena":    ScoreTimelineHidden, //   3 matchs — « Arena »
		"CTF:Arena":            ScoreTimelineEvents, //   2 matchs — « Arena »
		"BTB:Team Snipers":     ScoreTimelineHidden, //   2 matchs — « Team Snipers »
		"Husky Raid:Assault":   ScoreTimelineEvents, //   1 match  — « Husky Raid »
		"Slayer:Arena":         ScoreTimelineHidden, //   1 match  — « Arena »

		// --- Le suffixe de carte est retiré, et lui seul ---------------------------------
		"Super Fiesta:Slayer on Forbidden - Forge": ScoreTimelineHidden,
		"Community:Team Slayer on Starboard":       ScoreTimelineHidden,
		"Husky Raid:CTF on Catalyst":               ScoreTimelineEvents,
		"Arena:Strongholds on Vagabond":            ScoreTimelineCurve,

		// --- Le cas nominal, inchangé ----------------------------------------------------
		"Arena:Slayer":            ScoreTimelineHidden,
		"Arena:CTF":               ScoreTimelineEvents,
		"Arena:Neutral Flag CTF":  ScoreTimelineEvents,
		"CTF:Arena Neutral Flag":  ScoreTimelineEvents,
		"Arena:King of the Hill":  ScoreTimelineEvents,
		"KOTH:Arena":              ScoreTimelineEvents,
		"Assault:One Bomb":        ScoreTimelineEvents,
		"Assault:Neutral Bomb":    ScoreTimelineEvents,
		"Arena:Escalation Slayer": ScoreTimelineHidden,

		// --- VOLONTAIREMENT au repli, faute de décision produit (cf. le TOML) ------------
		"Arena:Strongholds": ScoreTimelineCurve,
		"Arena:Oddball":     ScoreTimelineCurve,
		"BTB:Total Control": ScoreTimelineCurve,
		"BTB:Stockpile":     ScoreTimelineCurve,
		"Arena:Extraction":  ScoreTimelineCurve,
		"Arena:VIP":         ScoreTimelineCurve,
		"Arena:Land Grab":   ScoreTimelineCurve,

		// --- Ce que l'appariement attrape EN PLUS, et qui est sans conséquence -----------
		// Les libellés Firefight tombent en `events`. Ces matchs PvE n'ont pas de calque de
		// score d'ÉQUIPE, donc le graphe ne rend rien de toute façon — c'est écrit ici pour
		// que ce soit une décision assumée et non une surprise.
		"Firefight:Heroic King of the Hill": ScoreTimelineEvents,
		"Gruntpocalypse:Heroic KOTH":        ScoreTimelineEvents,
	}
	for pairName, want := range cases {
		if got := hi.ScoreTimelineKind(pairName); got != want {
			t.Errorf("ScoreTimelineKind(%q) = %q, want %q", pairName, got, want)
		}
	}
}
