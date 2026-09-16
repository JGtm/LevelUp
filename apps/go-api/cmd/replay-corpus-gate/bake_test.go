package main

// bake_test.go — LA LIGNE DE COMMANDE DE L'ENFANT `replay-build` (D6 (1.9.9), 2026-09-17).
//
// LE FAIT MESURE : `bake.go` construisait les arguments EN DUR (`--map`, `--title`, `--facts`,
// matchID). Aucun plafond memoire n'etait transmis, l'enfant gardait donc toujours
// `filmproc.DefaultLimitGiB` = 3 Gio souple / 3,75 Gio dur — et les deux plus gros temoins BTB
// ont ete mesures juste au-dessus : `e5adf7b2` a 3,779 Gio, `4f77afc1` a 3,807 Gio COTE BASE.
// Ils tombaient donc au hasard de la pression memoire, d'un run a l'autre.
//
// LA MUTATION QUI FAIT ROUGIR CE FICHIER, NOMMEE : retirer `"--mem-gib", strconv.Itoa(p.MemGiB)`
// de `argsCuisson` — le plafond n'est plus transmis et l'enfant reprend celui de production.

import (
	"strconv"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// TestArgsCuissonTransmetLePlafondMemoire — LE COMPORTEMENT DEMANDE : `--mem-gib` figure dans
// la ligne construite, avec la valeur du gate, ET les quatre arguments d'origine restent.
func TestArgsCuissonTransmetLePlafondMemoire(t *testing.T) {
	p := cuissonParams{TitleSlug: "halo_infinite", MemGiB: 7}
	got := argsCuisson(p, "lc_stacks", "/tmp/facts_e5adf7b2.json", "e5adf7b2-...-@1")

	attendu := []string{
		"--map", "lc_stacks",
		"--title", "halo_infinite",
		"--facts", "/tmp/facts_e5adf7b2.json",
		"--mem-gib", "7",
		"e5adf7b2-...-@1",
	}
	if len(got) != len(attendu) {
		t.Fatalf("%d argument(s), %d attendus :\n  obtenu %q\n  attendu %q",
			len(got), len(attendu), got, attendu)
	}
	for i := range attendu {
		if got[i] != attendu[i] {
			t.Errorf("argument %d = %q, %q attendu (ligne complete : %q)", i, got[i], attendu[i], got)
		}
	}
	if got[len(got)-1] != "e5adf7b2-...-@1" {
		t.Errorf("le matchID doit rester le DERNIER argument positionnel : %q", got)
	}
}

// TestArgsCuissonTransmetLeDesarmement — `--mem-gib 0` est l'echappatoire documentee de
// l'operateur : elle doit arriver TELLE QUELLE a l'enfant, pas etre remplacee par un defaut.
func TestArgsCuissonTransmetLeDesarmement(t *testing.T) {
	got := argsCuisson(cuissonParams{TitleSlug: "halo_infinite", MemGiB: 0}, "m", "f", "id")
	var vu bool
	for i, a := range got {
		if a == "--mem-gib" && i+1 < len(got) {
			vu = true
			if got[i+1] != "0" {
				t.Errorf("--mem-gib = %q, \"0\" attendu (desarmement demande)", got[i+1])
			}
		}
	}
	if !vu {
		t.Fatalf("--mem-gib absent de la ligne : %q", got)
	}
}

// TestPlafondMemoireGateCouvreLesDeuxTemoinsBTB — LE CHIFFRE, PAS L'INTENTION. Le defaut du
// gate doit couvrir les deux pics MESURES (3,779 et 3,807 Gio cote base) une fois la marge dure
// de +25 % appliquee, et rester STRICTEMENT au-dessus du defaut de production — sans quoi D6 se
// reproduit au prochain run.
func TestPlafondMemoireGateCouvreLesDeuxTemoinsBTB(t *testing.T) {
	const picMesureMax = 3.807 // Gio, `4f77afc1` cote base, 2026-09-16
	if plafondMemoireGate <= filmproc.DefaultLimitGiB {
		t.Fatalf("plafondMemoireGate = %d, strictement au-dessus de filmproc.DefaultLimitGiB (%d) "+
			"attendu : sinon les deux temoins BTB retombent au hasard de la pression memoire",
			plafondMemoireGate, filmproc.DefaultLimitGiB)
	}
	dur := float64(filmproc.HardLimitFor(plafondMemoireGate)) / (1 << 30)
	if dur <= picMesureMax {
		t.Errorf("plafond DUR = %.3f Gio pour un souple de %d, au-dessus de %.3f Gio attendu "+
			"(pic mesure de `4f77afc1` cote base)", dur, plafondMemoireGate, picMesureMax)
	}
}

// TestArgsCuissonNePasseJamaisLeDossierDeChunks — `stageFilm` a deja place les chunks au chemin
// que l'enfant deduit de LEVELUP_REPO_ROOT. Le repeter ici serait une deuxieme ecriture de la
// regle de disposition, qui divergerait de `filmcache` au premier changement.
func TestArgsCuissonNePasseJamaisLeDossierDeChunks(t *testing.T) {
	got := argsCuisson(cuissonParams{TitleSlug: "halo_infinite", MemGiB: plafondMemoireGate}, "m", "f", "id")
	for _, a := range got {
		if a == "--chunks" || a == "--chunk-dir" {
			t.Fatalf("la ligne passe un dossier de chunks (%q) : %q", a, got)
		}
	}
	if got[len(got)-2] != strconv.Itoa(plafondMemoireGate) {
		t.Errorf("l'avant-dernier argument doit etre la valeur de --mem-gib : %q", got)
	}
}
