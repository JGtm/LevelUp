package replay

// pickups_bridge_research_test.go — LE GATE DE PUBLICATION DU CANAL NATIF : la traversée
// slot -> joueur, EXERCÉE.
//
// POURQUOI CE TEST EXISTE. Le lot de recherche a prouvé que la référence du ramassage natif
// résout en `512 + index` = le slot du bipède ramasseur (32/32 paires de vérité terrain, deux
// films). Mais il a laissé une réserve écrite : « la traversée slot -> xuid elle-même n'a pas
// été ré-exercée ; seule l'identité du slot est prouvée ». Publier un `xuid` sans lever cette
// réserve, ce serait publier une identité qu'on n'a pas vérifiée.
//
// CE QUE LE TEST MESURE, ET CE QU'IL NE MESURE PAS. Les deux canaux (natif et i43..i46)
// rendent tous deux un SLOT, et ces slots sont égaux sur les paires — donc « le xuid du
// ramasseur natif est celui du canal i43..i46 » est VRAI PAR CONSTRUCTION dès lors que le
// pont est une fonction du slot. Publier ce 100 % comme un résultat serait se flatter d'une
// tautologie. Ce qui est réellement en question, et que ce test mesure, c'est :
//
//	B1 — le pont NOMME-t-il ces slots ? (taux de résolution — s'il est bas, `xuid` sera
//	     souvent absent et le canal n'apporte pas l'attribution promise) ;
//	B2 — le pont est-il SANS COLLISION sur ce film ? (`OwnerReport.SlotCollisions` : c'est
//	     l'objection « les slots sont réattribués entre manches » ; une collision non nulle
//	     voudrait dire qu'un même slot porte deux joueurs et que l'attribution est fausse) ;
//	B3 — l'égalité des slots entre les deux canaux, RE-VÉRIFIÉE ici sur le chemin de
//	     production (elle fonde la tautologie ci-dessus : si elle tombait, tout tombe).
//
// SEUILS ÉCRITS AVANT LA MESURE : B1 >= 80 % des ramassages natifs nommés · B2 = 0 collision ·
// B3 = 100 % des paires à slots égaux.
//
// Garde BIPED_PICKUP_FILM.

import (
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const pickupsBridgeEnv = "BIPED_PICKUP_FILM"

func TestPickupsBridgeNamesPickers(t *testing.T) {
	dir := os.Getenv(pickupsBridgeEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", pickupsBridgeEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, pStats, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	if len(pickups) == 0 {
		t.Skip("aucun ramassage natif sur ce film")
	}
	// QuantaOnly : le pont ne se sert des positions que pour DÉCOUPER LES VIES (quel slot est
	// occupé, de quand à quand). Les coordonnées monde ne l'intéressent pas, et exiger les
	// bornes de carte ferait dépendre ce gate d'un catalogue sans rien lui apporter.
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	positions, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts illisible : %v", err)
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("index de joueur illisible : %v", err)
	}
	table, tableCollisions := injectiveOrEmpty(idx)
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		fire = nil
	}
	own := buildOwners(indexBySlot(positions), deaths, table, fireRefs(fire))

	t.Logf("== PONT slot -> joueur, EXERCE sur les ramassages natifs · %s ==", dir)
	t.Logf("ramassages natifs : %d (listes multiples %d) · morts %d · slots ponts %d · "+
		"vies nommees %d/%d · collisions de slot %d · index non injectif %d",
		len(pickups), pStats.MultiEvent, len(deaths), len(own.SlotXUID),
		own.DeathsNamed, own.LivesTotal, own.SlotCollisions, tableCollisions)

	// B1 — le pont nomme-t-il les ramasseurs ?
	named, byClass, namedByClass := 0, map[uint8]int{}, map[uint8]int{}
	for _, p := range pickups {
		byClass[p.Class]++
		if _, ok := own.SlotXUID[p.Slot]; ok {
			named++
			namedByClass[p.Class]++
		}
	}
	t.Logf("B1 — ramasseurs NOMMES : %d / %d (%.1f %%)", named, len(pickups),
		100*float64(named)/float64(len(pickups)))
	for c := uint8(0); c < 8; c++ {
		if byClass[c] == 0 {
			continue
		}
		t.Logf("     classe %d : %d / %d nommes", c, namedByClass[c], byClass[c])
	}

	// B3 — l'egalite des slots entre les deux canaux, re-verifiee sur le chemin de production.
	kf, err := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
	if err != nil {
		t.Fatalf("images-cles illisibles : %v", err)
	}
	chg, _, err := filmdec.ScanFilmHeldWeaponChanges(dir, spawnSetFrom(kf))
	if err != nil {
		t.Fatalf("changements d arme illisibles : %v", err)
	}
	paires, egaux, ambigus := 0, 0, 0
	for _, c := range chg {
		if c.Kind != filmdec.HeldWeaponTaken && c.Kind != filmdec.HeldWeaponSwapped {
			continue
		}
		var cand []filmdec.BipedPickup
		for _, p := range pickups {
			if p.CatalogID != c.Family {
				continue
			}
			d := int64(p.TimestampUS) - int64(c.TimestampUS)
			if d < 0 {
				d = -d
			}
			if d <= 500_000 {
				cand = append(cand, p)
			}
		}
		switch len(cand) {
		case 0:
		case 1:
			paires++
			if cand[0].Slot == c.Slot {
				egaux++
			}
		default:
			ambigus++
		}
	}
	t.Logf("B3 — paires non ambigues %d (ambigues ecartees %d) · MEME slot des deux cotes : %d (%.1f %%)",
		paires, ambigus, egaux, 100*float64(egaux)/float64(max(paires, 1)))
	t.Log("LECTURE : les deux canaux rendant le MEME slot, « meme xuid » en decoule par " +
		"construction — ce n'est pas une mesure independante, et on ne la presente pas comme telle.")

	okB1 := 100*float64(named)/float64(len(pickups)) >= 80
	okB2 := own.SlotCollisions == 0
	okB3 := paires > 0 && egaux == paires
	t.Logf("VERDICT B1 (>= 80 %% nommes) : %v · B2 (0 collision de slot) : %v · B3 (100 %% slots egaux) : %v",
		okB1, okB2, okB3)
	if !okB2 {
		t.Errorf("B2 : %d collision(s) de slot — un meme slot porte deux joueurs, l'attribution "+
			"des ramassages serait fausse", own.SlotCollisions)
	}
	if !okB3 {
		t.Errorf("B3 : %d/%d paires a slots egaux — le fondement de l'attribution est rompu", egaux, paires)
	}
	if !okB1 {
		t.Errorf("B1 : %d/%d ramasseurs nommes, sous le seuil de 80 %%", named, len(pickups))
	}
}
