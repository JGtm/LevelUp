package killcollector

// bijection_provenance_test.go — LES COMPTEURS DE PROVENANCE DU LIEN `indice -> joueur` (lot 1.8).
//
// Ils existent pour que l exploitation sache quelle part de chaque passe vient d une LECTURE
// (la table des joueurs du film) et quelle part d un REPLI (la bijection inferee des votes du
// kill-feed) — D14 (c) du chantier, D-10 d ADR 0034. Un repli qu on ne compte pas ne se retire
// jamais, faute de savoir s il sert encore.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/observability"
)

// TestProvenanceDeLaBijectionEstPubliee — la passe nominale : ce qui est LU, ce qui est INFERE,
// et le controle par le kill-feed. Les compteurs sont cumulatifs : le test mesure des DELTAS.
func TestProvenanceDeLaBijectionEstPubliee(t *testing.T) {
	avant := lireProvenance()
	publishBijectionProvenance(killsource.FilmTablePinning{
		Build: "HI_1_13_0", Seats: 8, Pinned: 7, Inferred: 1,
		Agree: 5, Contradict: 1, Silent: 1,
	})
	apres := lireProvenance()
	for nom, attendu := range map[string]int64{
		metricBijTableFilm:  7,
		metricBijInference:  1,
		metricBijSilence:    1,
		metricBijContradict: 1,
	} {
		if got := apres[nom] - avant[nom]; got != attendu {
			t.Errorf("%s : delta %d, attendu %d", nom, got, attendu)
		}
	}
}

// TestRefusDeTableCompteSaCAUSE — « la table a ete refusee » sans dire pourquoi n oriente aucun
// diagnostic : la cause entre dans le NOM du compteur, comme `filmdec_unknown_build_<build>`.
func TestRefusDeTableCompteSaCause(t *testing.T) {
	avant := observability.LoadCounter(metricBijTableRefusee + "sans_section")
	publishBijectionProvenance(killsource.FilmTablePinning{
		Refusal: killsource.FilmTableNoSection, Inferred: 24,
	})
	if got := observability.LoadCounter(metricBijTableRefusee+"sans_section") - avant; got != 1 {
		t.Errorf("le refus `sans_section` n est pas compte (delta %d)", got)
	}
}

// TestBuildInconnuPublieSonCompteurNomme — D-4 d ADR 0034 : un film au build hors profil est mis
// de cote AVEC son compteur nomme, jamais lu au profil du build voisin.
//
// AUCUN CORPUS NE PEUT DECLENCHER CE CAS (mesure du lot 1.5 : 0 build inconnu sur les 1 351 films
// du cache), donc seul un test peut prouver que le cablage existe.
func TestBuildInconnuPublieSonCompteurNomme(t *testing.T) {
	const nom = "filmdec_unknown_build_hi_9_99_0"
	avant := observability.LoadCounter(nom)
	publishBijectionProvenance(killsource.FilmTablePinning{
		Refusal: killsource.FilmTableUnknownBuild, Build: "HI_9_99_0", Inferred: 8,
	})
	if got := observability.LoadCounter(nom) - avant; got != 1 {
		t.Errorf("%s : delta %d, attendu 1 — le compteur de build inconnu n est pas cable", nom, got)
	}
	if got := observability.LoadCounter(metricBijTableRefusee + "build_inconnu"); got < 1 {
		t.Error("la cause `build_inconnu` n est pas comptee a cote du build")
	}
}

func lireProvenance() map[string]int64 {
	out := map[string]int64{}
	for _, n := range []string{metricBijTableFilm, metricBijInference, metricBijSilence, metricBijContradict} {
		out[n] = observability.LoadCounter(n)
	}
	return out
}
