package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// manches_compteurs_test.go — LA DECOUPE PAR MANCHE CONFRONTEE AU TEMPS.
//
// # LE DEFAUT QUE CES TESTS FIGENT (mesure du 2026-09-06, `51ebbc0f` et `d9781168`)
//
// Un enregistrement mal aligne declarait la manche 0 alors qu'il etait date 57 s APRES le debut
// de la manche 1, et portait 60 assistances. La suite non decroissante ne pouvait pas l'ecarter
// (une valeur PLUS GRANDE prolonge la suite au lieu de la rompre) : la manche 0 s'achevait sur
// 60, ce 60 devenait le decalage de la manche 1, et le document publiait 63 assistances pour un
// joueur qui en a 5 a la feuille (69 pour 11 sur `d9781168`). Le total concatenait alors les
// manches dans un ordre ou les INSTANTS RECULAIENT ({3167, 60} puis {3057, 61}).
//
// Chaque test ci-dessous meurt si une piece du correctif est retiree — la mutation est nommee
// dans son en-tete. Les enregistrements sont SYNTHETIQUES : aucun film, la CI les joue.

const (
	// manchesDecalageSlot : le decalage d'emission entre deux slots, en ms. Les slots OUVRENT
	// une manche au MEME instant (c'est ce que le film montre : chaque slot y emet son
	// enregistrement dense de remise a zero au meme instant) puis progressent a leur rythme.
	//
	// LE COUPLE (decalage, pas) N'EST PAS LIBRE : le pont d'identite apparie les instants de
	// mort a 150 ms pres et exige que le meilleur candidat double le suivant. Il faut donc
	// qu'aucun `d * decalage` ne tombe a moins de 150 ms d'un multiple du pas, sinon deux
	// slots distants de `d` se comptent des coincidences croisees et plus personne n'est
	// nomme. 200 et 3 000 le garantissent (il faudrait d = 15 slots).
	manchesDecalageSlot = 200
	// manchesPasProgression : l'ecart entre deux progressions d'un meme slot.
	manchesPasProgression = 3_000
	// Les deux manches de la fixture, sur l'horloge du film.
	manchesDebutR0 = 1_000
	manchesDebutR1 = 12_000
)

// manchesSlots : huit slots de joueur, comme un film 4v4.
var manchesSlots = []int{10, 12, 14, 16, 18, 20, 22, 24}

// deuxManchesFixture reproduit la forme de `51ebbc0f` : deux manches nettement separees dans le
// temps, huit slots de joueur qui ouvrent chaque manche ENSEMBLE puis progressent chacun a son
// rythme, et un train de score de mode par manche pour que `RealRounds` les tienne pour reelles.
//
// `egare` ajoute L'ECHANTILLON EGARE : un enregistrement date DANS la manche 1 (14 000 ms) qui
// declare la manche 0 et porte 60 assistances pour le slot 12.
func deuxManchesFixture(egare bool) []objectiveevents.StatRecord {
	recs := manchesCorps(0, manchesDebutR0)
	recs = append(recs, manchesCorps(1, manchesDebutR1)...)
	if egare {
		recs = append(recs, statRec(14_000, 12, 0, map[int]objectiveevents.StatValue{
			2: {A: 0, B: 0}, 3: {A: 60},
		}))
	}
	return recs
}

// manchesCorps rend une manche complete : le train de score de mode du slot d'equipe, puis le
// bloc des huit slots de joueur.
func manchesCorps(round, debut int) []objectiveevents.StatRecord {
	return append(modeRamp(6, round, debut, 500, 1, 2, 3),
		manchesBloc(round, debut, 3, manchesSlots)...)
}

// manchesBloc rend, pour chaque slot donne, l'ouverture a zero de la manche puis `n`
// progressions espacees de [manchesPasProgression], decalees slot par slot.
func manchesBloc(round, debut, n int, slots []int) []objectiveevents.StatRecord {
	var recs []objectiveevents.StatRecord
	for j, slot := range slots {
		recs = append(recs, coreLine(slot, round, debut, 0, 0, 0, 0)...)
		for i := 0; i < n; i++ {
			k := int64(i + 1)
			t := debut + 500 + j*manchesDecalageSlot + i*manchesPasProgression
			recs = append(recs, coreLine(slot, round, t, k, k, k, k*10)...)
		}
	}
	return recs
}

// manchesInstants rend les trois instants de progression du slot d'index j dans une manche.
func manchesInstants(j, debut int) []int {
	out := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		out = append(out, debut+500+j*manchesDecalageSlot+i*manchesPasProgression)
	}
	return out
}

// manchesMorts date les morts au meme instant que la progression du compteur de morts : c'est
// l'ancre du pont d'identite PAR MANCHE. Le xuid du slot d'index j est `100<slot>`.
func manchesMorts() []Death {
	var out []Death
	for j, slot := range manchesSlots {
		xuid := uint64(1_000 + slot)
		for _, debut := range []int{manchesDebutR0, manchesDebutR1} {
			for _, t := range manchesInstants(j, debut) {
				out = append(out, Death{XUID: xuid, TimeMS: int64(t)})
			}
		}
	}
	return out
}

// assistsDuSlot rend la serie d'assistances par manche d'un slot, telle que la production la
// decoupe.
func assistsDuSlot(recs []objectiveevents.StatRecord, slot int) map[int][]objectiveevents.ScorePoint {
	return objectiveevents.SeriesByRound(recs, objectiveevents.AssistsComponent, false)[slot]
}

// dernierPoint rend l'instant et la valeur du dernier point d'une suite, ou (-1, -1).
func dernierPoint(pts []objectiveevents.ScorePoint) (int, int64) {
	if len(pts) == 0 {
		return -1, -1
	}
	return pts[len(pts)-1].TimeMS, pts[len(pts)-1].Value
}

// TestEchantillonEgareNAlimentePasSaMancheDeclaree — LE TEST QUI FONDE LE CORRECTIF.
//
// MUTATION : retirer `windows.Excludes(r)` de `rawSeriesByRound` (named_series.go) fait
// remonter la manche 0 du slot 12 a 60 assistances, et le test echoue.
func TestEchantillonEgareNAlimentePasSaMancheDeclaree(t *testing.T) {
	sain := assistsDuSlot(deuxManchesFixture(false), 12)
	avecEgare := assistsDuSlot(deuxManchesFixture(true), 12)

	tSain, vSain := dernierPoint(sain[0])
	tEgare, vEgare := dernierPoint(avecEgare[0])
	if vSain != 3 {
		t.Fatalf("temoin sans egare : la manche 0 finit a %d assistances (attendu 3)", vSain)
	}
	if tEgare != tSain || vEgare != vSain {
		t.Errorf("l'echantillon egare (14 000 ms, dans la manche 1, 60 assistances) a ete range en "+
			"manche 0 : elle finit a {%d, %d} au lieu de {%d, %d}", tEgare, vEgare, tSain, vSain)
	}
	// La manche 1 du meme slot n'est pas touchee : le correctif ECARTE, il ne deplace pas.
	if _, v := dernierPoint(avecEgare[1]); v != 3 {
		t.Errorf("manche 1 du slot 12 : %d assistances, attendu 3 (le correctif ne doit rien y changer)", v)
	}
}

// TestEchantillonEgareNeGonflePasLeTotalDuJoueur — le meme defaut, vu du DOCUMENT.
//
// Sans le correctif le total valait 63 (60 de decalage + 3) ; avec, il vaut 6, la somme des deux
// manches. Le test prouve aussi que le total publie reste CHRONOLOGIQUE.
//
// MUTATION : retirer `windows.Excludes(r)` de `rawSeriesByRound` porte le total a 63.
func TestEchantillonEgareNeGonflePasLeTotalDuJoueur(t *testing.T) {
	tl, cov := buildScoreTimeline(&ScoreInput{Records: deuxManchesFixture(true)},
		manchesMorts(), multiRoundClock())
	if tl == nil || cov == nil {
		t.Fatal("aucun calque publie")
	}
	if cov.Rounds != 2 {
		t.Fatalf("couverture : %d manche(s), attendu 2", cov.Rounds)
	}
	p := playerByXUID(tl.Players, "1012")
	if p == nil {
		t.Fatalf("le joueur du slot 12 n'est pas publie : %+v", tl.Players)
	}
	if got := lastValue(p.Assists); got != 6 {
		t.Errorf("assistances totales = %d, attendu 6 (3 par manche) — l'echantillon egare a compte", got)
	}
	assertChronologique(t, "assistances", p.Assists)
}

// TestTotalNonChronologiqueEstRefuse — LE FILET, teste pour lui-meme.
//
// Le controle vit dans `objectiveevents.ChronologicalTotal` et sert les DEUX cumuls (par slot et
// par joueur). Il est mis a l'epreuve sur la forme exacte du defaut publie par `51ebbc0f` :
// {3167, 60} avant {3057, 61}, donc un instant qui recule.
//
// MUTATION : retirer l'appel a `ChronologicalTotal` dans `cumulateRounds` ou dans
// `seriesOfRounds` laisse passer le point qui recule.
func TestTotalNonChronologiqueEstRefuse(t *testing.T) {
	pts := []objectiveevents.ScorePoint{
		{TimeMS: 1_412, Slot: 12, Value: 1},
		{TimeMS: 3_167, Slot: 12, Value: 60},
		{TimeMS: 3_057, Slot: 12, Value: 61},
		{TimeMS: 4_225, Slot: 12, Value: 62},
	}
	got := objectiveevents.ChronologicalTotal(pts)
	if len(got) != 3 {
		t.Fatalf("%d points retenus, attendu 3 (le point qui recule doit etre ecarte) : %+v", len(got), got)
	}
	for i := 1; i < len(got); i++ {
		if got[i].TimeMS < got[i-1].TimeMS {
			t.Fatalf("la suite retenue recule encore : %+v", got)
		}
	}
	if got[2].TimeMS != 4_225 {
		t.Errorf("le point ecarte doit etre celui qui recule (3 057), pas un autre : %+v", got)
	}
}

// TestSerieCumuleeParSlotResteChronologique — le meme filet, sur le chemin `cumulateRounds`.
//
// Il mord la ou les bornes de manche ne peuvent RIEN : sur un film dont le numero de manche ne
// suit pas l'horloge (forme de `fb1a1a72`), la concatenation des manches dans l'ordre des
// MANCHES remonte le temps, et c'est le seul rempart.
//
// MUTATION : retirer l'appel a `ChronologicalTotal` dans `cumulateRounds` (named_series.go).
func TestSerieCumuleeParSlotResteChronologique(t *testing.T) {
	// Manche 0 TARDIVE, manche 1 PRECOCE : aucune borne n'est posable, le cumul recule.
	recs := manchesCorps(0, 30_000)
	recs = append(recs, manchesCorps(1, manchesDebutR0)...)

	series := objectiveevents.SeriesTotal(recs, objectiveevents.AssistsComponent, false)
	if len(series) == 0 {
		t.Fatal("aucune serie cumulee : le corpus ne prouve rien")
	}
	for slot, pts := range series {
		for i := 1; i < len(pts); i++ {
			if pts[i].TimeMS < pts[i-1].TimeMS {
				t.Fatalf("slot %d : la serie cumulee recule (%d apres %d)",
					slot, pts[i].TimeMS, pts[i-1].TimeMS)
			}
		}
	}
}

// TestMonoMancheInchangeParLesBornes — LA NEUTRALITE MONO-MANCHE, par construction.
//
// Un film a une seule manche n'a pas de borne : meme un enregistrement date n'importe ou et
// declarant la manche 0 reste dans la serie, exactement comme avant le correctif. C'est ce qui
// garantit que les films mono-manche du parc sortent a l'octet — mesure du 2026-09-06 sur
// `000d5950`, `c0a82e88` et `53ce4390`.
//
// TEMOIN, PAS MUTATION — et c'est assume : la neutralite tient a ce qu'une SEULE manche n'offre
// aucune PAIRE de manches, donc aucune borne ; aucune retouche d'une seule ligne ne la casse
// (abaisser le `len(marks) < 2` ne suffit pas, la boucle des paires reste vide). La preuve du
// comportement est ailleurs et elle est chiffree : `000d5950`, `c0a82e88` et `53ce4390` re-cuits
// le 2026-09-06 sortent IDENTIQUES A L'OCTET. Ce test fige l'invariant pour que sa disparition
// eventuelle se voie.
func TestMonoMancheInchangeParLesBornes(t *testing.T) {
	recs := manchesCorps(0, manchesDebutR0)
	tardif := statRec(20_000, 12, 0, map[int]objectiveevents.StatValue{3: {A: 60}})
	recs = append(recs, tardif)

	if objectiveevents.ResolveRoundWindows(recs).Excludes(tardif) {
		t.Fatal("une borne de manche a ete posee sur un film MONO-MANCHE")
	}
	_, v := dernierPoint(assistsDuSlot(recs, 12)[0])
	if v != 60 {
		t.Errorf("la serie mono-manche a change : elle finit a %d au lieu de 60 "+
			"(l'echantillon tardif doit y rester)", v)
	}
}

// TestMancheSansConsensusNeFixeAucuneBorne — la forme de `a4083bd2` (Team Slayer, un mode SANS
// manche, dont `RealRounds` retient pourtant trois manches).
//
// Sa « manche 1 » tient en UN enregistrement d'UN slot, date APRES le debut de la manche 2 : un
// debut de manche est un CONSENSUS, et un slot n'en est pas un. Sans cette garde, la borne
// 0 -> 1 tombait a 485 722 ms, 153 enregistrements sur 719 etaient jetes, 24 compteurs de joueur
// baissaient et l'identite des camps passait de `a` a `unresolved` (mesure du 2026-09-06).
//
// MUTATION : retirer le test de majorite `2*len(debuts) <= slotsMax` de `chainedRounds` fait
// tomber la borne de la manche 0 a 20 000 ms, et les huit emissions tardives de la manche 0
// sont jetees — `Outliers` cesse d'etre nul.
func TestMancheSansConsensusNeFixeAucuneBorne(t *testing.T) {
	recs := manchesCorps(0, manchesDebutR0)
	// La manche 0 continue APRES le parasite : ce sont ces emissions-la qu'une borne posee sur
	// le parasite jetterait (sur `a4083bd2`, 153 enregistrements sur 719).
	for j, slot := range manchesSlots {
		recs = append(recs, statRec(25_000+j*manchesDecalageSlot, slot, 0,
			map[int]objectiveevents.StatValue{3: {A: 4}}))
	}
	// Le parasite : UN enregistrement, UN slot, date ENTRE les deux vraies manches — la
	// chaine des debuts reste donc croissante, et seule la majorite de slots peut le refuser.
	recs = append(recs, statRec(20_000, 10, 1, map[int]objectiveevents.StatValue{3: {A: 7}}))
	recs = append(recs, manchesCorps(2, 40_000)...)

	if n := objectiveevents.ResolveRoundWindows(recs).Outliers(recs); n != 0 {
		t.Fatalf("une manche declaree par UN SEUL slot a fixe une borne : %d enregistrements ecartes", n)
	}
	if _, v := dernierPoint(assistsDuSlot(recs, 12)[0]); v != 4 {
		t.Errorf("la manche 0 du slot 12 finit a %d assistances au lieu de 4 : son emission tardive "+
			"a ete jetee par une borne posee sur le parasite", v)
	}
}

// TestDebutsNonCroissantsNePosentAucuneBorne — la garde d'ensemble : si le numero de manche ne
// suit pas l'horloge, aucune borne n'est posee du tout.
//
// Le cas construit ici est celui que la seule credibilite des bornes ne rattrape pas : la
// manche 0 est OUVERTE TARD par la majorite de ses slots (debut = 50 000 ms) mais l'ESSENTIEL de
// ses emissions est precoce, si bien que sa mediane d'instants tombe avant le debut de la
// manche 1. Une borne 0 -> 1 y passerait le test de credibilite tout en remontant le temps.
//
// MUTATION : retirer `!increasingStarts(marks)` de `ResolveRoundWindows` fait poser cette borne,
// et les emissions tardives de la manche 0 sont jetees.
func TestDebutsNonCroissantsNePosentAucuneBorne(t *testing.T) {
	precoces, tardifs := manchesSlots[:3], manchesSlots[3:]
	// Manche 0 : trois slots emettent tot et beaucoup, les cinq autres n'ouvrent qu'a 50 000 ms.
	recs := modeRamp(6, 0, 1_000, 500, 1, 2, 3)
	recs = append(recs, manchesBloc(0, 1_000, 10, precoces)...)
	recs = append(recs, manchesBloc(0, 50_000, 1, tardifs)...)
	// Manche 1 : les huit slots l'ouvrent a 20 000 ms — AVANT le debut consensuel de la manche 0.
	recs = append(recs, manchesCorps(1, 20_000)...)

	if n := objectiveevents.ResolveRoundWindows(recs).Outliers(recs); n != 0 {
		t.Fatalf("des bornes ont ete posees sur un etiquetage de manche qui ne suit pas l'horloge : "+
			"%d enregistrements ecartes", n)
	}
}

// TestEchantillonPrecoceNAlimentePasSaMancheDeclaree — L'AUTRE BORD de la fenetre.
//
// Un enregistrement peut aussi declarer une manche AVANT qu'elle ait commence : sur `24dbb67d`,
// deux slots declarent la manche 1 des 85 193 ms alors que les huit autres l'ouvrent a
// 298 909 ms. Il ouvrirait alors la courbe de la manche a un instant qui n'est pas le sien.
//
// C'est aussi ce cas qui interdit de prendre le MINIMUM comme debut de manche : le minimum
// aurait suivi l'egare a 85 193 ms et fait jeter 3 612 enregistrements legitimes de la manche 0
// (la mediane en jette 16).
//
// MUTATION : retirer `r.TimeMS < win.fromMS` de `RoundWindows.Excludes` fait ouvrir la manche 1
// du slot 10 a 5 000 ms.
func TestEchantillonPrecoceNAlimentePasSaMancheDeclaree(t *testing.T) {
	recs := deuxManchesFixture(false)
	precoce := statRec(5_000, 10, 1, map[int]objectiveevents.StatValue{3: {A: 0}})
	recs = append(recs, precoce)

	if !objectiveevents.ResolveRoundWindows(recs).Excludes(precoce) {
		t.Fatal("un enregistrement date AVANT le debut de la manche qu'il declare n'est pas ecarte")
	}
	pts := assistsDuSlot(recs, 10)[1]
	if len(pts) == 0 || pts[0].TimeMS != manchesDebutR1 {
		t.Errorf("la manche 1 du slot 10 s'ouvre a %+v au lieu de %d : l'egare precoce l'a ouverte",
			pts, manchesDebutR1)
	}
}

// TestBorneNonCredibleNestPasPosee — la forme de `fb1a1a72` et `72b0a25e` : les debuts CROISSENT
// et les deux manches ont bien leurs huit slots, mais la « manche 0 » est repandue sur tout le
// film alors que la manche suivante ouvre tot. Sa mediane d'instants tombe APRES la borne
// candidate : la borne ne separe donc rien et ne doit pas etre posee.
//
// Sans cette garde, la borne tomberait a 5 000 ms et l'essentiel de la manche 0 serait jete
// (sur `fb1a1a72`, environ 900 enregistrements sur 1 017).
//
// MUTATION : retirer le test `prev.middleMS >= next.startMS || next.middleMS < next.startMS` de
// `ResolveRoundWindows` fait poser la borne, et `Outliers` cesse d'etre nul.
func TestBorneNonCredibleNestPasPosee(t *testing.T) {
	recs := modeRamp(6, 0, 1_000, 500, 1, 2, 3)
	recs = append(recs, manchesBloc(0, 1_000, 10, manchesSlots)...) // manche 0 : tout le film
	recs = append(recs, modeRamp(6, 1, 5_000, 500, 1, 2, 3)...)
	recs = append(recs, manchesBloc(1, 5_000, 3, manchesSlots)...) // manche 1 : ouverte tot

	if n := objectiveevents.ResolveRoundWindows(recs).Outliers(recs); n != 0 {
		t.Errorf("une borne qui ne separe pas les deux manches a ete posee : %d enregistrements ecartes", n)
	}
}

// TestEchantillonEgareNeNommeAucuneAction — la MEME garde, sur l'autre marche.
//
// Les actions d'objectif passent par `rawSeriesByKey` (une seule marche pour tous les
// emplacements), pas par `rawSeriesByRound` : le filtre doit y etre AUSSI. Le defaut mesure est
// spectaculaire — sur `51ebbc0f` (Oddball, donc SANS drapeau) un enregistrement egare portant
// `comp 24 A = 58` faisait publier 58 vols de drapeau, et 994 sur `24dbb67d`.
//
// MUTATION : retirer `windows.Excludes(r)` de `rawSeriesByKey` (named_series.go) fait renaitre
// les 58 vols.
func TestEchantillonEgareNeNommeAucuneAction(t *testing.T) {
	recs := deuxManchesFixture(false)
	// L'egare : date DANS la manche 1, declare la manche 0, porte 58 vols de drapeau.
	recs = append(recs, statRec(14_000, 12, 0, map[int]objectiveevents.StatValue{24: {A: 58}}))

	vols := 0
	for _, e := range objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeFlag) {
		if e.Stat == objectiveevents.StatFlagSteals {
			vols++
		}
	}
	if vols != 0 {
		t.Errorf("%d vols de drapeau nommes a partir d'un enregistrement egare (attendu 0)", vols)
	}
}

// TestMedianeBasseNeCoupePasLOuvertureDUneManche — le DEBUT d'une manche se prend a la mediane
// BASSE des premiers instants par slot.
//
// Sur une longueur paire, la mediane haute tombe APRES le premier slot qui a ouvert la manche, et
// ce slot perd son ouverture. Le corpus a deux slots le montre a nu.
//
// MUTATION : `v[(len(v)-1)/2]` -> `v[len(v)/2]` dans `medianOfSlice` exclut l'ouverture du slot 10.
func TestMedianeBasseNeCoupePasLOuvertureDUneManche(t *testing.T) {
	recs := modeRamp(10, 0, 1_000, 500, 1, 2, 3)
	recs = append(recs, coreLine(10, 0, 1_000, 1, 1, 1, 10)...)
	recs = append(recs, coreLine(12, 0, 1_500, 1, 1, 1, 10)...)
	ouverture := statRec(20_000, 10, 1, map[int]objectiveevents.StatValue{3: {A: 1}})
	recs = append(recs, modeRamp(10, 1, 20_000, 500, 1, 2, 3)...)
	recs = append(recs, ouverture)
	recs = append(recs, coreLine(12, 1, 20_050, 1, 1, 1, 10)...)

	if objectiveevents.ResolveRoundWindows(recs).Excludes(ouverture) {
		t.Error("l'ouverture de la manche 1 par le PREMIER slot (20 000 ms) est jugee hors de sa " +
			"propre manche : la borne a ete prise a la mediane haute des debuts (20 050 ms)")
	}
}

// assertChronologique verifie qu'une serie cumulee ne recule jamais dans le temps.
func assertChronologique(t *testing.T, nom string, s ScoreSeries) {
	t.Helper()
	for i := 1; i < len(s.Total); i++ {
		if s.Total[i].T < s.Total[i-1].T {
			t.Errorf("%s : le total recule dans le temps (%d apres %d)", nom, s.Total[i].T, s.Total[i-1].T)
		}
	}
}
