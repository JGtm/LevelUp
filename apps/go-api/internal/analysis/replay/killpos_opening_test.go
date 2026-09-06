package replay

// killpos_opening_test.go — le décalage d'entame, et LA FRONTIÈRE DE VIE qui le rend publiable.
//
// CE QUE CE FICHIER ÉPINGLE. (1) `ShiftKillRefs` décale tous les couples sans muter l'entrée.
// (2) `BuildKillOpenings` rend la position d'AVANT le coup fatal, et l'instant qu'elle porte
// est celui du KILL — c'est la clé de jointure. (3) Elle refuse une position issue d'une AUTRE
// VIE que celle du kill : un point d'apparition n'est jamais présenté comme une entame, et le
// côté écarté est COMPTÉ.
//
// LES TROIS TESTS DE REFUS PORTENT LEUR PROPRE TÉMOIN : chacun vérifie d'abord que le
// placement NU (`BuildKillPositions` sur les couples décalés) tombe bien dans le piège. Sans ce
// témoin, un test « aucune entame » resterait vert alors même que le filtre de vie aurait
// disparu — c'est exactement le défaut relevé en revue le 2026-09-06 sur la version
// précédente de ce fichier, où le refus venait d'un débordement `uint64` et non d'une règle.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestShiftKillRefsDecaleSansMuterLEntree(t *testing.T) {
	kills := []KillRef{
		{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000},
		{KillerXUID: 222, VictimXUID: 111, TimeMS: 42_500},
	}
	avant := append([]KillRef(nil), kills...)
	got := ShiftKillRefs(kills, -OpeningLeadMS)
	if len(got) != 2 {
		t.Fatalf("2 couples attendus, obtenu %d", len(got))
	}
	for i, want := range []int64{8_500, 41_000} {
		if got[i].TimeMS != want {
			t.Errorf("couple %d : instant %d, attendu %d", i, got[i].TimeMS, want)
		}
		if got[i].KillerXUID != avant[i].KillerXUID || got[i].VictimXUID != avant[i].VictimXUID {
			t.Errorf("couple %d : les identités ne doivent pas bouger : %+v", i, got[i])
		}
	}
	for i := range avant {
		if kills[i] != avant[i] {
			t.Fatalf("entrée MUTÉE au rang %d : %+v -> %+v", i, avant[i], kills[i])
		}
	}
	if ShiftKillRefs(nil, -OpeningLeadMS) != nil {
		t.Error("une entrée vide rend nil, pas une tranche de longueur 0")
	}
}

// TestBuildKillOpeningsRendLaPositionDAvantEtLInstantDuKill : le cas nominal, sur des
// trajectoires où la réponse est connue d'avance — le tueur avance de 10 m sur son adversaire
// pendant le temps-pour-tuer. L'entame doit rendre la position D'AVANT, et l'INSTANT DU KILL.
func TestBuildKillOpeningsRendLaPositionDAvantEtLInstantDuKill(t *testing.T) {
	const mortMS = 3_000
	// posAt(slot, tUS, x, y, cap) : le dernier nombre est un CAP DE VISÉE, pas une altitude —
	// seul l'écart en X porte la mesure ici. Les deux joueurs apparaissent à t = 0 et vivent
	// sans interruption jusqu'au coup fatal : une seule vie chacun.
	pos := []filmdec.BipedPosition{
		posAt(1, 0, 0, 0, 0), posAt(2, 0, 20, 0, 0),
		posAt(1, uint64(mortMS-OpeningLeadMS)*1000, 10, 0, 0),
		posAt(2, uint64(mortMS-OpeningLeadMS)*1000, 20, 0, 0),
		posAt(1, mortMS*1000, 19, 0, 0), posAt(2, mortMS*1000, 20, 0, 0),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: mortMS}}

	fatal, _ := BuildKillPositions(pos, slotXUID, kills, 0)
	entame, rep := BuildKillOpenings(pos, slotXUID, kills, 0)
	if len(fatal) != 1 || len(entame) != 1 || rep.Both != 1 || rep.OpeningOutOfLife != 0 {
		t.Fatalf("les deux placements devaient aboutir : fatal %+v, entame %+v (%+v)",
			fatal, entame, rep)
	}
	if fatal[0].Killer.X != 19 {
		t.Errorf("position fatale du tueur : X = %v, attendu 19", fatal[0].Killer.X)
	}
	if entame[0].Killer.X != 10 || entame[0].Victim.X != 20 {
		t.Errorf("position d'entame : tueur X = %v (attendu 10), victime X = %v (attendu 20)",
			entame[0].Killer.X, entame[0].Victim.X)
	}
	if entame[0].TimeMS != mortMS {
		t.Errorf("l'entame porte l'instant %d ; c'est celui du KILL (%d) qui joint la mort",
			entame[0].TimeMS, mortMS)
	}
}

// TestBuildKillOpeningsRefuseUnPointDApparition : LE cas de la revue du 2026-09-06. Les deux
// joueurs n'ont d'échantillon qu'à leur apparition (t = 0) ; la mort tombe 1 550 ms plus tard,
// donc l'instant décalé (50 ms) est DANS la tolérance de 120 ms du point d'apparition. Le
// placement nu y répond par les deux points d'apparition ; l'entame doit refuser.
func TestBuildKillOpeningsRefuseUnPointDApparition(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posAt(1, 0, 0, 0, 0),
		posAt(2, 0, 800, 800, 0),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 1_550}}

	// Témoin : sans frontière de vie, le placement TOMBE dans le piège. Si ce témoin cesse de
	// valoir, le test qui suit ne prouve plus rien et il faut le réécrire.
	piege, _ := BuildKillPositions(pos, slotXUID, ShiftKillRefs(kills, -OpeningLeadMS), 0)
	if len(piege) != 1 || piege[0].Killer == nil || piege[0].Victim == nil {
		t.Fatalf("témoin invalide : le placement nu devait rendre les deux apparitions, %+v", piege)
	}

	got, rep := BuildKillOpenings(pos, slotXUID, kills, 0)
	if len(got) != 0 {
		t.Fatalf("une apparition n'est pas une entame : %+v", got)
	}
	if rep.OpeningOutOfLife != 2 || rep.Dropped != 1 || rep.Both != 0 {
		t.Fatalf("les deux côtés devaient être comptés hors vie et la mort abandonnée : %+v", rep)
	}
}

// TestBuildKillOpeningsRefuseLaVieSuivante : la victime a DEUX vies sur le même slot, séparées
// d'un trou supérieur à `lifeGapUS`. Elle est tuée peu après sa seconde apparition : l'instant
// décalé tombe 50 ms AVANT cette apparition — dans la tolérance de placement, mais hors de la
// vie. Le tueur, lui, vit sans interruption : son entame reste publiée.
func TestBuildKillOpeningsRefuseLaVieSuivante(t *testing.T) {
	const mortMS = 21_450
	pos := []filmdec.BipedPosition{
		// tueur : une seule vie, pas de trou supérieur à 5 s.
		posAt(1, 0, 0, 0, 0), posAt(1, 4_000_000, 1, 0, 0), posAt(1, 8_000_000, 2, 0, 0),
		posAt(1, 12_000_000, 3, 0, 0), posAt(1, 16_000_000, 4, 0, 0),
		posAt(1, 20_000_000, 5, 0, 0), posAt(1, mortMS*1000, 6, 0, 0),
		// victime : première vie 0 -> 2 s, puis 18 s d'absence, puis réapparition à 20 s.
		posAt(2, 0, 500, 500, 0), posAt(2, 2_000_000, 501, 500, 0),
		posAt(2, 20_000_000, 9, 0, 0), posAt(2, mortMS*1000, 7, 0, 0),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: mortMS}}

	piege, _ := BuildKillPositions(pos, slotXUID, ShiftKillRefs(kills, -OpeningLeadMS), 0)
	if len(piege) != 1 || piege[0].Victim == nil {
		t.Fatalf("témoin invalide : le placement nu devait rendre la réapparition, %+v", piege)
	}

	got, rep := BuildKillOpenings(pos, slotXUID, kills, 0)
	if len(got) != 1 || got[0].Victim != nil || got[0].Killer == nil {
		t.Fatalf("seule l'entame du tueur devait survivre : %+v", got)
	}
	if got[0].Killer.X != 5 || got[0].TimeMS != mortMS {
		t.Errorf("entame du tueur : X = %v (attendu 5), instant %d (attendu %d)",
			got[0].Killer.X, got[0].TimeMS, mortMS)
	}
	if rep.OpeningOutOfLife != 1 || rep.KillerOnly != 1 || rep.Both != 0 || rep.Dropped != 0 {
		t.Fatalf("un seul côté hors vie, la mort reste écrite en KillerOnly : %+v", rep)
	}
}

// TestBuildKillOpeningsInstantNegatifNeDonneAucunePosition : une mort survenue dans la première
// seconde et demie du match n'a pas d'entame.
//
// CE TEST NE DOIT RIEN AU DÉBORDEMENT `uint64` de la conversion d'instant, et c'est pour cela
// qu'il porte un `offsetUS` NON NUL : l'instant de match négatif (−100 ms) devient un instant
// de film POSITIF (100 000 µs), à 100 ms du premier échantillon — donc DANS la tolérance de
// placement, comme le témoin le vérifie. Ce qui refuse, c'est la frontière de vie : avant sa
// première position, un joueur n'existe pas.
func TestBuildKillOpeningsInstantNegatifNeDonneAucunePosition(t *testing.T) {
	const offsetUS = 200_000 // l'origine du film précède celle du fil des morts de 200 ms
	pos := []filmdec.BipedPosition{
		posAt(1, 200_000, 1, 2, 0), posAt(2, 200_000, 4, 5, 0),
		posAt(1, 700_000, 1, 2, 0), posAt(2, 700_000, 4, 5, 0),
		posAt(1, 1_600_000, 1, 2, 0), posAt(2, 1_600_000, 4, 5, 0),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 1_400}}
	decales := ShiftKillRefs(kills, -OpeningLeadMS)
	if decales[0].TimeMS != -100 {
		t.Fatalf("instant décalé attendu -100 ms, obtenu %d", decales[0].TimeMS)
	}

	piege, _ := BuildKillPositions(pos, slotXUID, decales, offsetUS)
	if len(piege) != 1 || piege[0].Killer == nil || piege[0].Victim == nil {
		t.Fatalf("témoin invalide : l'instant décalé est DANS la tolérance, %+v", piege)
	}

	got, rep := BuildKillOpenings(pos, slotXUID, kills, offsetUS)
	if len(got) != 0 {
		t.Fatalf("aucune entame ne devait être écrite : %+v", got)
	}
	if rep.OpeningOutOfLife != 2 || rep.Dropped != 1 {
		t.Fatalf("les deux côtés devaient être comptés hors vie : %+v", rep)
	}
}

// TestBuildKillOpeningsEntreeVide : rien à placer, rien à compter — et surtout aucun panic sur
// l'index des trajectoires, qui n'existe pas dans ce cas.
func TestBuildKillOpeningsEntreeVide(t *testing.T) {
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 5_000}}
	got, rep := BuildKillOpenings(nil, map[uint32]uint64{1: 111}, kills, 0)
	if len(got) != 0 || rep.Kills != 1 || rep.Dropped != 1 || rep.OpeningOutOfLife != 0 {
		t.Fatalf("entrée sans position : aucune entame, une mort abandonnée, %+v (%+v)", got, rep)
	}
}
