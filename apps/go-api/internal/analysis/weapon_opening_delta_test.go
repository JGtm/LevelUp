package analysis

import (
	"math"
	"testing"
)

// mk : un frag mesuré, forme courte pour les fixtures de ce fichier.
func mk(matchID, killer string, timeMS int64, side Side, dist float64) MeasuredKill {
	return MeasuredKill{
		MatchID: matchID, KillerXUID: killer, TimeMS: timeMS,
		WeaponKey: "hinf_br75", Side: side, DistanceM: dist,
	}
}

const epsDelta = 1e-9

// TestWeaponOpeningDelta_ApparieParFragEtPasEntreMedianes — LE test du fichier, et il est
// construit pour que l'erreur qu'il interdit soit VISIBLE : les médianes des deux populations
// sont ÉGALES (10 m des deux côtés), alors que le delta apparié vaut -5 m. Un calcul « médiane
// des entames − médiane des coups fatals » rendrait 0 et raconterait que le joueur ne ferme
// jamais la distance.
func TestWeaponOpeningDelta_ApparieParFragEtPasEntreMedianes(t *testing.T) {
	kills := []MeasuredKill{
		mk("m1", "k", 1000, SideKiller, 5),  // apparié : entame 10 -> -5
		mk("m1", "k", 2000, SideKiller, 10), // apparié : entame 15 -> -5
		mk("m1", "k", 3000, SideKiller, 15), // apparié : entame 20 -> -5
		mk("m1", "k", 4000, SideKiller, 20), // aucune entame : hors appariement
		mk("m1", "k", 5000, SideKiller, 30), // idem — la médiane des coups fatals vaut alors 15
	}
	openings := []MeasuredKill{
		mk("m1", "k", 1000, SideKiller, 10),
		mk("m1", "k", 2000, SideKiller, 15),
		mk("m1", "k", 3000, SideKiller, 20),
	}

	st := WeaponOpeningDelta(kills, openings, SideKiller)
	if st.MeasuredOpenings != 3 || st.Paired != 3 {
		t.Fatalf("MeasuredOpenings=%d Paired=%d, attendu 3 et 3", st.MeasuredOpenings, st.Paired)
	}
	if math.Abs(st.MedianOpeningM-15) > epsDelta {
		t.Errorf("MedianOpeningM = %v, attendu 15", st.MedianOpeningM)
	}
	if math.Abs(st.MedianDeltaM-(-5)) > epsDelta {
		t.Errorf("MedianDeltaM = %v, attendu -5 (l'engagement se ferme de 5 m)", st.MedianDeltaM)
	}
	if math.Abs(st.ClosingShare-1) > epsDelta {
		t.Errorf("ClosingShare = %v, attendu 1 (les trois frags se ferment)", st.ClosingShare)
	}
	// Le témoin : la soustraction de médianes que ce fichier existe pour interdire.
	medianeKills := MedianFloat([]float64{5, 10, 15, 20, 30})
	if math.Abs(medianeKills-st.MedianOpeningM) > epsDelta {
		t.Fatalf("fixture cassée : les deux médianes doivent être ÉGALES (%v vs %v) pour que le "+
			"test distingue l'appariement d'une soustraction de médianes",
			medianeKills, st.MedianOpeningM)
	}
}

// TestWeaponOpeningDelta_EntameSansCoupFatalNEstPasAppariee — une entame dont le coup fatal
// n'est pas mesuré compte dans la couverture, jamais dans le delta. La confusion des deux
// dénominateurs présenterait un écart calculé sur une poignée de frags comme s'il en décrivait
// des centaines.
func TestWeaponOpeningDelta_EntameSansCoupFatalNEstPasAppariee(t *testing.T) {
	kills := []MeasuredKill{mk("m1", "k", 1000, SideKiller, 4)}
	openings := []MeasuredKill{
		mk("m1", "k", 1000, SideKiller, 10),
		mk("m1", "k", 9999, SideKiller, 40), // aucun coup fatal mesuré à cet instant
	}

	st := WeaponOpeningDelta(kills, openings, SideKiller)
	if st.MeasuredOpenings != 2 {
		t.Errorf("MeasuredOpenings = %d, attendu 2", st.MeasuredOpenings)
	}
	if st.Paired != 1 {
		t.Errorf("Paired = %d, attendu 1", st.Paired)
	}
	if math.Abs(st.MedianDeltaM-(-6)) > epsDelta {
		t.Errorf("MedianDeltaM = %v, attendu -6", st.MedianDeltaM)
	}
}

// TestWeaponOpeningDelta_LeCoteFiltre — les morts ne se moyennent pas avec les frags. Le même
// instant existe des deux côtés dans cette fixture (deux matchs distincts), et seul le côté
// demandé doit peser.
func TestWeaponOpeningDelta_LeCoteFiltre(t *testing.T) {
	kills := []MeasuredKill{
		mk("m1", "moi", 1000, SideKiller, 5),
		mk("m2", "adverse", 1000, SideVictim, 50),
	}
	openings := []MeasuredKill{
		mk("m1", "moi", 1000, SideKiller, 10),
		mk("m2", "adverse", 1000, SideVictim, 60),
	}

	frags := WeaponOpeningDelta(kills, openings, SideKiller)
	if frags.Paired != 1 || math.Abs(frags.MedianOpeningM-10) > epsDelta {
		t.Errorf("côté tueur : Paired=%d MedianOpeningM=%v, attendu 1 et 10",
			frags.Paired, frags.MedianOpeningM)
	}
	morts := WeaponOpeningDelta(kills, openings, SideVictim)
	if morts.Paired != 1 || math.Abs(morts.MedianOpeningM-60) > epsDelta {
		t.Errorf("côté victime : Paired=%d MedianOpeningM=%v, attendu 1 et 60",
			morts.Paired, morts.MedianOpeningM)
	}
}

// TestWeaponOpeningDelta_PartDeFermetureAuxBornes — la part se compte sur les frags APPARIÉS,
// et un delta EXACTEMENT nul n'est pas une fermeture (la distance n'a pas bougé).
func TestWeaponOpeningDelta_PartDeFermetureAuxBornes(t *testing.T) {
	kills := []MeasuredKill{
		mk("m1", "k", 1, SideKiller, 5),  // -5 : ferme
		mk("m1", "k", 2, SideKiller, 10), // 0  : ne ferme pas
		mk("m1", "k", 3, SideKiller, 15), // +5 : s'ouvre
		mk("m1", "k", 4, SideKiller, 20), // +10 : s'ouvre
	}
	openings := []MeasuredKill{
		mk("m1", "k", 1, SideKiller, 10),
		mk("m1", "k", 2, SideKiller, 10),
		mk("m1", "k", 3, SideKiller, 10),
		mk("m1", "k", 4, SideKiller, 10),
	}

	st := WeaponOpeningDelta(kills, openings, SideKiller)
	if st.Paired != 4 {
		t.Fatalf("Paired = %d, attendu 4", st.Paired)
	}
	if math.Abs(st.ClosingShare-0.25) > epsDelta {
		t.Errorf("ClosingShare = %v, attendu 0.25 (un seul delta strictement négatif)", st.ClosingShare)
	}
	if math.Abs(st.MedianDeltaM-2.5) > epsDelta {
		t.Errorf("MedianDeltaM = %v, attendu 2.5 (médiane de -5, 0, 5, 10)", st.MedianDeltaM)
	}
}

// TestWeaponOpeningDelta_AucuneEntame — le cas NOMINAL tant que le backfill de kill_openings
// n'a pas tourné. Tout est à zéro, et c'est à l'appelant de ne rien publier plutôt que
// d'afficher « 0,0 m » (D5).
func TestWeaponOpeningDelta_AucuneEntame(t *testing.T) {
	st := WeaponOpeningDelta([]MeasuredKill{mk("m1", "k", 1, SideKiller, 5)}, nil, SideKiller)
	if st != (WeaponOpeningStats{}) {
		t.Errorf("stats = %+v, attendu la valeur zéro", st)
	}
}

// TestWeaponOpeningDelta_NeMutePasLesEntrees — les tranches d'entrée sont partagées avec
// l'agrégat par arme, qui les relit après. Un tri en place ici réordonnerait ses groupes.
func TestWeaponOpeningDelta_NeMutePasLesEntrees(t *testing.T) {
	kills := []MeasuredKill{
		mk("m1", "k", 1, SideKiller, 30),
		mk("m1", "k", 2, SideKiller, 5),
	}
	openings := []MeasuredKill{
		mk("m1", "k", 1, SideKiller, 40),
		mk("m1", "k", 2, SideKiller, 3),
	}
	WeaponOpeningDelta(kills, openings, SideKiller)
	if kills[0].DistanceM != 30 || openings[0].DistanceM != 40 {
		t.Errorf("les entrées ont été réordonnées : kills[0]=%v openings[0]=%v",
			kills[0].DistanceM, openings[0].DistanceM)
	}
}

// TestWeaponRangeSideTotals_IgnoreLeSeuilEtLAutreCote — les deux nombres de tête décrivent le
// JOUEUR : ils comptent TOUS les frags mesurés du côté demandé, y compris ceux d'armes sous le
// seuil de publication (ici « hinf_hydra », un seul frag), et jamais ceux de l'autre côté.
func TestWeaponRangeSideTotals_IgnoreLeSeuilEtLAutreCote(t *testing.T) {
	kills := []MeasuredKill{
		mk("m1", "k", 1, SideKiller, 4),
		mk("m1", "k", 2, SideKiller, 8),
		{MatchID: "m1", KillerXUID: "k", TimeMS: 3, WeaponKey: "hinf_hydra",
			Side: SideKiller, DistanceM: 30},
		mk("m2", "adv", 4, SideVictim, 100),
	}

	med, n := WeaponRangeSideTotals(kills, SideKiller)
	if n != 3 {
		t.Fatalf("measured = %d, attendu 3 (l'arme sous le seuil compte ici)", n)
	}
	if math.Abs(med-8) > epsDelta {
		t.Errorf("médiane = %v, attendu 8 (médiane de 4, 8, 30)", med)
	}
	if _, nv := WeaponRangeSideTotals(kills, SideVictim); nv != 1 {
		t.Errorf("côté victime : measured = %d, attendu 1", nv)
	}
	if _, nz := WeaponRangeSideTotals(nil, SideKiller); nz != 0 {
		t.Errorf("entrée vide : measured = %d, attendu 0", nz)
	}
}

// TestWeaponOpeningDelta_LeTueurFaitPartieDeLaCle — la TROISIÈME composante de la clé
// d'appariement (constat F7, revue adversariale du lot 4, 2026-09-06 : retirer `KillerXUID`
// de `measuredKillKey` laissait toute la suite verte).
//
// LE CAS EST ATTEIGNABLE, ET IL N'EST PAS EXOTIQUE : `port.WeaponRangeFilters` porte une
// LISTE de xuids, donc un scope peut couvrir plusieurs joueurs, et deux d'entre eux fraguent
// couramment dans le même match à la même milliseconde. Sans le tueur dans la clé, les deux
// frags s'écrasent dans la table d'appariement : le dernier lu fournit sa distance aux DEUX
// entames, et le delta publié décrit un engagement qui n'a jamais eu lieu.
//
// FIXTURE CONSTRUITE POUR QUE L'ERREUR SOIT VISIBLE : même match, même instant, deux tueurs.
// Apparié correctement -> deux deltas négatifs (-2 et -70), part de fermeture 100 %. Avec la
// collision -> le frag de kB (30 m) sert aussi à kA, dont le delta devient +18 : la part de
// fermeture tombe à 50 % et la médiane change de signe. La MÉDIANE SEULE ne suffirait pas à
// distinguer les deux cas, la part de fermeture le fait.
func TestWeaponOpeningDelta_LeTueurFaitPartieDeLaCle(t *testing.T) {
	kills := []MeasuredKill{
		mk("m1", "kA", 1000, SideKiller, 10),
		mk("m1", "kB", 1000, SideKiller, 30),
	}
	openings := []MeasuredKill{
		mk("m1", "kA", 1000, SideKiller, 12),  // kA ferme de 2 m
		mk("m1", "kB", 1000, SideKiller, 100), // kB ferme de 70 m
	}

	st := WeaponOpeningDelta(kills, openings, SideKiller)
	if st.Paired != 2 {
		t.Fatalf("Paired = %d, attendu 2 (deux tueurs distincts au même instant)", st.Paired)
	}
	if math.Abs(st.MedianDeltaM-(-36)) > epsDelta {
		t.Errorf("MedianDeltaM = %v, attendu -36 (médiane de -2 et -70) — la valeur -26 "+
			"signalerait que le frag de kB a été apparié à l'entame de kA", st.MedianDeltaM)
	}
	if math.Abs(st.ClosingShare-1) > epsDelta {
		t.Errorf("ClosingShare = %v, attendu 1 : les DEUX engagements se ferment. Une part de "+
			"0,5 signalerait que la clé confond les deux tueurs (l'entame de kA appariée à la "+
			"distance de kB rendrait un delta positif)", st.ClosingShare)
	}
}
