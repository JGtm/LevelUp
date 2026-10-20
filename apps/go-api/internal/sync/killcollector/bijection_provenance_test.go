package killcollector

// bijection_provenance_test.go — LES COMPTEURS DE PROVENANCE DU LIEN `indice -> joueur` (lot 1.8).
//
// Ils existent pour que l exploitation sache quelle part de chaque passe vient d une LECTURE
// (la table des joueurs du film) et quelle part d un REPLI (la bijection inferee des votes du
// kill-feed) — D14 (c) du chantier, D-10 d ADR 0034. Un repli qu on ne compte pas ne se retire
// jamais, faute de savoir s il sert encore.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
)

// TestProvenanceDeLaBijectionEstPubliee — la passe nominale : ce qui est LU, ce qui est INFERE,
// et le controle par le kill-feed. Les compteurs sont cumulatifs : le test mesure des DELTAS.
func TestProvenanceDeLaBijectionEstPubliee(t *testing.T) {
	avant := lireProvenance()
	publishBijectionProvenance(decfilm.FilmTablePinning{
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
	publishBijectionProvenance(decfilm.FilmTablePinning{
		Refusal: decfilm.FilmTableNoSection, Inferred: 24,
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
	publishBijectionProvenance(decfilm.FilmTablePinning{
		Refusal: decfilm.FilmTableUnknownBuild, Build: "HI_9_99_0", Inferred: 8,
	})
	if got := observability.LoadCounter(nom) - avant; got != 1 {
		t.Errorf("%s : delta %d, attendu 1 — le compteur de build inconnu n est pas cable", nom, got)
	}
	if got := observability.LoadCounter(metricBijTableRefusee + "build_inconnu"); got < 1 {
		t.Error("la cause `build_inconnu` n est pas comptee a cote du build")
	}
}

// TestAmbiguiteNeCompteQueLesFilmsQuiBASCULENT — LES TROIS REGIMES DE L AFFECTATION.
//
// `killsource_bijection_noms_libres_en_trop` pretend mesurer « la population qui perd la
// publication ligne par ligne avec le critere corrige ». Cette population est exactement celle
// que `FilmTablePinning.AffectationUnique` fait passer de VRAI a FAUX face a l ancien
// `Inferred <= 1` : UN indice libre pour AU MOINS DEUX noms libres.
//
// IL SURCOMPTAIT JUSQU AU 2026-09-16 (revue de jalon M1, ronde 2, constat F3) : la condition
// `Inferred > 0 && FreeNames > Inferred` mordait aussi sur `Inferred >= 2`, ou l ancienne porte
// refusait DEJA — donc ou rien ne bascule.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : remettre `t.Inferred > 0 && t.FreeNames > t.Inferred`
// dans `publishBijectionProvenance` — le regime « 2 indices libres / 3 noms » compte alors 1
// au lieu de 0. Jouee et restauree par NOM le 2026-09-16.
func TestAmbiguiteNeCompteQueLesFilmsQuiBascule(t *testing.T) {
	for _, cas := range []struct {
		nom     string
		pinning decfilm.FilmTablePinning
		attendu int64
	}{
		{
			// AUCUN NOM LIBRE EN TROP : un indice a inferer, un seul nom pour lui.
			// L affectation est FORCEE — les deux portes publient, rien ne bascule.
			nom:     "1 indice libre / 1 nom libre",
			pinning: decfilm.FilmTablePinning{Seats: 8, Pinned: 7, Inferred: 1, FreeNames: 1},
			attendu: 0,
		},
		{
			// LA POPULATION QUI BASCULE : un indice pour deux noms. L ancienne porte
			// (`Inferred <= 1`) publiait un occupant tire au sort ; la corrigee refuse.
			nom:     "1 indice libre / 2 noms libres",
			pinning: decfilm.FilmTablePinning{Seats: 8, Pinned: 7, Inferred: 1, FreeNames: 2},
			attendu: 1,
		},
		{
			// LE REGIME QUI SURCOMPTAIT : deux indices libres. `Inferred <= 1` etait DEJA
			// faux, donc l ancienne porte refusait deja — aucune publication n est perdue,
			// et ce compteur-ci ne doit pas bouger. C est `killsource_bijection_inference`
			// qui porte ces indices devines.
			nom:     "2 indices libres / 3 noms libres",
			pinning: decfilm.FilmTablePinning{Seats: 8, Pinned: 6, Inferred: 2, FreeNames: 3},
			attendu: 0,
		},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			// CONTROLE CROISE : « basculer » se definit par les DEUX portes — l ancienne
			// publiait (`Inferred <= 1`) et la corrigee refuse (`!AffectationUnique()`).
			// Sans lui, le tableau des cas pourrait deriver de ce qu il pretend couvrir.
			bascule := cas.pinning.Inferred <= 1 && !cas.pinning.AffectationUnique()
			if bascule != (cas.attendu == 1) {
				t.Fatalf("le cas est mal pose : ancienne porte %v, AffectationUnique %v, "+
					"compteur attendu %d", cas.pinning.Inferred <= 1,
					cas.pinning.AffectationUnique(), cas.attendu)
			}
			avant := observability.LoadCounter(metricBijAmbigue)
			publishBijectionProvenance(cas.pinning)
			if got := observability.LoadCounter(metricBijAmbigue) - avant; got != cas.attendu {
				t.Errorf("%s : delta %d, attendu %d — le compteur ne mesure pas la population "+
					"qui perd la publication ligne par ligne", metricBijAmbigue, got, cas.attendu)
			}
		})
	}
}

func lireProvenance() map[string]int64 {
	out := map[string]int64{}
	for _, n := range []string{metricBijTableFilm, metricBijInference, metricBijSilence, metricBijContradict} {
		out[n] = observability.LoadCounter(n)
	}
	return out
}
