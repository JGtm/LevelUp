//go:build research

package grammar

// sieges_5_10_parent_research_test.go — QUI EST DANS QUEL VEHICULE, A QUEL SIEGE, ET COMMENT UN
// VEHICULE DISPARAIT.
//
// QUATRE MESURES, DANS CET ORDRE, ET CHACUNE PEUT REFUTER LA SUIVANTE :
//
//  1. LE RECENSEMENT. Combien de lectures d `i10` sur la bande BIPEDE prennent la branche
//     ATTACHEE, par chemin (delta = transition, image-cle = etat), sur combien de slots.
//  2. LA RESOLUTION DU PARENT. Le handle `Quant16` est rendu SANS la base de sa categorie
//     (`varwidth.go` : 0x200 / 0x300 / 0x400 selon la categorie, et le depot ne l ajoute pas).
//     La base se MESURE : celle qui fait atterrir le plus de handles sur un slot dont le monde
//     dit qu il est un VEHICULE (`ti=40`) gagne, et le tableau publie les autres.
//  3. LE SIEGE. `Tail6` (+0x3a0, R(6) a sentinelle) est le seul champ de six bits de la queue.
//     L ORACLE EST ECRIT AVANT LA MESURE (D1 du lot 5.5) : DEUX OCCUPANTS DU MEME VEHICULE NE
//     PEUVENT PAS PARTAGER UN SIEGE AU MEME INSTANT.
//  4. LA DISSOLUTION. `i14 object-dissolver` hors de son etat NEUTRE (13) sur un `ti=40` : le
//     seul canal au nom explicite pour une fin de vie NON DESTRUCTRICE, et la piste que
//     `components_object_state.go` nommait deja (« i14 dans les paquets DELTA, pas dans le
//     record de creation »).

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"
)

// s510Cle designe UN instant d UN vehicule : la clef sous laquelle deux occupants se croisent.
type s510Cle struct {
	ts     uint64
	parent uint32
}

// s510Bases sont les bases candidates de `FUN_1406d3140` (cf. `varwidth.go` : l initialiseur
// `FUN_140d10bb0` pose 0, 0x200, 0x300 ou 0x400 selon la categorie).
var s510Bases = []int{0, 0x200, 0x300, 0x400}

// TestSieges510Parent publie le recensement, la resolution du parent, la loi du siege et les
// sejours reconstruits.
func TestSieges510Parent(t *testing.T) {
	rec, ok := s510Passe(t)
	if !ok {
		return
	}
	att := s510Attaches(rec)
	s510Recensement(t, rec, att)
	base := s510Resolution(t, rec, att)
	s510Siege(t, rec, att, base)
	s510Episodes(t, rec, att, base)
	s510Dissolutions(t, rec)
	s510Suppressions(t, rec)
}

// s510Attaches ne garde que les lectures de la bande BIPEDE qui ont pris la branche attachee, et
// dont le slot est LIE AU BIPEDE dans le monde (meme filtre qu au lot 5.3.4 : l attribution de
// slot du chemin d inference est partielle, et un slot herite fabrique des occupants).
func s510Attaches(rec *s510Rec) []s510Parent {
	var out []s510Parent
	for _, l := range rec.parents {
		if l.ti != BipedTypeIndex || !l.st.Attached {
			continue
		}
		if rec.archetype(l.slot) != BipedTypeIndex {
			continue
		}
		out = append(out, l)
	}
	return out
}

// s510Recensement publie la population des deux branches sur la bande bipede, par chemin.
func s510Recensement(t *testing.T, rec *s510Rec, att []s510Parent) {
	t.Helper()
	var bip, libre, attDelta, attKf int
	slots := map[uint32]bool{}
	for _, l := range rec.parents {
		if l.ti != BipedTypeIndex {
			continue
		}
		bip++
		if !l.st.Attached {
			libre++
		}
	}
	for _, l := range att {
		slots[l.slot] = true
		if l.imageCle {
			attKf++
		} else {
			attDelta++
		}
	}
	t.Logf("RECENSEMENT i10 sur ti=35 : %d lectures · branche LIBRE %d · branche ATTACHEE "+
		"(slot lie au bipede) %d = %d en delta (TRANSITIONS) + %d en image-cle (ETAT), sur %d "+
		"slots bipede distincts", bip, libre, len(att), attDelta, attKf, len(slots))
}

// s510Resolution mesure la base du handle et rend celle qui maximise les atterrissages sur un
// VEHICULE. Le tableau publie TOUTES les bases candidates : c est le temoin de la mesure.
func s510Resolution(t *testing.T, rec *s510Rec, att []s510Parent) int {
	t.Helper()
	best, bestHits := s510Bases[0], -1
	for _, b := range s510Bases {
		var veh, bip, inconnu int
		for _, l := range att {
			switch rec.archetype(s510Slot(l.st, b)) {
			case VehicleTypeIndex:
				veh++
			case BipedTypeIndex:
				bip++
			default:
				inconnu++
			}
		}
		t.Logf("  base %#x : parent VEHICULE %d (%.1f %%) · parent bipede %d · slot inconnu %d",
			b, veh, m533bPart(veh, len(att)), bip, inconnu)
		if veh > bestHits {
			best, bestHits = b, veh
		}
	}
	t.Logf("RESOLUTION DU PARENT : base %#x retenue, %d lectures sur %d (%.1f %%) nomment un "+
		"vehicule", best, bestHits, len(att), m533bPart(bestHits, len(att)))
	return best
}

// s510Slot rend le slot du parent : la valeur du handle (30 bits utiles) plus la base mesuree.
func s510Slot(st ObjectParentState, base int) uint32 {
	return uint32(int(st.Quant16&0x3FFFFFFF) + base) //nolint:gosec // slot du monde
}

// s510Siege confronte `Tail6` a l oracle ecrit avant la mesure : deux occupants du MEME
// vehicule ne partagent pas un siege au MEME instant.
func s510Siege(t *testing.T, rec *s510Rec, att []s510Parent, base int) {
	t.Helper()
	dist := map[int]int{}
	var muets, aValeur int
	parInstant := map[s510Cle]map[uint32]uint32{} // (instant, parent) -> slot bipede -> siege
	for _, l := range att {
		p := s510Slot(l.st, base)
		if rec.archetype(p) != VehicleTypeIndex {
			continue
		}
		if !l.st.HasTail6 {
			muets++
			continue
		}
		aValeur++
		dist[int(l.st.Tail6)]++
		k := s510Cle{ts: l.ts, parent: p}
		if parInstant[k] == nil {
			parInstant[k] = map[uint32]uint32{}
		}
		parInstant[k][l.slot] = uint32(l.st.Tail6)
	}
	t.Logf("SIEGE CANDIDAT `Tail6` (+0x3a0, R(6)), PARENT VEHICULE SEULEMENT : %d lectures a "+
		"valeur, %d muettes (sentinelle) — ventilation :%s", aValeur, muets, m533cTable(dist))
	s510Collisions(t, parInstant)
}

// s510Collisions publie l oracle : combien d instants a plusieurs occupants, et combien y
// partagent une valeur de siege (la reponse ATTENDUE d un vrai siege est ZERO).
func s510Collisions(t *testing.T, parInstant map[s510Cle]map[uint32]uint32) {
	t.Helper()
	var multi, collisions int
	for _, occ := range parInstant {
		if len(occ) < 2 {
			continue
		}
		multi++
		vus := map[uint32]int{}
		for _, s := range occ {
			vus[s]++
		}
		if len(vus) < len(occ) {
			collisions++
		}
	}
	t.Logf("ORACLE DU SIEGE : %d couples (instant, vehicule) portent AU MOINS DEUX occupants ; "+
		"%d y partagent une valeur de siege (%.1f %%) — attendu 0 %%",
		multi, collisions, m533bPart(collisions, multi))
}

// s510Episode est un sejour continu d un bipede dans un vehicule.
type s510Episode struct {
	slot, parent   uint32
	debutMS, finMS int64
	debutKf        bool
	sieges         map[int]int
	lectures       int
}

// s510TrouMS est la coupure au-dela de laquelle deux lectures consecutives du MEME couple
// (bipede, vehicule) appartiennent a DEUX sejours. Les images-cles sont espacees de ~20 s : la
// coupure doit les laisser se chainer, sinon chaque image-cle ouvrirait son propre sejour.
const s510TrouMS = 25000

// s510Episodes reconstruit les sejours par (bipede, vehicule) et les publie tries par debut.
func s510Episodes(t *testing.T, rec *s510Rec, att []s510Parent, base int) {
	t.Helper()
	type cle struct{ slot, parent uint32 }
	courant := map[cle]*s510Episode{}
	var tous []*s510Episode
	for _, l := range att {
		p := s510Slot(l.st, base)
		if rec.archetype(p) != VehicleTypeIndex {
			continue
		}
		k := cle{slot: l.slot, parent: p}
		ms := rec.ms(l.ts)
		ep := courant[k]
		if ep == nil || ms-ep.finMS > s510TrouMS {
			ep = &s510Episode{slot: l.slot, parent: p, debutMS: ms, debutKf: l.imageCle,
				sieges: map[int]int{}}
			courant[k] = ep
			tous = append(tous, ep)
		}
		ep.finMS, ep.lectures = ms, ep.lectures+1
		if l.st.HasTail6 {
			ep.sieges[int(l.st.Tail6)]++
		}
	}
	sort.SliceStable(tous, func(i, j int) bool { return tous[i].debutMS < tous[j].debutMS })
	t.Logf("SEJOURS LUS : %d (coupure %d ms)", len(tous), s510TrouMS)
	for _, ep := range tous {
		t.Logf("  bipede %d dans vehicule %d : %s -> %s (%d lectures, debut %s) · sieges %s",
			ep.slot, ep.parent, s510Horloge(ep.debutMS), s510Horloge(ep.finMS),
			ep.lectures, s510Chemin(ep.debutKf), m533cTable(ep.sieges))
	}
}

// s510Chemin nomme le chemin d une lecture.
func s510Chemin(imageCle bool) string {
	if imageCle {
		return "image-cle"
	}
	return "delta"
}

// s510Dissolutions publie les lectures d `i14` NON NEUTRES, par archetype puis, pour les
// vehicules, une ligne par lecture : c est le candidat de la fin de vie « despawn ».
func s510Dissolutions(t *testing.T, rec *s510Rec) {
	t.Helper()
	parTI := map[int]int{}
	for _, d := range rec.diss {
		parTI[int(d.ti)]++
	}
	t.Logf("DISSOLUTION `i14` hors du NEUTRE (%d) : %d lectures sur %d — par archetype :%s",
		objectDissolverEtatNeutre, rec.i14NonNeutres, rec.i14Lectures, m533cTable(parTI))
	var n int
	for _, d := range rec.diss {
		if d.ti != VehicleTypeIndex {
			continue
		}
		n++
		t.Logf("  vehicule %d a %s (%s) : etat %d · duree brute %d · drapeau %v",
			d.slot, s510Horloge(rec.ms(d.ts)), s510Chemin(d.imageCle),
			d.val.Etat, d.val.DureeQ, d.val.Drapeau)
	}
	t.Logf("DISSOLUTIONS DE VEHICULE : %d", n)
}

// s510Suppressions publie les records de SUPPRESSION (`recDel`, type 2). C EST LE CANAL QUE LE
// DECODEUR LISAIT ET JETAIT : `DecodeFrameRecords` saute ses 32 bits et delie le slot. Un
// vehicule retire par le jeu — la fin de vie que le document publie « unknown » — doit s y
// trouver, DATE.
func s510Suppressions(t *testing.T, rec *s510Rec) {
	t.Helper()
	parTI := map[int]int{}
	for _, d := range rec.dels {
		parTI[int(d.ti)]++
	}
	t.Logf("SUPPRESSIONS `recDel` : %d records — par archetype (0 = jamais vu vivant) :%s",
		len(rec.dels), m533cTable(parTI))
	var n int
	for _, d := range rec.dels {
		if d.ti != VehicleTypeIndex {
			continue
		}
		n++
		t.Logf("  vehicule %d/%d retire a %s", d.slot, d.gen, s510Horloge(rec.ms(d.ts)))
	}
	t.Logf("SUPPRESSIONS DE VEHICULE : %d", n)
}

// TestSieges510Sejours publie les SEJOURS FERMES : un attachement ouvre, la lecture suivante
// d `i10` sur le MEME bipede ferme (branche libre = il descend ; autre parent = il change de
// vehicule). C est la forme directement comparable aux `rides[]` du document — et la sortie est
// donnee en FRAMES du document (`frameIntervalMs` 100, origine `t0FilmMs`) quand
// `SIEGE510_T0FILM` est fourni.
func TestSieges510Sejours(t *testing.T) {
	rec, ok := s510Passe(t)
	if !ok {
		return
	}
	var t0Film int64
	if v := os.Getenv("SIEGE510_T0FILM"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("SIEGE510_T0FILM %q : %v", v, err)
		}
		t0Film = int64(n)
	}
	parSlot := map[uint32][]s510Parent{}
	for _, l := range rec.parents {
		if l.ti != BipedTypeIndex {
			continue
		}
		parSlot[l.slot] = append(parSlot[l.slot], l)
	}
	slots := make([]int, 0, len(parSlot))
	for sl := range parSlot {
		slots = append(slots, int(sl))
	}
	sort.Ints(slots)
	var ouverts, fermes int
	for _, sl := range slots {
		lectures := parSlot[uint32(sl)] //nolint:gosec // slot du monde
		for i, l := range lectures {
			if !l.st.Attached {
				continue
			}
			p := s510Slot(l.st, 0x200)
			if rec.archetype(p) != VehicleTypeIndex {
				continue
			}
			fin, ferme := int64(-1), false
			if i+1 < len(lectures) {
				fin, ferme = rec.ms(lectures[i+1].ts), true
				fermes++
			} else {
				ouverts++
			}
			t.Logf("  bipede %d dans vehicule %d : %s -> %s · siege %s · frames %s",
				sl, p, s510Horloge(rec.ms(l.ts)), s510FinTexte(fin, ferme),
				s510SiegeTexte(l.st), s510Frames(rec.ms(l.ts), fin, ferme, t0Film))
		}
	}
	t.Logf("SEJOURS NOMMES PAR LE FILM : %d fermes par une lecture suivante, %d encore ouverts "+
		"a la fin", fermes, ouverts)
}

// s510FinTexte rend la borne de fin d un sejour.
func s510FinTexte(fin int64, ferme bool) string {
	if !ferme {
		return "(fin du film)"
	}
	return s510Horloge(fin)
}

// s510SiegeTexte rend le siege lu, ou la sentinelle.
func s510SiegeTexte(st ObjectParentState) string {
	if !st.HasTail6 {
		return "MUET"
	}
	return fmt.Sprintf("%d", st.Tail6)
}

// s510Frames rend l intervalle en frames du document : (instant film - t0FilmMs) / 100.
func s510Frames(debut, fin int64, ferme bool, t0Film int64) string {
	if t0Film == 0 {
		return "(t0 du document non fourni)"
	}
	f := func(ms int64) int64 { return (ms - t0Film) / 100 }
	if !ferme {
		return fmt.Sprintf("%d -> fin", f(debut))
	}
	return fmt.Sprintf("%d -> %d", f(debut), f(fin))
}

// TestSieges510Temoin dump TOUT ce que la passe sait d UN slot — le temoin nomme du lot : le
// Razorback `776/1` de `4f77afc1`, que l utilisateur a vu DISPARAITRE SANS EXPLOSION. Env
// `SIEGE510_TEMOIN` (numero de slot).
func TestSieges510Temoin(t *testing.T) {
	cible := os.Getenv("SIEGE510_TEMOIN")
	if cible == "" {
		t.Skip("SIEGE510_TEMOIN absent : instrument de recherche")
	}
	n, err := strconv.Atoi(cible)
	if err != nil {
		t.Fatalf("SIEGE510_TEMOIN %q : %v", cible, err)
	}
	slot := uint32(n) //nolint:gosec // numero de slot fourni par l operateur
	rec, ok := s510Passe(t)
	if !ok {
		return
	}
	t.Logf("TEMOIN slot %d : archetype memorise %d", slot, rec.arch[slot])
	for _, l := range rec.parents {
		p := s510Slot(l.st, 0x200)
		if l.slot != slot && p != slot {
			continue
		}
		t.Logf("  i10 a %s (%s) : record slot %d ti %d · attache %v · parent %d · siege %v/%d",
			s510Horloge(rec.ms(l.ts)), s510Chemin(l.imageCle), l.slot, l.ti, l.st.Attached,
			p, l.st.HasTail6, l.st.Tail6)
	}
	for _, d := range rec.diss {
		if d.slot != slot {
			continue
		}
		t.Logf("  i14 a %s : etat %d · duree %d", s510Horloge(rec.ms(d.ts)), d.val.Etat, d.val.DureeQ)
	}
	for _, d := range rec.dels {
		if d.slot != slot {
			continue
		}
		t.Logf("  SUPPRESSION a %s : generation %d · archetype memorise %d",
			s510Horloge(rec.ms(d.ts)), d.gen, d.ti)
	}
	t.Logf("  BANDE : %d suppressions dans [%d, %d]", s510DelsBande(rec, slot), slot-8, slot+8)
	for _, d := range rec.dels {
		if d.slot+8 < slot || d.slot > slot+8 {
			continue
		}
		t.Logf("    voisin %d/%d supprime a %s · archetype %d",
			d.slot, d.gen, s510Horloge(rec.ms(d.ts)), d.ti)
	}
}

// s510DelsBande compte les suppressions dans une petite bande autour du temoin.
func s510DelsBande(rec *s510Rec, slot uint32) int {
	var n int
	for _, d := range rec.dels {
		if d.slot+8 >= slot && d.slot <= slot+8 {
			n++
		}
	}
	return n
}
