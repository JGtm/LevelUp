package replay

// equipment_pickup_keyframe_research_test.go — VOLET B : nommer les non-armes par l'ETAT
// COMPLET des images-cles, et non plus par le seul delta i48.
//
// ## POURQUOI CETTE VOIE, ET CE QU'ELLE CHANGE
//
// Le lot 4 n'a exploite que le DELTA i48 : precis (temoin decale a 0,0 %) mais peu couvrant
// (19,5 % et 25,0 % des ramassages non-arme etiquetes). Le levier non tente est celui qui a
// servi d'oracle aux ARMES : l'ETAT porte par les images-cles, diffe entre deux releves
// consecutifs. C'est grossier (fenetre ~20 s) mais massif — c'est la COUVERTURE qui manquait,
// pas la precision.
//
// ## CE QUE LES IMAGES-CLES PORTENT VRAIMENT (recense avant d'ecrire, lecteur existant)
//
// `ScanFilmKeyframeInventory` rend deja, par bipede et par image-cle (`KeyframeInventory`) :
//   - `Grenades [4]uint32` + `GrenadesRead` — l'etat COMPLET des compteurs de grenade ;
//   - `AbilityRank int` — le rang de palette de la capacite portee, MAIS seulement dans la
//     fenetre 16..23 de la palette ; hors d'elle, l'ancre ne matche pas et la lecture n'existe
//     pas.
//
// LES DEUX FILMS DE REFERENCE SONT DE LA PALETTE `famille_b`, rangs 19-22 — donc ENTIEREMENT
// dans la fenetre lisible. La limitation d'ancrage ne mord pas ici ; elle mordrait sur un film
// de famille A (rangs 1-12), et il faut le dire avant de generaliser.
//
// Aucun lecteur n'est reimplemente : on appelle celui de la production.
//
// ## LA REGLE D'ETIQUETAGE
//
// Autour de chaque ramassage natif de classe non-arme, on prend le dernier releve d'image-cle
// AVANT et le premier APRES, sur le MEME slot, et on regarde ce qui APPARAIT :
//   - le rang de capacite CHANGE vers une valeur lisible -> etiquette « rang N » ;
//   - un compteur de grenade MONTE -> etiquette « grenade rang i ».
// UNE SEULE etiquette dans la fenetre = etiquetage ; plusieurs = ambigu, on s'abstient. Une
// valeur `R(32)` qui recoit deux etiquettes differentes est une COLLISION : on la publie, on
// ne vote pas.
//
// ## SEUILS ECRITS AVANT LA MESURE
//
//	B1 — couverture : >= 50 % des ramassages non-arme recoivent exactement une etiquette
//	     (contre 19,5-25,0 % pour la voie delta). Sous ce seuil, la voie n'apporte pas la
//	     couverture qui la justifie.
//	B2 — LE RISQUE DE CETTE VOIE, et il est structurel : la fenetre fait ~20 s, donc un
//	     instant decale tombe souvent dans la MEME paire d'images-cles. Le temoin decale doit
//	     rester sous 25 % pour que l'etiquetage mesure autre chose que « il se passe toujours
//	     quelque chose en 20 s ». S'il est haut, la voie est REFUTEE malgre sa couverture.
//	B3 — concordance avec la table du lot 4 (`eef5d48d` = rang 21 Thruster, `8e2dc574` =
//	     rang 19) : aucune contradiction la ou les deux voies se recouvrent.
//
// Garde BIPED_PICKUP_FILM. Un film par process.

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// eqkLabels rend les etiquettes que le passage d'un releve a l'autre fait APPARAITRE.
func eqkLabels(avant, apres KeyframeInventory) []string {
	var out []string
	if avant.AbilityRank != apres.AbilityRank && apres.AbilityRank >= 0 {
		out = append(out, fmt.Sprintf("rang %d", apres.AbilityRank))
	}
	if avant.GrenadesRead && apres.GrenadesRead {
		for i := range apres.Grenades {
			if apres.Grenades[i] > avant.Grenades[i] {
				out = append(out, fmt.Sprintf("grenade rang %d", i))
			}
		}
	}
	return out
}

// eqkAround rend le dernier releve avant `at` et le premier apres, pour un slot.
func eqkAround(list []KeyframeInventory, at uint64, decalUS int64) (KeyframeInventory, KeyframeInventory, bool) {
	var avant, apres KeyframeInventory
	okA, okB := false, false
	for _, r := range list {
		ts := int64(r.TimestampUS) + decalUS
		if ts <= int64(at) {
			if !okA || ts > int64(avant.TimestampUS)+decalUS {
				avant, okA = r, true
			}
			continue
		}
		if !okB || ts < int64(apres.TimestampUS)+decalUS {
			apres, okB = r, true
		}
	}
	return avant, apres, okA && okB
}

func TestEquipmentPickupKeyframeNaming(t *testing.T) {
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
	inv, st, err := ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0)
	if err != nil {
		t.Fatalf("inventaire d images-cles illisible : %v", err)
	}
	bySlot := map[uint32][]KeyframeInventory{}
	rangsLus, grenadesLues := 0, 0
	for _, r := range inv {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
		if r.AbilityRank >= 0 {
			rangsLus++
		}
		if r.GrenadesRead {
			grenadesLues++
		}
	}
	for s := range bySlot {
		l := bySlot[s]
		sort.Slice(l, func(i, j int) bool { return l[i].TimestampUS < l[j].TimestampUS })
		bySlot[s] = l
	}
	t.Logf("== VOLET B — ETAT DES IMAGES-CLES · %s ==", dir)
	t.Logf("releves d inventaire : %d sur %d records de %d images-cles · rang de capacite LU : %d · compteurs de grenade LUS : %d",
		len(inv), st.Records, st.Keyframes, rangsLus, grenadesLues)
	if rangsLus == 0 && grenadesLues == 0 {
		t.Log("VERDICT : les images-cles ne portent NI rang de capacite NI compteur de grenade " +
			"sur ce film — negatif propre, la voie n existe pas ici.")
		return
	}

	table := map[uint32]map[string]int{}
	nonArmes, etiquetes, ambigus, sansPaire, sansChangement := 0, 0, 0, 0, 0
	temoin := make([]int, len(eqnDecalages))
	parClasse := map[uint8][2]int{}
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		nonArmes++
		c := parClasse[p.Class]
		c[1]++
		for i, d := range eqnDecalages {
			if a, b, ok := eqkAround(bySlot[p.Slot], p.TimestampUS, d); ok && len(eqkLabels(a, b)) == 1 {
				temoin[i]++
			}
		}
		avant, apres, ok := eqkAround(bySlot[p.Slot], p.TimestampUS, 0)
		if !ok {
			sansPaire++
			parClasse[p.Class] = c
			continue
		}
		lab := eqkLabels(avant, apres)
		switch len(lab) {
		case 0:
			sansChangement++
		case 1:
			etiquetes++
			c[0]++
			if table[p.CatalogID] == nil {
				table[p.CatalogID] = map[string]int{}
			}
			table[p.CatalogID][lab[0]]++
		default:
			ambigus++
		}
		parClasse[p.Class] = c
	}
	pire := 0
	for _, n := range temoin {
		if n > pire {
			pire = n
		}
	}
	t.Logf("non-armes : %d · ETIQUETES : %d (%.1f %%) · ambigus : %d · sans changement : %d · sans paire d images-cles : %d",
		nonArmes, etiquetes, pct100(etiquetes, nonArmes), ambigus, sansChangement, sansPaire)
	t.Logf("TEMOIN decale (pire des 3) : %d (%.1f %%) — c est LE risque de cette voie : la fenetre fait ~20 s",
		pire, pct100(pire, nonArmes))
	for c := uint8(0); c < 8; c++ {
		if v := parClasse[c]; v[1] > 0 {
			t.Logf("  classe %d : %d / %d etiquetes (%.1f %%)", c, v[0], v[1], pct100(v[0], v[1]))
		}
	}

	ids := make([]uint32, 0, len(table))
	for id := range table {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	collisions := 0
	t.Log("TABLE identifiant -> etiquette(s) :")
	for _, id := range ids {
		m := table[id]
		if len(m) > 1 {
			collisions++
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		s := ""
		for i, k := range keys {
			if i > 0 {
				s += " · "
			}
			s += fmt.Sprintf("%s x%d", k, m[k])
		}
		mark := ""
		if len(m) > 1 {
			mark = "  <-- COLLISION"
		}
		t.Logf("  %08x : %s%s", id, s, mark)
	}
	t.Logf("identifiants etiquetes : %d · dont en COLLISION : %d (%.1f %%)",
		len(table), collisions, pct100(collisions, len(table)))
	// B3 — concordance avec la table du lot 4 (voie delta i48).
	for id, attendu := range map[uint32]string{0xeef5d48d: "rang 21", 0x8e2dc574: "rang 19"} {
		if m := table[id]; m != nil {
			t.Logf("B3 — %08x : voie delta disait %q · voie images-cles dit %v", id, attendu, m)
		}
	}
	t.Logf("VERDICT B1 (>= 50 %% etiquetes) : %v · B2 (temoin < 25 %%) : %v",
		pct100(etiquetes, nonArmes) >= 50, pct100(pire, nonArmes) < 25)
}
