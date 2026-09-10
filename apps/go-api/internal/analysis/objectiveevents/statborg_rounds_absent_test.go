package objectiveevents

// statborg_rounds_absent_test.go — GARDE ANTI-MANCHE-FANTOME n 5 : UNE MANCHE TOLEREE DANS LE
// TROU DOIT EXISTER (lot 6.7-B1, item 4).
//
// PREUVE D'ENTREE (audit du 2026-09-10 §3.3 et §4.1). `e60aaf06` est un `Strongholds:Arena` de
// 415 s au score 200-82 — un mode a points, UNE manche — et son artefact declarait
// `coverage.score.rounds = 3` alors que les dix series de son fil de score ne portent que la
// manche 0. Il perdait 130 de ses 154 actions en `noSlot` et publiait 10 captures de zone pour
// 38 a l'oracle API.
//
// CAUSE MESUREE SUR LE FILM (releve du 2026-09-11, 864 enregistrements) : le film declare les
// manches 0 et 2, et RIEN en manche 1 — zero enregistrement, ni de joueur ni d'equipe. La
// manche 2 porte 44 enregistrements de joueur (14 % de la manche 0, donc MATERIELLE au sens de
// [statMinRoundRecordShare]) et son intervalle [79 010, 172 093] tombe ENTIEREMENT DANS celui de
// la manche 0 [2 991, 411 036]. La tolerance d'UNE manche vide dans la chaine
// ([statMaxEmptyRoundRun]) — ecrite pour une manche JOUEE mais trop courte pour tirer une suite
// coherente — laissait la chaine sauter par-dessus le vide et atteindre cet ancrage.
//
// LE COUT NE PASSAIT PAS PAR LES SERIES mais par L'IDENTITE : les points de la manche 2 sont
// deja ecartes par [ChronologicalTotal] (ils sont anterieurs a la fin de la manche 0). Ce qui
// coutait, c'est que [SlotIdentityByRound] tire ses manches de [RealRounds] : avec trois manches
// il resolvait l'identite PAR MANCHE, la manche 2 n'ayant aucun slot emetteur du compteur de
// morts, et [RoundIdentity.At] envoyait dans cette manche vide tout evenement date apres
// 79 076 ms — soit la quasi-totalite du match.

import "testing"

// TestRealRoundsRefuseUneMancheDerriereUnTrouVide — un trou VIDE rompt la chaine.
func TestRealRoundsRefuseUneMancheDerriereUnTrouVide(t *testing.T) {
	var recs []StatRecord
	recs = append(recs, joueurSerie(0, 1_000, 300, 200)...)
	// Pas de manche 1 : aucun enregistrement, d'aucune sorte.
	recs = append(recs, joueurSerie(2, 80_000, 44, 0)...)

	real := RealRounds(recs)
	if real[1] || real[2] {
		t.Errorf("manches retenues %v : une manche derriere un trou VIDE est un ancrage, "+
			"pas une manche", real)
	}
	if len(real) != 1 || !real[0] {
		t.Errorf("attendu la seule manche 0, obtenu %v", real)
	}
}

// TestRealRoundsGardeUneMancheCourteQuiEXISTE — LE TEMOIN, et il est la mutation : la meme forme
// avec une manche 1 qui EXISTE (trois enregistrements, trop peu pour etre materielle et trop peu
// pour une suite coherente) garde la chaine ouverte. Si la garde regardait autre chose que la
// PRESENCE — la matiere, la coherence — ce test rougirait.
func TestRealRoundsGardeUneMancheCourteQuiEXISTE(t *testing.T) {
	var recs []StatRecord
	recs = append(recs, joueurSerie(0, 1_000, 300, 200)...)
	recs = append(recs, joueurSerie(1, 60_000, 3, 0)...)
	recs = append(recs, joueurSerie(2, 80_000, 44, 0)...)

	real := RealRounds(recs)
	for _, r := range []int{0, 1, 2} {
		if !real[r] {
			t.Errorf("manche %d perdue alors que la manche 1 EXISTE : %v", r, real)
		}
	}
}

// TestRealRoundsTolereUneManche0AbsenteEnTeteDeChaine — LA CONTRE-EPREUVE. Un film qui numerote
// ses manches A PARTIR DE 1 ne declare RIEN en manche 0 : ce n'est pas un trou, c'est un
// decalage de numerotation, et la chaine doit rester ouverte. Sans cette reserve, la garde
// ci-dessus ferait perdre toutes leurs series aux films de cette forme.
func TestRealRoundsTolereUneManche0AbsenteEnTeteDeChaine(t *testing.T) {
	real := RealRounds(modeSerie(6, 1, 1_000, 1, 2, 3))
	if !real[1] {
		t.Errorf("manche 1 perdue alors qu'aucune manche n'avait encore ete admise : %v", real)
	}
}
