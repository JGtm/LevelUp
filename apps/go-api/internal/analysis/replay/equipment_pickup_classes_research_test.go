package replay

// equipment_pickup_classes_research_test.go — LOT 4, ÉTAPE 2 : QU'EST-CE QUI SÉPARE LA
// CLASSE 2 DE LA CLASSE 3 ?
//
// L'HYPOTHÈSE À TESTER, telle qu'elle a été formulée : « classe 2 = équipement, classe 3 =
// grenades ». Elle ne se suppose pas — deux juges INDÉPENDANTS la départagent.
//
//	J1 — LE RANG DE PALETTE (i48). Un ramassage d'ÉQUIPEMENT doit s'accompagner d'une
//	     transmission i48 qui dit ce que le joueur porte désormais. Une grenade, non : i48
//	     est le composant de la capacité, pas de l'inventaire de grenades.
//	J2 — LE COMPTEUR DE GRENADES (i22, `InventoryDelta.Grenades`). Un ramassage de GRENADE
//	     doit faire MONTER un compteur sur le même slot. C'est le juge direct, et il ne
//	     partage aucune source avec J1.
//
// SEUILS ÉCRITS AVANT LA MESURE :
//
//	C1 — les deux classes se SÉPARENT si, sur l'un des deux juges au moins, leurs taux
//	     diffèrent d'au moins 40 points.
//	C2 — chaque taux retenu doit dépasser son TÉMOIN décalé d'un facteur 3 ; sinon il ne
//	     mesure que la densité du canal témoin.
//	C3 — l'hypothèse « 2 = équipement, 3 = grenades » n'est RETENUE que si la classe 2 mène
//	     sur J1 ET la classe 3 sur J2. Si la séparation existe mais dans l'autre sens, on
//	     l'écrit dans ce sens-là. Si elle n'existe pas, le négatif est publié tel quel.
//
// Garde BIPED_PICKUP_FILM.

import (
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// eqcGrenadeWindowUS : la fenêtre dans laquelle un ramassage de grenade doit se voir sur le
// compteur. Deux secondes — i22 n'est transmis qu'au CHANGEMENT, donc la lecture suivante
// peut tarder un peu, mais pas dix secondes sans qu'on cesse de mesurer le même geste.
const eqcGrenadeWindowUS = 2_000_000

// eqcGrenadeRise dit si un compteur de grenade du slot MONTE autour de l'instant `at`.
//
// LA COMPARAISON SE FAIT ENTRE LA DERNIÈRE LECTURE AVANT ET LA PREMIÈRE APRÈS, rang par rang :
// i22 porte le vecteur complet des compteurs, et une prise ne touche qu'un rang.
func eqcGrenadeRise(deltas []filmdec.InventoryDelta, slot uint32, at uint64, decalUS int64) bool {
	var avant, apres []uint32
	var gapAvant, gapApres uint64 = eqcGrenadeWindowUS + 1, eqcGrenadeWindowUS + 1
	for _, d := range deltas {
		if d.Slot != slot || d.Grenades == nil {
			continue
		}
		ts := int64(d.TimestampUS) + decalUS
		if ts <= int64(at) {
			if g := uint64(int64(at) - ts); g < gapAvant {
				avant, gapAvant = d.Grenades, g
			}
			continue
		}
		if g := uint64(ts - int64(at)); g < gapApres {
			apres, gapApres = d.Grenades, g
		}
	}
	if avant == nil || apres == nil {
		return false
	}
	n := len(avant)
	if len(apres) < n {
		n = len(apres)
	}
	for i := 0; i < n; i++ {
		if apres[i] > avant[i] {
			return true
		}
	}
	return false
}

// eqcTally accumule, par classe, les deux juges et leurs témoins.
type eqcTally struct {
	total   int
	rang    int
	grenade int
	tRang   int
	tGren   int
}

func TestEquipmentPickupClassSemantics(t *testing.T) {
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
	ranks, _, err := filmdec.ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Fatalf("rangs de capacité illisibles : %v", err)
	}
	deltas, dStats, err := filmdec.ScanFilmInventoryDeltas(dir)
	if err != nil {
		t.Fatalf("inventaire delta illisible : %v", err)
	}
	avecGrenades := 0
	for _, d := range deltas {
		if d.Grenades != nil {
			avecGrenades++
		}
	}
	t.Logf("== ÉTAPE 2 — CLASSE 2 vs CLASSE 3 · %s ==", dir)
	t.Logf("ramassages natifs : %d · transmissions i48 : %d · lectures i22 portant les compteurs : %d / %d (i22 au masque : %d)",
		len(pickups), len(ranks), avecGrenades, len(deltas), dStats.WithI22)

	par := map[uint8]*eqcTally{}
	for _, p := range pickups {
		e := par[p.Class]
		if e == nil {
			e = &eqcTally{}
			par[p.Class] = e
		}
		e.total++
		if len(eqnDistinct(eqnRankAt(ranks, p.Slot, p.TimestampUS, 0))) == 1 {
			e.rang++
		}
		if eqcGrenadeRise(deltas, p.Slot, p.TimestampUS, 0) {
			e.grenade++
		}
		// TÉMOINS : on garde le PIRE des trois décalages, celui qui flatte le plus le hasard.
		for _, d := range eqnDecalages {
			if len(eqnDistinct(eqnRankAt(ranks, p.Slot, p.TimestampUS, d))) == 1 {
				e.tRang++
				break
			}
		}
		for _, d := range eqnDecalages {
			if eqcGrenadeRise(deltas, p.Slot, p.TimestampUS, d) {
				e.tGren++
				break
			}
		}
	}
	for c := uint8(0); c < 8; c++ {
		e := par[c]
		if e == nil {
			continue
		}
		nature := "NON-ARME"
		if filmdec.BipedPickupIsWeaponClass(c) {
			nature = "arme"
		}
		t.Logf("  classe %d (%s) n=%d · J1 rang i48 : %.1f %% (témoin %.1f %%) · J2 compteur de grenade en HAUSSE : %.1f %% (témoin %.1f %%)",
			c, nature, e.total,
			pct100(e.rang, e.total), pct100(e.tRang, e.total),
			pct100(e.grenade, e.total), pct100(e.tGren, e.total))
	}
	c2, c3 := par[2], par[3]
	if c2 == nil || c3 == nil {
		t.Log("VERDICT : une des deux classes est absente de ce film — rien à départager.")
		return
	}
	ecartJ1 := pct100(c2.rang, c2.total) - pct100(c3.rang, c3.total)
	ecartJ2 := pct100(c2.grenade, c2.total) - pct100(c3.grenade, c3.total)
	t.Logf("ÉCART classe 2 − classe 3 : J1 (rang i48) %+.1f points · J2 (grenade) %+.1f points", ecartJ1, ecartJ2)
	sepJ1, sepJ2 := ecartJ1 <= -40 || ecartJ1 >= 40, ecartJ2 <= -40 || ecartJ2 >= 40
	t.Logf("VERDICT C1 (séparation >= 40 points sur au moins un juge) : %v (J1 %v, J2 %v)",
		sepJ1 || sepJ2, sepJ1, sepJ2)
	// C3 : l'hypothèse d'origine exige que la classe 2 mène sur J1 ET la classe 3 sur J2.
	t.Logf("VERDICT C3 (hypothèse « 2 = équipement, 3 = grenades ») : %v",
		ecartJ1 >= 40 && ecartJ2 <= -40)
}
