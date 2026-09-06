package filmdec

// lot1_evenements_research_test.go — LOT 1 : LE MODELE « LISTE D'EVENEMENTS PUIS TRAME »,
// teste de bout en bout sur la famille 0xCA (unit_zoom).
//
// LE MODELE M (reconciliation des lots D et E) : un paquet delta =
//
//	[1 bit configuration][liste d'evenements : ( 1 [R(7) type][3 refs gardees][charge] )* 0]
//	[trame de records jusqu'a la fin]
//
// Il explique d'un coup : le k=2 de 0xA0/0x80/0x89 (bit 1 = 0 -> liste VIDE, trame au
// bit 2, prouve par l'oracle) ; l'arithmetique octet0 = 0xC0 | (type >> 1) qui tombe juste
// sur TOUTES les familles a bit 1 = 1 (0xD2 -> 36 action_weapon_fire, 0xC2 -> 5
// projectile_detonate, 0xC0 -> 0/1 damage_aftermath/..., 0xD3 -> 38/39 reload/throw,
// 0xE9 -> 82 PlayerGameEventSmall, 0xCA -> 20/21 incident/unit_zoom) ; et les « en-tetes de
// largeur variable » mesures (la liste d'evenements a une longueur propre au paquet).
//
// POURQUOI 0xCA : le type 21 (unit_zoom) a une charge FIXE de 2 bits (R(2), valeur-1 —
// grammaire PROUVEE au desassemblage, lot E) et ses domaines de reference sont lus dans
// l'exe (vtable+0x58 du descripteur 0x144724e80 : ref0 -> domaine 4 (R(9)), ref1 -> 8
// (R(13)), ref2 -> 7 (R(13))). L'evenement entier est donc decodable sans aucune largeur
// devinee — le test le plus pur du modele. ENJEU PRODUIT : si M tient, la conclusion
// « aucun evenement de zoom dans la bobine » (lot E, lecture decalee d'un bit) TOMBE, et la
// lunette revient dans le film (~400 000 paquets 0xCA sur le corpus).
//
// CRITERES ECRITS AVANT LA MESURE (temoin 000d5950 puis 00502e52) :
//
//	M1 — bit 8 des 0xCA : la repartition type 20 (incident) / type 21 (unit_zoom) est
//	     publiee ; le modele exige type < 123 par construction (toujours vrai ici).
//	M2 — pour les type 21 : la charge R(2) et l'index de ref0 sont publies ; attendu si
//	     l'evenement est un zoom : ref0 presente sur la quasi-totalite (l'unite qui zoome)
//	     et un PETIT nombre d'index distincts (une poignee d'unites par film).
//	M3 — apres [charge][continuation = 0], la TRAME doit decoder : part de paquets dont la
//	     trame se ferme proprement >= 50 % du taux de 0xA0 mesure par ailleurs (36 %), ET
//	     masques 1..7 >= 80 % sur les deltas lies aboutis. C'est le verdict du modele.
//	M4 — continuation = 1 (plusieurs evenements par liste) : compte publie, non decode
//	     (le 2e type est publie, on s'arrete la).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"fmt"
	"math/bits"
	"os"
	"sort"
	"testing"
)

// Les largeurs d index par domaine viennent de `refDomWidth` (event_list.go), la seule table du
// paquet depuis le 2026-09-05 (lot E, item E.3).

// lot1LireRef consomme une reference gardee du domaine dom ; rend (index, presente).
// Le domaine 1 porterait une sonde R(1) qui reduit la largeur a 9 — aucun des types testes
// ici ne l'utilise, la sonde n'est pas modelisee.
func lot1LireRef(br *BitReader, dom int) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	idx := br.ReadBits(refDomWidth(dom))
	br.Skip(2) // generation R(2)
	return idx, true
}

func TestLot1EvenementZoom(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	var (
		paquets, type20, type21, autresTypes int
		ref0Abs, cont1                       int
		zoomVals                             = map[uint64]int{}
		ref0Idx                              = map[uint64]int{}
		deuxieme                             = map[uint64]int{}
		trameOK, trameKO                     int
		deltasLies, masquesOK                int
	)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		cfg2 := DefaultFrameConfig()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg2)
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xCA {
				continue
			}
			paquets++
			br := NewBitReader(pay)
			br.Skip(1)         // bit de configuration
			if !br.ReadBit() { // continuation : un evenement suit
				continue // impossible pour 0xCA (bit 1 = 1 par construction)
			}
			typ := int(br.ReadBits(7))
			switch typ {
			case 20:
				type20++
				continue // charge variable (incident) : non decode ici
			case 21:
				type21++
			default:
				autresTypes++
				continue
			}
			// Trois references gardees, domaines lus dans l'exe : 4, 8, 7.
			idx0, ok0 := lot1LireRef(br, 4)
			_, _ = lot1LireRef(br, 8)
			_, _ = lot1LireRef(br, 7)
			if ok0 {
				ref0Idx[idx0]++
			} else {
				ref0Abs++
			}
			zoomVals[br.ReadBits(2)]++ // la charge : R(2), niveau de lunette + 1
			if br.ReadBit() {          // continuation
				cont1++
				deuxieme[br.ReadBits(7)]++
				continue // liste multiple : le 2e evenement n'est pas decode
			}
			// Fin de liste : la TRAME de records commence ici.
			w := NewWorld(reg)
			w.Restore(snap)
			recs, decErr := DecodeFrameRecords(br, w, DefaultFrameConfig())
			if decErr == nil {
				trameOK++
			} else {
				trameKO++
			}
			for i := range recs {
				r := &recs[i]
				nm := bits.OnesCount64(r.Trace.Mask)
				if r.Type == recDelta && r.DesyncAt == -1 && nm > 0 {
					deltasLies++
					if nm <= 7 {
						masquesOK++
					}
				}
			}
		}
	}
	t.Logf("== 0xCA sur %d paquets ==", paquets)
	t.Logf("M1 : type 21 (unit_zoom) x%d · type 20 (incident) x%d · autres x%d",
		type21, type20, autresTypes)
	t.Logf("M2 : ref0 (l'unite, domaine 4) : %d index distincts, absente x%d — distribution : %s",
		len(ref0Idx), ref0Abs, lot1TopU64(ref0Idx, 12))
	t.Logf("M2 : charge R(2) (niveau+1) : %s", lot1TopU64(zoomVals, 4))
	t.Logf("M4 : listes multiples (continuation=1) x%d — 2e type : %s", cont1, lot1TopU64(deuxieme, 6))
	t.Logf("M3 : trame apres l'evenement : fermee proprement %d / non %d (%.1f %%) · "+
		"deltas lies aboutis %d, masques 1..7 : %.1f %%",
		trameOK, trameKO, lot1Pct(trameOK, trameOK+trameKO), deltasLies, lot1Pct(masquesOK, deltasLies))
	okM3 := lot1Pct(trameOK, trameOK+trameKO) >= 18 && lot1Pct(masquesOK, deltasLies) >= 80 && deltasLies >= 30
	t.Logf("VERDICT M3 (fermeture >= 18 %%, masques >= 80 %%, n >= 30) : %s", lot1Verdict(okM3))
}

// lot1TopU64 rend les k entrees les plus frequentes d'un histogramme a cle entiere.
func lot1TopU64(m map[uint64]int, k int) string {
	type kv struct {
		k uint64
		v int
	}
	var s []kv
	for key, v := range m {
		s = append(s, kv{key, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	if len(s) > k {
		s = s[:k]
	}
	out := ""
	for i, e := range s {
		if i > 0 {
			out += " · "
		}
		out += fmt.Sprintf("%d x%d", e.k, e.v)
	}
	return out
}
