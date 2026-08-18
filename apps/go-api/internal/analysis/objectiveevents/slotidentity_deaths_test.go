package objectiveevents

import (
	"strconv"
	"testing"
)

// slotidentity_deaths_test.go — le pont par INSTANTS DE MORT, et la condition qui declenche
// son repli.
//
// LES DEUX PROPRIETES QUI COMPTENT, ET RIEN D'AUTRE :
//
//	film COMPLET  -> le resultat est IDENTIQUE a celui de `SlotIdentity` (la voie neuve ne
//	                 peut pas degrader ce qui marchait) ;
//	film TRONQUE  -> les huit slots sont nommes la ou le pont par totaux n'en nomme aucun.
//
// Les enregistrements sont SYNTHETIQUES : ils portent exactement ce que les deux ponts lisent
// (le compteur de morts en `comp 2 B`, les frags en `comp 2 A`, les assistances en `comp 3 A`),
// donc le test tourne en CI, sans film. La verite terrain sur films reels, elle, vit dans
// l'instrument sous garde de la phase 0 (`analysis/replay/objectifs_phase0_statborg_test.go`),
// qui appelle CE code.

// deathBridgeFixture fabrique un film synthetique a huit joueurs :
//
//	slot 10+2i  <-> xuid "100i", qui meurt (i+6) fois, aux instants 10000*(k+1) + 500*(i+1) —
//	des instants DISJOINTS d'un joueur a l'autre (500 ms d'ecart, contre 150 ms de
//	tolerance d'appariement), sans quoi la marge de prudence refuserait des slots.
//
// `emis` est le nombre de morts que le STATBORG a eu le temps de repliquer : le passer plus
// petit que le total simule un film TRONQUE (les compteurs s'arretent avant la fin, alors que
// la ligne de match, elle, porte le total).
func deathBridgeFixture(emis func(i int) int) ([]StatRecord, []PlayerLine, []DeathInstant) {
	var recs []StatRecord
	var lines []PlayerLine
	var deaths []DeathInstant
	for i := 0; i < 8; i++ {
		slot, xuid, total := 10+2*i, "100"+strconv.Itoa(i), i+6
		for k := 0; k < total; k++ {
			t := 10000*(k+1) + 500*(i+1)
			deaths = append(deaths, DeathInstant{XUID: xuid, TimeMS: t})
			if k < emis(i) {
				recs = append(recs, StatRecord{TimeMS: t, Slot: slot,
					Comps: map[int]StatValue{coreKillsComp: {A: int64(i), B: int64(k + 1)}}})
			}
		}
		// L'emission finale porte le triplet complet du joueur : c'est elle que le pont par
		// TOTAUX compare a la ligne de match.
		recs = append(recs, StatRecord{TimeMS: 900000 + i, Slot: slot,
			Comps: map[int]StatValue{
				coreKillsComp:   {A: int64(i), B: int64(emis(i))},
				coreAssistsComp: {A: int64(i)},
			}})
		lines = append(lines, PlayerLine{XUID: xuid, Kills: i, Deaths: total, Assists: i})
	}
	return recs, lines, deaths
}

// TestSlotIdentityResolvedFilmComplet — sur un film dont les compteurs vont au bout, le
// resultat doit etre EXACTEMENT celui du pont par totaux.
func TestSlotIdentityResolvedFilmComplet(t *testing.T) {
	recs, lines, deaths := deathBridgeFixture(func(i int) int { return i + 6 })
	attendu := slotIdentityFrom(recs, lines)
	if len(attendu) != 8 {
		t.Fatalf("fixture invalide : le pont par totaux nomme %d slots sur 8", len(attendu))
	}
	got, st := slotIdentityResolvedFrom(recs, lines, deaths)
	if st.Source != IdentitySourceTotals {
		t.Errorf("voie retenue %q, attendu %q (film complet)", st.Source, IdentitySourceTotals)
	}
	if len(got) != len(attendu) {
		t.Fatalf("%d slots nommes, attendu %d", len(got), len(attendu))
	}
	for slot, xuid := range attendu {
		if got[slot] != xuid {
			t.Errorf("slot %d -> %q, le pont par totaux disait %q", slot, got[slot], xuid)
		}
	}
	if st.Conflicts != 0 {
		t.Errorf("%d desaccords entre les deux ponts sur un film complet", st.Conflicts)
	}
}

// TestSlotIdentityResolvedFilmTronque — le cas qui a motive ce pont : le film s'arrete avant
// que les compteurs atteignent les totaux de l'API. Le pont par totaux ne nomme PERSONNE ; le
// pont par instants nomme les huit.
func TestSlotIdentityResolvedFilmTronque(t *testing.T) {
	recs, lines, deaths := deathBridgeFixture(func(i int) int { return i + 3 })
	if n := len(slotIdentityFrom(recs, lines)); n != 0 {
		t.Fatalf("fixture invalide : le pont par totaux nomme %d slots sur un film tronque", n)
	}
	got, st := slotIdentityResolvedFrom(recs, lines, deaths)
	if st.Source != IdentitySourceDeaths {
		t.Fatalf("voie retenue %q, attendu %q (film tronque)", st.Source, IdentitySourceDeaths)
	}
	if len(got) != 8 {
		t.Fatalf("%d slots nommes sur 8 (ByTotals=%d, ByDeaths=%d)", len(got), st.ByTotals, st.ByDeaths)
	}
	for i := 0; i < 8; i++ {
		slot, want := 10+2*i, "100"+strconv.Itoa(i)
		if got[slot] != want {
			t.Errorf("slot %d -> %q, attendu %q", slot, got[slot], want)
		}
	}
}

// TestSlotIdentityFromDeathsSeTaitSansMarge — deux joueurs qui meurent AUX MEMES INSTANTS ne
// se distinguent pas : le pont doit se taire plutot que d'en designer un.
//
// C'est la regle de prudence du paquet, et elle vaut plus que la couverture : sur une carte,
// un drapeau attribue au mauvais joueur serait invisible et credible.
func TestSlotIdentityFromDeathsSeTaitSansMarge(t *testing.T) {
	var recs []StatRecord
	var deaths []DeathInstant
	for k := 0; k < 5; k++ {
		t0 := 1000 * (k + 1)
		recs = append(recs, StatRecord{TimeMS: t0, Slot: 10,
			Comps: map[int]StatValue{coreKillsComp: {B: int64(k + 1)}}})
		deaths = append(deaths,
			DeathInstant{XUID: "jumeau-a", TimeMS: t0},
			DeathInstant{XUID: "jumeau-b", TimeMS: t0})
	}
	if got := slotIdentityFromDeaths(recs, deaths); len(got) != 0 {
		t.Errorf("le pont nomme %v alors que deux joueurs meurent aux memes instants", got)
	}
}

// TestSlotIdentityFromDeathsSeTaitSousLeMinimum — moins de morts communes que le minimum, et
// le pont ne conclut pas.
func TestSlotIdentityFromDeathsSeTaitSousLeMinimum(t *testing.T) {
	var recs []StatRecord
	var deaths []DeathInstant
	for k := 0; k < deathInstantMin-1; k++ {
		t0 := 1000 * (k + 1)
		recs = append(recs, StatRecord{TimeMS: t0, Slot: 10,
			Comps: map[int]StatValue{coreKillsComp: {B: int64(k + 1)}}})
		deaths = append(deaths, DeathInstant{XUID: "solitaire", TimeMS: t0})
	}
	if got := slotIdentityFromDeaths(recs, deaths); len(got) != 0 {
		t.Errorf("le pont nomme %v sur %d morts communes seulement (minimum %d)",
			got, deathInstantMin-1, deathInstantMin)
	}
}

// TestSlotIdentityResolvedSansFilDesMorts — fil des morts illisible : on retombe exactement
// sur le pont par totaux, sans erreur ni table vide.
func TestSlotIdentityResolvedSansFilDesMorts(t *testing.T) {
	recs, lines, _ := deathBridgeFixture(func(i int) int { return i + 6 })
	got, st := slotIdentityResolvedFrom(recs, lines, nil)
	if st.Source != IdentitySourceTotals || st.ByDeaths != 0 {
		t.Errorf("voie %q, ByDeaths=%d — attendu la voie par totaux seule", st.Source, st.ByDeaths)
	}
	if len(got) != 8 {
		t.Errorf("%d slots nommes sur 8 sans fil des morts", len(got))
	}
}

// TestSlotIdentityResolvedEcarteLesDesaccords — un slot que les deux ponts nomment
// DIFFEREMMENT sort de la table et se compte.
//
// Les deux chaines sont disjointes : quand elles se contredisent, l'une lit de travers et rien
// ne dit laquelle. Arbitrer serait choisir au hasard.
func TestSlotIdentityResolvedEcarteLesDesaccords(t *testing.T) {
	recs, lines, deaths := deathBridgeFixture(func(i int) int { return i + 3 })
	// Le slot 10 devient nommable par les TOTAUX, sur un AUTRE joueur que celui que ses
	// instants de mort designent : le triplet complet de "1007" est emis sur le slot 10.
	recs = append(recs, StatRecord{TimeMS: 950000, Slot: 10,
		Comps: map[int]StatValue{coreKillsComp: {A: 7, B: 13}, coreAssistsComp: {A: 7}}})
	got, st := slotIdentityResolvedFrom(recs, lines, deaths)
	if st.Conflicts != 1 {
		t.Fatalf("%d desaccords comptes, attendu 1 (ByTotals=%d, ByDeaths=%d)",
			st.Conflicts, st.ByTotals, st.ByDeaths)
	}
	if _, ok := got[10]; ok {
		t.Errorf("slot 10 publie (%q) alors que les deux ponts se contredisent dessus", got[10])
	}
}
