package replay

// coverage_registre_test.go — LE STATUT DU REGISTRE PUBLIE PAR L ARTEFACT, MESURE SUR LES SEPT
// MINI-BOBINES COMMISES (lot 3.1.1-b).
//
// # CE QUE CE TEST DIT, ET CE QU IL NE REMPLACE PAS
//
// Il lit `chunk_00` des sept bobines — un par build du cache — et classe leur empreinte de
// registre par la porte de publication ([statutDuRegistre]). AUCUN FILM N EST DECODE : la
// grammaire des composants s identifie sur le registre seul.
//
// C est la forme UNITAIRE du gain que le corpus gate doit trouver : avant ce lot, CINQ de ces
// sept bobines sortaient `inconnue` (le decodeur ne connaissait qu une empreinte, celle du build
// de reference) ; elles sortent `connue`. Le corpus gate reste le juge — ce test dit ce qu il
// doit dire.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestLesSeptBobinesClassentConnue : les sept builds du cache portent une empreinte que la table
// du profil ATTEND pour leur clef.
//
// Mutation qui doit le faire rougir : retirer une ligne de `profile.EmpreintesRegistre()` — la
// bobine de cette clef repasse `presumee` (son empreinte reste au catalogue sous la clef jumelle)
// ou `inconnue`.
func TestLesSeptBobinesClassentConnue(t *testing.T) {
	for _, b := range miniFilmBuilds() {
		t.Run(b.Build, func(t *testing.T) {
			chunk0 := chunk00DeLaBobine(t, b)
			reg, err := grammar.ParseRegistryChunk(chunk0)
			if err != nil {
				t.Fatalf("registre de %s : %v", b.Short8, err)
			}
			fp := grammar.RegistryFingerprint(reg)
			if got := statutDuRegistre(b.Build, fp); got != RegistryStatutConnue {
				t.Errorf("%s (%s) : empreinte 0x%016x classee %q, attendu %q",
					b.Short8, b.Build, fp, got, RegistryStatutConnue)
			}
		})
	}
}

// TestLeStatutDuRegistreRendLesTroisChaines : la porte de publication rend bien les TROIS
// chaines, et ce sont celles de la couche profil — pas une seconde liste de litteraux.
func TestLeStatutDuRegistreRendLesTroisChaines(t *testing.T) {
	const jamaisVue = 0x8d6dec5f4fc182c1 // `58e6f72a`, build `HI_1_5_1` (recensement du 2026-09-17)
	reference := chunk00DeLaBobine(t, miniFilmBuilds()[len(miniFilmBuilds())-1])
	reg, err := grammar.ParseRegistryChunk(reference)
	if err != nil {
		t.Fatalf("registre de reference : %v", err)
	}
	fpRef := grammar.RegistryFingerprint(reg)

	cas := []struct {
		nom     string
		build   string
		fp      uint64
		attendu string
	}{
		{"clef et empreinte accordees", "HI_1_13_0", fpRef, RegistryStatutConnue},
		{"empreinte du catalogue, clef inconnue", "HI_9_99_0", fpRef, RegistryStatutPresumee},
		{"empreinte jamais vue", "HI_1_13_0", jamaisVue, RegistryStatutInconnue},
	}
	for _, c := range cas {
		if got := statutDuRegistre(c.build, c.fp); got != c.attendu {
			t.Errorf("%s : statut %q, attendu %q", c.nom, got, c.attendu)
		}
	}
	// LES TROIS CHAINES SONT CELLES DE LA COUCHE PROFIL : une recopie ici divergerait au premier
	// statut ajoute, et le contrat de l artefact (schema 61) est fait de ces chaines-la.
	if RegistryStatutConnue != string(profile.StatutRegistreConnue) ||
		RegistryStatutPresumee != string(profile.StatutRegistrePresumee) ||
		RegistryStatutInconnue != string(profile.StatutRegistreInconnue) {
		t.Fatal("les statuts publies ont diverge de ceux de la couche profil")
	}
}
