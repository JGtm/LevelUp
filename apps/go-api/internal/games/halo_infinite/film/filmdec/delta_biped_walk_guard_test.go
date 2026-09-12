package filmdec

// delta_biped_walk_guard_test.go — LE MARCHEUR DE RECORDS DELTA BIPEDE RESTE UNIQUE
// (lot E, item E.4 du PLAN_V2_REJEU_FILM, 2026-09-05).
//
// # CE QUE CE FICHIER EMPECHE
//
// Neuf balayages de production portaient la MEME triple boucle — chunks, paquets delta, curseur
// de bits — avec le meme seuil `bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + i0Bits` et le
// meme appel a `matchBipedHeader`. Le squelette etait identique caractere pour caractere ; seuls
// le crochet installe et le corps de visite differaient. `delta_biped_walk.go` en fait UN
// marcheur ; ce fichier interdit qu'un dixieme site le recopie.
//
// CLAUDE.md regle 6 : « a la 3e copie, centraliser dans un helper ET ajouter un garde-rail (test
// grep) qui interdit l'ancien litteral ». Sans lui, la dette re-croit — c'est arrive une fois
// deja sur ce paquet (le predicat bot, de 8 a 36 copies apres centralisation).
//
// # CE QU'IL NE COUVRE PAS, ET C'EST DELIBERE
//
// LES SOURCES DE TEST SONT HORS PORTEE. Une vingtaine d'instruments de recherche ancrent leurs
// propres records pour mesurer une grammaire candidate, souvent avec une variante deliberee du
// seuil ou de la porte (`needTag1` different, borne de deroulage propre). Les faire passer par le
// marcheur les ferait mentir sur ce qu'ils mesurent. La portee s'etendra le jour ou ces
// instruments seront traites — c'est l'item G du registre, pas celui-ci.

import (
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// TestMarcheurDeltaBipedeEstUnique — aucune source de PRODUCTION hors `delta_biped_walk.go` ne
// compose le seuil de record ni n'appelle `matchBipedHeader`.
func TestMarcheurDeltaBipedeEstUnique(t *testing.T) {
	const (
		seuil  = "bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt"
		ancrer = "matchBipedHeader("
	)
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		for i, ligne := range lignesDeCode(t, nom) {
			if ligne == "" {
				continue
			}
			switch {
			case strings.Contains(ligne, seuil) && nom != "delta_biped_walk.go":
				t.Errorf("%s:%d recompose le seuil de record bipede :\n\t%s\n"+
					"Il se compose une seule fois, dans `deltaBipedMinRecord` "+
					"(delta_biped_walk.go).", nom, i+1, ligne)
			case strings.Contains(ligne, ancrer) &&
				nom != "delta_biped_walk.go" && nom != "offline_biped.go":
				t.Errorf("%s:%d ancre un record bipede a la main :\n\t%s\n"+
					"L'ancrage passe par `walkDeltaBipedRecords` (contexte de film) ou "+
					"`walkDeltaBipedPayload` (payload seul), delta_biped_walk.go. Neuf copies de "+
					"cette boucle ont ete ramenees a une le 2026-09-05 (lot E, item E.4).",
					nom, i+1, ligne)
			}
		}
	}
}

// TestMarcheurDeltaBipedeSArreteSurLaBorne — un payload qui ne porte aucun record valide est
// rendu SANS rien publier et sans boucler, et le seuil se compose comme avant.
//
// CE TEST NE COUVRE PAS L'AVANCE, et c'est ecrit ici parce que le journal du lot l'a un temps
// laisse croire (correction C1 de la revue, 2026-09-06) : un payload de zeros ne publie AUCUN
// record, donc l'instruction `p = i0 + i0Bits` n'y est jamais executee. L'avance est couverte
// par `TestMarcheurDeltaBipedeNeRebalaiePasUnRecordPublie`, ci-dessous.
func TestMarcheurDeltaBipedeSArreteSurLaBorne(t *testing.T) {
	lay := I0Layout{GateBits: DefaultI0GateBits, AxisW: [3]uint{14, 14, 14}}
	slots := NewSlotBand(map[uint32]bool{7: true})
	pay := make([]byte, 64)

	var vus []deltaBipedRecord
	walkDeltaBipedPayload(pay, slots, lay, true, func(r deltaBipedRecord) {
		vus = append(vus, r)
	})
	// Un payload de zeros ne porte aucun record valide : le marcheur doit rendre la main sans
	// rien publier et SANS BOUCLER. C'est la borne, et elle est la raison d'etre du seuil.
	if len(vus) != 0 {
		t.Fatalf("%d record(s) ancre(s) dans un payload de zeros : la porte d'en-tete ne tient plus", len(vus))
	}
	// Le seuil se compose comme avant : en-tete + masque minimal + i0.
	want := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	if got := deltaBipedMinRecord(lay.TotalBits()); got != want {
		t.Errorf("deltaBipedMinRecord = %d, attendu %d", got, want)
	}
}

// leurreDansLAxeDUnRecord fabrique les trois quanta d'axe d'un record biped de sorte que les 40
// bits qu'ils occupent PORTENT, des leur premier bit, un en-tete de record valide sur `slot` :
// 21 bits d'en-tete, le masque minimal (2 x 6 bits), puis les GateBits d'i0 a zero (i0 absolu,
// region 0). Les bits qui restent sont libres.
//
// C'EST LA PIECE QUI REND L'AVANCE OBSERVABLE. Un marcheur qui reprend son balayage a l'interieur
// d'un record deja publie tombe sur ce leurre et le publie ; un marcheur qui avance de
// `i0 + i0Bits` ne le voit jamais. L'en-tete est ecrit par `writeBipedHeaderEtMasque`, l'ecrivain
// partage des tests du paquet — la grammaire n'est pas re-decrite ici.
func leurreDansLAxeDUnRecord(slot uint32) (qx, qy, qz uint64) {
	axe := [3]int{int(cliffLayout.AxisW[0]), int(cliffLayout.AxisW[1]), int(cliffLayout.AxisW[2])}
	total := axe[0] + axe[1] + axe[2]
	w := &bitWriter{}
	writeBipedHeaderEtMasque(w, slot, 1, bipedMinMaskCnt)
	w.bits(0, cliffLayout.GateBits) // le gate d'i0 du leurre : i0 absolu, region 0
	w.bits(0, total-w.n)            // le reste de l'axe, libre
	return uint64(readBitsAt(w.buf, 0, axe[0])),
		uint64(readBitsAt(w.buf, axe[0], axe[1])),
		uint64(readBitsAt(w.buf, axe[0]+axe[1], axe[2]))
}

// TestMarcheurDeltaBipedeNeRebalaiePasUnRecordPublie — L'AVANCE, EXERCEE POUR DE VRAI.
//
// Le payload porte DEUX records bipedes reconnus, colles l'un a l'autre, et le composant i0 du
// premier porte un LEURRE : un en-tete de record parfaitement valide, plante 5 bits apres le
// debut d'i0. Le marcheur doit publier exactement les deux vrais records, a leurs positions, et
// ne jamais voir le leurre.
//
// PREUVE PAR MUTATION (rejouee le 2026-09-06) : `p = i0 + i0Bits` remplace par `p = i0 + 1` dans
// `walkDeltaBipedPayload` fait publier le LEURRE a la place du second record — le test rougit sur
// la position et sur le slot. C'est le trou P1 de la revue E-R1 (constat C1) : la mutation
// passait tous les temoins du lot, golden des familles compris.
func TestMarcheurDeltaBipedeNeRebalaiePasUnRecordPublie(t *testing.T) {
	const (
		slotVrai   uint32 = 517 // un slot biped du film de reference
		slotLeurre uint32 = 9   // le slot du leurre : distinct, pour que le rouge se lise
		masque     uint64 = 3
	)
	qx, qy, qz := leurreDansLAxeDUnRecord(slotLeurre)

	w := &bitWriter{}
	writeBipedRecord(w, slotVrai, 1, masque, qx, qy, qz)       // record A, leurre dans son i0
	writeBipedRecord(w, slotVrai, 1, masque, 4096, 5000, 8192) // record B, colle a A
	pay := w.buf

	enTete := bipedHeaderBits + bipedIndexBits*int(masque)
	i0Bits := cliffLayout.TotalBits()
	longueur := enTete + i0Bits
	attendus := []int{enTete, longueur + enTete}

	slots := NewSlotBand(map[uint32]bool{slotVrai: true, slotLeurre: true})
	var vus []deltaBipedRecord
	walkDeltaBipedPayload(pay, slots, cliffLayout, true, func(r deltaBipedRecord) {
		vus = append(vus, r)
	})

	if len(vus) != len(attendus) {
		t.Fatalf("%d record(s) publie(s), attendu %d : %s", len(vus), len(attendus), decrireRecords(vus))
	}
	for i, r := range vus {
		if r.I0 != attendus[i] {
			t.Errorf("record %d ancre a i0=%d, attendu %d — le marcheur a repris son balayage "+
				"DANS un record deja publie (avance `p = i0 + i0Bits`, delta_biped_walk.go)\n%s",
				i+1, r.I0, attendus[i], decrireRecords(vus))
		}
		if r.Slot != slotVrai {
			t.Errorf("record %d publie sur le slot %d : c'est le LEURRE plante dans l'axe du "+
				"premier record, donc un record publie a ete re-balaye en chevauchement\n%s",
				i+1, r.Slot, decrireRecords(vus))
		}
	}
	// L'invariant, dit autrement : deux records publies ne se chevauchent jamais.
	for i := 1; i < len(vus); i++ {
		if ecart := vus[i].I0 - vus[i-1].I0; ecart < i0Bits {
			t.Errorf("les records %d et %d se chevauchent : %d bits d'ecart entre leurs i0, "+
				"le composant i0 en fait %d", i, i+1, ecart, i0Bits)
		}
	}
}

// decrireRecords rend les records publies sous une forme lisible dans un message d'echec.
func decrireRecords(vus []deltaBipedRecord) string {
	if len(vus) == 0 {
		return "  (aucun record publie)"
	}
	var b strings.Builder
	for i, r := range vus {
		fmt.Fprintf(&b, "  record %d : i0=%d slot=%d masque=%v\n", i+1, r.I0, r.Slot, r.Mask)
	}
	return b.String()
}

// comptesDeRecordsMiniBobine — LE COMPTE DE RECORDS QUE LE MARCHEUR PUBLIE, famille par famille,
// sur la mini-bobine versionnee. Chiffres MESURES le 2026-09-06 sur la bobine du depot.
//
// CE QUE LE GOLDEN DES FAMILLES NE VOYAIT PAS. Les neuf balayages de canal delta rendent DEUX
// choses : une liste de lectures et des DENOMINATEURS (`<X>Stats.Records` = le nombre de records
// que le marcheur a ancres). Le golden des familles (`golden_minibobine_test.go`) fige la LISTE et
// JETTE les denominateurs — `ajouterSlice(r, "camoStates", camo, err)` ignore le `st` du milieu.
// Une liste de lectures peut ne pas bouger alors que le marcheur a ancre des records de plus :
// ces comptes-ci ferment ce trou.
//
// CE TEMOIN N'EST PAS L'ORACLE DE L'AVANCE, ET LA MESURE LE DIT. La mutation `p = i0 + i0Bits` ->
// `p = i0 + 1` (constat C1 de la revue E-R1) a ete rejouee le 2026-09-06 avec ces comptes en
// place : ils sont RESTES IDENTIQUES, 28 005 et 5 282 compris. Sur cette bobine, reprendre le
// balayage a l'interieur d'un record deja publie ne produit AUCUN ancrage de plus — la porte
// d'en-tete (prefixe, slot dans la bande, tag = 1, deux bits nuls, masque croissant depuis zero,
// gate d'i0 nul) est trop stricte pour qu'un i0 de position en declenche un par hasard. AUCUN
// DOUBLON N'ETAIT DONC ABSORBE PAR UN DEDOUBLONNAGE AVAL : il n'y avait pas de doublon. C'est la
// raison de fond pour laquelle le golden ne bougeait pas, et c'est pourquoi l'oracle de l'avance
// est le temoin synthetique ci-dessus, pas ces chiffres.
//
// CE QU'ILS ATTRAPENT QUAND MEME : un ancrage qui SAUTE des records ou en publie deux fois, une
// bande de slots elargie, une porte deplacee — tout ce qui change la population ancree.
var comptesDeRecordsMiniBobine = map[string]int{
	"marcheurDeltaBipede": 28005,
	"abilityCharges":      28005,
	"abilityImpulses":     28005,
	"abilityRanks":        28005,
	"camoStates":          28005,
	"equipmentChanges":    28005,
	"grappleReads":        28005,
	"heldWeaponChanges":   28005,
	"inventoryDeltas":     28005,
}

// TestMarcheurDeltaBipedeCompteSesRecordsSurLaMiniBobine — sur des OCTETS REELS, le marcheur
// ancre exactement le nombre de records mesure. Un chevauchement, une avance trop courte ou une
// porte deplacee change ce compte.
func TestMarcheurDeltaBipedeCompteSesRecordsSurLaMiniBobine(t *testing.T) {
	film, err := filmsource.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	release := LockProcessDecode()
	defer release()
	fc := NewFilmContext(film)

	familles := []struct {
		nom     string
		records func() (int, error)
	}{
		{"marcheurDeltaBipede", func() (int, error) { return comptRecordsBruts(fc) }},
		{"abilityCharges", func() (int, error) { _, st, err := ScanAbilityCharges(fc); return st.Records, err }},
		{"abilityImpulses", func() (int, error) { _, st, err := ScanAbilityImpulses(fc); return st.Records, err }},
		{"abilityRanks", func() (int, error) { _, st, err := ScanAbilityRanks(fc); return st.Records, err }},
		{"camoStates", func() (int, error) { _, st, err := ScanCamoStates(fc); return st.Records, err }},
		{"equipmentChanges", func() (int, error) {
			_, st, err := ScanEquipmentChanges(fc, nil)
			return st.Walk.Records, err
		}},
		{"grappleReads", func() (int, error) { _, st, err := ScanGrappleReads(fc); return st.Records, err }},
		{"heldWeaponChanges", func() (int, error) {
			_, st, err := ScanHeldWeaponChanges(fc, nil)
			return st.Records, err
		}},
		{"inventoryDeltas", func() (int, error) { _, st, err := ScanInventoryDeltas(fc); return st.Records, err }},
	}
	if len(familles) != len(comptesDeRecordsMiniBobine) {
		t.Fatalf("%d familles mesurees contre %d figees", len(familles), len(comptesDeRecordsMiniBobine))
	}
	for _, f := range familles {
		got, err := f.records()
		if err != nil {
			t.Errorf("%s : %v", f.nom, err)
			continue
		}
		if want := comptesDeRecordsMiniBobine[f.nom]; got != want {
			t.Errorf("%s : le marcheur a ancre %d records, %d mesures le 2026-09-06.\n"+
				"Un compte qui bouge sans changement de decodage DECLARE veut dire que l'ancrage "+
				"ou l'avance du marcheur a change (delta_biped_walk.go).", f.nom, got, want)
		}
	}
}

// comptRecordsBruts marche la mini-bobine avec le marcheur de PRODUCTION, sans aucune famille au
// dessus : c'est le compte du marcheur lui-meme.
func comptRecordsBruts(fc *FilmContext) (int, error) {
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return 0, ErrNoFilmChunk
	}
	lay, err := fc.I0Layout()
	if err != nil {
		return 0, err
	}
	n := 0
	walkDeltaBipedRecords(fc, chunks, fc.BipedSlots(), lay, func(deltaBipedRecord) { n++ })
	return n, nil
}
