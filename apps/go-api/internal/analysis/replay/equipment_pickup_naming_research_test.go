package replay

// equipment_pickup_naming_research_test.go — LOT 4, ÉTAPES 1 ET 2 : NOMMER l'objet des
// ramassages non-arme, et ÉLUCIDER ce qui sépare la classe 2 de la classe 3.
//
// L'ÉTAT AU SORTIR DU LOT 3. Les classes 2 et 3 du canal natif sont publiées — datées à la
// milliseconde, attribuées à leur ramasseur — mais leur objet reste un `R(32)` BRUT que rien
// ne nomme : le catalogue d'armes ne le connaît pas (mesuré : 0,0 % de ces classes portent une
// famille vue par i43..i46, sur 118 événements de deux films).
//
// ## ÉTAPE 1 — LA VOIE DE NOMMAGE, ET POURQUOI ELLE EXISTE
//
// Le canal i48 ne dit pas seulement QUAND un joueur change d'équipement : il dit CE QU'IL
// PORTE désormais, sous la forme d'un RANG DE PALETTE (`AbilityRank.Rank`, cf.
// filmdec/ability_rank.go). Et ce rang, le manifeste du titre le nomme
// (`replay_labels.toml`, `[[ability_palettes]]`). Chaque appariement natif ↔ i48 sur le MÊME
// slot à moins de 500 ms étiquette donc un `R(32)` avec un rang — et le rang avec un nom.
//
// LA MESURE FONDATRICE EST DÉJÀ FAITE (lot 2) : 16/26 et 10/13 des ramassages non-arme ayant
// une émission i48 dans la fenêtre ont EXACTEMENT le slot émetteur, contre 0,0 % sur les six
// témoins décalés. L'appariement est sémantique, pas une coïncidence de densité.
//
// ## CE QUI COMPTE COMME UN NOM, ET CE QUI N'EN EST PAS UN
//
// Un `R(32)` n'est NOMMÉ que si tous ses appariements s'accordent sur UN rang. Deux étiquettes
// différentes pour une même valeur = COLLISION : on la publie telle quelle et on ne nomme pas.
// Un vote majoritaire mettrait « grappin » sur un propulseur — c'est exactement la règle que
// `classifyAbilityPalette` s'impose déjà en amont.
//
// ## ÉTAPE 2 — L'HYPOTHÈSE « CLASSE 2 = ÉQUIPEMENT, CLASSE 3 = GRENADES » SE TESTE
//
// Elle ne se suppose pas. Deux juges indépendants :
//
//	J1 — le RANG. Si une classe reçoit des rangs de palette et l'autre presque aucun, c'est
//	     que la première est de l'équipement au sens d'i48 et la seconde autre chose.
//	J2 — le COMPTEUR DE GRENADES (i22, `InventoryDelta.Grenades`). Un ramassage de grenade
//	     doit faire MONTER un compteur sur le même slot. C'est le juge direct, et il est
//	     indépendant du premier.
//
// Garde BIPED_PICKUP_FILM (constante `pickupsBridgeEnv`), comme le reste du chantier.

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// eqnTolUS est la fenêtre d'appariement : la même que celle sous laquelle l'accord
// natif ↔ i48 a été mesuré au lot 2.
const eqnTolUS = 500_000

// eqnDecalages : les témoins de hasard, identiques à ceux des lots précédents.
var eqnDecalages = []int64{37_000_000, -53_000_000, 91_000_000}

// eqnRankAt rend les rangs i48 transmis par `slot` à moins de eqnTolUS de `at` (décalés de
// decalUS pour les témoins).
func eqnRankAt(ranks []filmdec.AbilityRank, slot uint32, at uint64, decalUS int64) []int {
	var out []int
	for _, r := range ranks {
		if r.Slot != slot {
			continue
		}
		d := int64(r.TimestampUS) + decalUS - int64(at)
		if d < 0 {
			d = -d
		}
		if d <= eqnTolUS {
			out = append(out, r.Rank)
		}
	}
	return out
}

// eqnDistinct réduit une liste de rangs à ses valeurs distinctes, triées.
func eqnDistinct(in []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, v := range in {
		if !seen[v] {
			seen[v], out = true, append(out, v)
		}
	}
	sort.Ints(out)
	return out
}

// eqnNaming porte le résultat du nommage d'un identifiant de catalogue.
type eqnNaming struct {
	byRank map[int]int // rang -> nombre d'appariements
	class  map[uint8]int
}

// eqnPalette charge la palette de capacités du titre et rend le nom d'un rang.
//
// LE NOM VIENT DU MANIFESTE, JAMAIS D'UNE CONSTANTE DE TEST : si `replay_labels.toml` change,
// ce test change avec lui. Un rang que la palette ne nomme pas reste un rang — on ne lui
// invente pas d'étiquette.
func eqnPalette(t *testing.T) map[int]string {
	t.Helper()
	labels := goldenReplayLabels(t)
	out := map[int]string{}
	for _, p := range labels.AbilityPalettes() {
		for rank, v := range p.Ranks {
			out[rank] = fmt.Sprintf("%s / %s [%s]", v.En, v.Fr, p.ID)
		}
	}
	return out
}

// TestEquipmentPickupNaming — ÉTAPE 1. Étiqueter les `R(32)` des classes non-arme par le rang
// de palette que le canal i48 transmet dans la même fenêtre.
//
// SEUILS ÉCRITS AVANT LA MESURE :
//
//	N1 — au moins 30 % des ramassages non-arme reçoivent un rang (sous ce seuil la voie ne
//	     couvre pas assez pour bâtir une table).
//	N2 — le TÉMOIN décalé (pire des trois) reste sous 10 % : sinon l'appariement ne mesure
//	     que la densité des émissions i48.
//	N3 — au plus 20 % des identifiants étiquetés sont en COLLISION (deux rangs distincts pour
//	     une même valeur). Au-delà, la voie ne nomme rien de stable.
func TestEquipmentPickupNaming(t *testing.T) {
	dir := os.Getenv(pickupsBridgeEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", pickupsBridgeEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, _, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	ranks, rStats, err := filmdec.ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Fatalf("rangs de capacité illisibles : %v", err)
	}
	noms := eqnPalette(t)
	t.Logf("== ÉTAPE 1 — NOMMER LES RAMASSAGES NON-ARME · %s ==", dir)
	t.Logf("ramassages natifs : %d · transmissions i48 (rang lisible) : %d sur %d lectures · palette : %d rang(s) nommé(s)",
		len(pickups), len(ranks), rStats.Read, len(noms))

	table := map[uint32]*eqnNaming{}
	nonArmes, etiquetes, ambigus := 0, 0, 0
	temoin := make([]int, len(eqnDecalages))
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		nonArmes++
		for i, d := range eqnDecalages {
			if len(eqnDistinct(eqnRankAt(ranks, p.Slot, p.TimestampUS, d))) == 1 {
				temoin[i]++
			}
		}
		got := eqnDistinct(eqnRankAt(ranks, p.Slot, p.TimestampUS, 0))
		if len(got) == 0 {
			continue
		}
		if len(got) > 1 {
			// Plusieurs rangs dans la fenêtre : l'appariement ne DÉSIGNE rien.
			ambigus++
			continue
		}
		etiquetes++
		e := table[p.CatalogID]
		if e == nil {
			e = &eqnNaming{byRank: map[int]int{}, class: map[uint8]int{}}
			table[p.CatalogID] = e
		}
		e.byRank[got[0]]++
		e.class[p.Class]++
	}
	pireTemoin := 0
	for _, n := range temoin {
		if n > pireTemoin {
			pireTemoin = n
		}
	}
	t.Logf("non-armes : %d · ÉTIQUETÉS (un seul rang dans la fenêtre) : %d (%.1f %%) · ambigus (plusieurs rangs) : %d · TÉMOIN décalé (pire des 3) : %d (%.1f %%)",
		nonArmes, etiquetes, pct100(etiquetes, nonArmes), ambigus, pireTemoin, pct100(pireTemoin, nonArmes))

	ids := make([]uint32, 0, len(table))
	for id := range table {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	collisions := 0
	t.Log("TABLE identifiant -> rang -> nom :")
	for _, id := range ids {
		e := table[id]
		if len(e.byRank) > 1 {
			collisions++
		}
		t.Logf("  %08x : rangs %s · classes %s%s",
			id, eqnHist(e.byRank, noms), eqnClassHist(e.class),
			map[bool]string{true: "  <-- COLLISION, NON NOMMÉ", false: ""}[len(e.byRank) > 1])
	}
	t.Logf("identifiants étiquetés : %d · dont en COLLISION : %d (%.1f %%)",
		len(table), collisions, pct100(collisions, len(table)))
	t.Logf("VERDICT N1 (>= 30 %% étiquetés) : %v · N2 (témoin < 10 %%) : %v · N3 (collisions <= 20 %%) : %v",
		pct100(etiquetes, nonArmes) >= 30, pct100(pireTemoin, nonArmes) < 10,
		pct100(collisions, len(table)) <= 20)
}

// eqnHist rend un histogramme rang -> compte, avec le nom du rang quand la palette le donne.
func eqnHist(m map[int]int, noms map[int]string) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += " · "
		}
		nom := noms[k]
		if nom == "" {
			nom = "(non nommé par la palette)"
		}
		out += fmt.Sprintf("%d x%d = %s", k, m[k], nom)
	}
	return out
}

func eqnClassHist(m map[uint8]int) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf("%d:%d", k, m[uint8(k)])
	}
	return out
}

func pct100(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}
