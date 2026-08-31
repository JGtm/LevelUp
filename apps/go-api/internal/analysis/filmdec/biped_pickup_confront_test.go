package filmdec

// biped_pickup_confront_test.go — ETAPE 4 DU CHANTIER RAMASSAGE : ce que l'evenement natif
// apporte au canal de production, mesure contre lui.
//
// PIEGE EVITE, ET IL EST GROS : le film de reference porte 135 biped_pickup pour ~476 s de
// jeu. Un appariement « il existe un biped_pickup a moins d'une seconde » est presque
// toujours vrai PAR HASARD (couverture aleatoire ~57 %), et un appariement dans une fenetre
// d'image-cle de 20 s l'est TOUJOURS. Ces criteres-la ne prouveraient rien. On exige donc que
// l'evenement natif NOMME la meme arme (son R(32) == la famille du canal i43..i46), et chaque
// taux est double d'un TEMOIN obtenu en decalant tous les horodatages des ramassages : le
// meme calcul sur un flux volontairement desynchronise donne le plancher de hasard.
//
// Garde BIPED_PICKUP_FILM, comme le reste du chantier.

import (
	"sort"
	"testing"
)

// bpkTolUS est la tolerance d'appariement temporel. Un paquet est date a la milliseconde ;
// 500 ms laisse de la marge sans rendre l'appariement gratuit.
const bpkTolUS = 500_000

// bpkDecalages sont les temoins : trois decalages fixes appliques aux ramassages. On garde le
// PIRE (le plus favorable au hasard) pour ne pas se flatter.
var bpkDecalages = []int64{37_000_000, -53_000_000, 91_000_000}

// bpkCollecte rend tous les biped_pickup du film, decodes et tries par horodatage.
func bpkCollecte(t *testing.T, f bpkFilm) []bpkEvent {
	t.Helper()
	var evs []bpkEvent
	bpkEachEvent(t, f, func(typ int, pay []byte, _ WorldSnapshot, tsUS uint64) {
		if typ != bpkTypePickup {
			return
		}
		br := NewBitReader(pay)
		br.Skip(bpkHeaderBits)
		e := bpkDecode(br)
		e.TimestampUS = tsUS
		evs = append(evs, e)
	})
	sort.Slice(evs, func(i, j int) bool { return evs[i].TimestampUS < evs[j].TimestampUS })
	return evs
}

// bpkAccordArme dit s'il existe un biped_pickup a moins de bpkTolUS de ts portant l'arme fam.
// decalUS decale artificiellement les ramassages : c'est le temoin de hasard.
func bpkAccordArme(evs []bpkEvent, ts uint64, fam uint32, decalUS int64) bool {
	for _, e := range evs {
		if !e.ObjetPresent || e.Objet != fam {
			continue
		}
		d := int64(e.TimestampUS) + decalUS - int64(ts)
		if d < 0 {
			d = -d
		}
		if d <= bpkTolUS {
			return true
		}
	}
	return false
}

// bpkArmeDansFenetre dit si un biped_pickup portant fam tombe dans [lo, hi] (temoin decale).
func bpkArmeDansFenetre(evs []bpkEvent, lo, hi uint64, fam uint32, decalUS int64) bool {
	for _, e := range evs {
		if !e.ObjetPresent || e.Objet != fam {
			continue
		}
		ts := int64(e.TimestampUS) + decalUS
		if ts >= int64(lo) && ts <= int64(hi) {
			return true
		}
	}
	return false
}

// bpkTop32 est bpkTop pour un histogramme a cle uint32.
func bpkTop32(m map[uint32]int, k int) string {
	h := make(map[uint64]int, len(m))
	for id, n := range m {
		h[uint64(id)] = n
	}
	return bpkTop(h, k)
}

// bpkPire rend le plus grand compte parmi les temoins.
func bpkPire(v []int) int {
	m := 0
	for _, n := range v {
		if n > m {
			m = n
		}
	}
	return m
}

// bpkAccord mesure (a) : chaque prise vue par i43..i46 a-t-elle un biped_pickup qui porte LA
// MEME arme, a moins de bpkTolUS ? Rend (prises, appariees, pire temoin) et remplit familles.
func bpkAccord(evs []bpkEvent, chg []HeldWeaponChange, familles map[uint32]bool) (int, int, int) {
	prises, appariees := 0, 0
	temoin := make([]int, len(bpkDecalages))
	for _, c := range chg {
		if c.Family != NoWeaponVariant {
			familles[c.Family] = true
		}
		if c.Kind != HeldWeaponTaken && c.Kind != HeldWeaponSwapped {
			continue
		}
		prises++
		if bpkAccordArme(evs, c.TimestampUS, c.Family, 0) {
			appariees++
		}
		for i, d := range bpkDecalages {
			if bpkAccordArme(evs, c.TimestampUS, c.Family, d) {
				temoin[i]++
			}
		}
	}
	return prises, appariees, bpkPire(temoin)
}

// bpkRappelStats porte le resultat de la mesure (b).
//
// trouSimple / trouSimpleNomme excluent les fenetres d'image-cle ou PLUS DE DEUX armes
// arrivent d'un coup : une telle fenetre n'est pas une suite de ramassages, c'est une
// reapparition ou l'arsenal complet est re-annonce. Les compter comme des prises ratees
// chargerait le denominateur d'evenements qui n'ont jamais eu lieu. C'est une lecture
// POSTERIEURE a la premiere mesure, et elle est publiee comme telle, a cote du taux brut.
type bpkRappelStats struct {
	arrivees, expliquees, trou, trouNomme, temoin int
	trouSimple, trouSimpleNomme                   int
}

// bpkRappel mesure (b) : les ARRIVEES d'arme revelees par les images-cles que i43..i46
// n'explique pas sont-elles NOMMEES par un biped_pickup de la fenetre portant cette arme ?
func bpkRappel(t *testing.T, evs []bpkEvent, chg []HeldWeaponChange,
	kf []KeyframeLoadout) bpkRappelStats {
	t.Helper()
	bySlot := map[uint32][]KeyframeLoadout{}
	for _, k := range kf {
		bySlot[k.Slot] = append(bySlot[k.Slot], k)
	}
	idBySlot := map[uint32][]hwEvent{}
	for _, c := range chg {
		idBySlot[c.Slot] = append(idBySlot[c.Slot], hwEvent{
			Slot: c.Slot, TimestampUS: c.TimestampUS, Kind: hwKindIdentity, IDHigh: c.Family,
		})
	}
	var st bpkRappelStats
	temoin := make([]int, len(bpkDecalages))
	for slot, list := range bySlot {
		sort.SliceStable(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for i := 1; i < len(list); i++ {
			prev, cur := hwFamilies(list[i-1]), hwFamilies(list[i])
			lo, hi := list[i-1].TimestampUS, list[i].TimestampUS
			nouvelles := 0
			for fam := range cur {
				if !prev[fam] {
					nouvelles++
				}
			}
			for fam := range cur {
				if prev[fam] {
					continue
				}
				st.arrivees++
				if hwHasFamily(idBySlot[slot], lo, hi, fam) {
					st.expliquees++
					continue
				}
				st.trou++
				nomme := bpkArmeDansFenetre(evs, lo, hi, fam, 0)
				if nomme {
					st.trouNomme++
				}
				if nouvelles <= 2 {
					st.trouSimple++
					if nomme {
						st.trouSimpleNomme++
					}
				}
				t.Logf("  TROU slot=%d fenetre=[%d,%d]ms arme=%s (%d arrivees dans la fenetre) : biped_pickup NOMMANT cette arme = %v",
					slot, lo/1000, hi/1000, hwName(fam), nouvelles, nomme)
				for k, d := range bpkDecalages {
					if bpkArmeDansFenetre(evs, lo, hi, fam, d) {
						temoin[k]++
					}
				}
			}
		}
	}
	st.temoin = bpkPire(temoin)
	return st
}

// bpkComposition mesure (c) : quelle part des familles d'arme d'i43..i46 le type 9
// connait-il, quelle part de ses evenements designe autre chose qu'une arme portee, et le
// R(3) de tete de charge separe-t-il les deux ? Rend le nombre de familles couvertes.
func bpkComposition(t *testing.T, evs []bpkEvent, objets map[uint32]int,
	familles map[uint32]bool) int {
	t.Helper()
	couvertes, armes := 0, 0
	hist := make(map[uint32]int, len(familles))
	for fam := range familles {
		hist[fam] = 1
		if objets[fam] > 0 {
			couvertes++
		}
	}
	for id, n := range objets {
		if familles[id] {
			armes += n
		}
	}
	t.Logf("(c) COMPOSITION : %d identifiants distincts cotes type 9 · familles d'arme d'i43..i46 CONNUES du type 9 : %d / %d (%.1f %%)",
		len(objets), couvertes, len(familles), bpkPct(couvertes, len(familles)))
	t.Logf("    evenements portant une arme connue d'i43..i46 : %d / %d (%.1f %%) — le reste designe des objets que le canal porteur ne voit PAS",
		armes, len(evs), bpkPct(armes, len(evs)))
	t.Logf("    identifiants du type 9 : %s", bpkTop32(objets, 14))
	t.Logf("    familles i43..i46      : %s", bpkTop32(hist, 14))
	// Le R(3) de tete de charge separe-t-il les armes du reste ? Table croisee.
	armeParKind, totalParKind := map[uint64]int{}, map[uint64]int{}
	for _, e := range evs {
		totalParKind[e.Kind]++
		if e.ObjetPresent && familles[e.Objet] {
			armeParKind[e.Kind]++
		}
	}
	for k := uint64(0); k < 8; k++ {
		if totalParKind[k] == 0 {
			continue
		}
		t.Logf("    R(3)=%d : %d evenements dont %d portant une arme d'i43..i46 (%.1f %%)",
			k, totalParKind[k], armeParKind[k], bpkPct(armeParKind[k], totalParKind[k]))
	}
	return couvertes
}

// TestBipedPickupConfrontation — LE TEST QUI DECIDE DU PRODUIT.
//
//	(a) ACCORD — les prises que i43..i46 voit sont-elles retrouvees, ARME NOMMEE, par le
//	    type 9 ? C'est la condition pour que l'evenement natif remplace le canal actuel.
//	(b) RAPPEL — le trou de rappel du canal actuel (les arrivees d'arme que les images-cles
//	    revelent mais que i43..i46 rate) est-il comble, ARME NOMMEE, par le type 9 ?
//	(c) COMPOSITION — quelle part des familles d'arme d'i43..i46 le type 9 connait-il, et
//	    quelle part de ses evenements designe des objets que le canal porteur ne voit pas ?
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	C1 — (a) >= 70 %, ET au moins 3x le temoin decale.
//	C2 — (b) >= 50 %, ET au moins 3x le temoin decale.
//	C3 — (c) le type 9 couvre >= 80 % des familles d'arme d'i43..i46.
func TestBipedPickupConfrontation(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	evs := bpkCollecte(t, f)
	if len(evs) == 0 {
		t.Skip("aucun biped_pickup sur ce film")
	}
	objets := map[uint32]int{}
	kinds := map[uint64]int{}
	for _, e := range evs {
		kinds[e.Kind]++
		if e.ObjetPresent {
			objets[e.Objet]++
		}
	}
	duree := evs[len(evs)-1].TimestampUS - evs[0].TimestampUS

	ref := hwKeyframeRef(t, f.dir)
	chg, st, err := ScanFilmHeldWeaponChanges(f.dir, ref.setAt)
	if err != nil {
		t.Fatalf("balayage des changements d'arme : %v", err)
	}
	kf, err := ScanFilmKeyframeLoadouts(f.dir, hwCatalogue())
	if err != nil {
		t.Fatalf("images-cles illisibles : %v", err)
	}
	t.Logf("== CONFRONTATION sur %s (%d chunks) ==", f.dir, f.chunks)
	t.Logf("biped_pickup : %d sur %.0f s de jeu · i43..i46 : %d emissions (records=%d)",
		len(evs), float64(duree)/1e6, len(chg), st.Records)
	t.Logf("charge R(3) : %s", bpkTop(kinds, 8))

	familles := map[uint32]bool{}
	prises, appariees, temoinA := bpkAccord(evs, chg, familles)
	t.Logf("(a) ACCORD (MEME arme, <= %d ms) : %d / %d prises i43..i46 (%.1f %%) · TEMOIN decale (pire des 3) : %d (%.1f %%)",
		bpkTolUS/1000, appariees, prises, bpkPct(appariees, prises), temoinA, bpkPct(temoinA, prises))

	r := bpkRappel(t, evs, chg, kf)
	t.Logf("(b) RAPPEL : arrivees=%d · expliquees par i43..i46=%d · TROU=%d · trou NOMME par un biped_pickup=%d (%.1f %%) · TEMOIN decale (pire des 3) : %d (%.1f %%)",
		r.arrivees, r.expliquees, r.trou, r.trouNomme, bpkPct(r.trouNomme, r.trou),
		r.temoin, bpkPct(r.temoin, r.trou))
	t.Logf("(b') RAPPEL hors fenetres de reapparition (<= 2 arrivees simultanees) : %d / %d (%.1f %%) — lecture POSTERIEURE, publiee a cote du taux brut",
		r.trouSimpleNomme, r.trouSimple, bpkPct(r.trouSimpleNomme, r.trouSimple))

	famillesCouvertes := bpkComposition(t, evs, objets, familles)
	t.Logf("VERDICT C1 (accord >= 70 %% et >= 3x temoin) : %s · C2 (rappel >= 50 %% et >= 3x temoin) : %s · C3 (familles couvertes >= 80 %%) : %s",
		bpkVerdict(bpkPct(appariees, prises) >= 70 && appariees >= 3*temoinA),
		bpkVerdict(bpkPct(r.trouNomme, r.trou) >= 50 && r.trouNomme >= 3*r.temoin),
		bpkVerdict(bpkPct(famillesCouvertes, len(familles)) >= 80))
}
