package replay

// objectifs_phase0_famille_test.go — ITEM 0.1 : LA FAMILLE DU DRAPEAU.
//
// LA MESURE, en une phrase : dans le record d'image-cle d'un bipede, existe-t-il un
// identifiant de 32 bits qui apparait quand ce joueur PORTE le drapeau, et seulement alors ?
//
// LE BALAYAGE EST SANS PREDICAT, ET C'EST LE POINT. On ne cherche pas une famille connue
// (le drapeau n'est dans aucun catalogue d'armes) : on prend TOUTES les fenetres de 32 bits
// de l'emprise du record, et on laisse l'oracle trancher. La selectivite ne vient donc pas
// d'une liste ecrite d'avance mais de la CONFRONTATION : une valeur de hasard ne se repete
// pas d'un record a l'autre, et une constante de structure apparait aussi bien hors portage
// que pendant — les deux se font eliminer par le meme test, sans reglage.
//
// LES DEUX TAUX PUBLIES, ET POURQUOI LES DEUX. Le taux DE PORTAGE seul ne prouve rien (une
// constante presente partout le sature) ; le taux HORS PORTAGE seul non plus. C'est leur
// ECART qui est le signal, et il est mesure sur des denominateurs rendus.
//
// CE QUE CET INSTRUMENT NE FAIT PAS : il ne decode rien de l'objet, ne lit aucune grammaire
// de corps d'image-cle (voie fermee par le lot R4) et n'ecrit rien. Lecture seule.

import (
	"fmt"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// objRecord est un record de bipede d'image-cle, reduit a ce que la mesure consomme :
// quand, qui, et l'ensemble des valeurs de 32 bits que son emprise contient.
type objRecord struct {
	TS   uint64 // horloge du FILM, microsecondes
	Slot uint32
	Vals []uint32
}

// objScanKeyframeBipeds balaye les records de bipede de TOUTES les images-cles du film et
// rend, pour chacun, l'ensemble des fenetres de 32 bits de son emprise.
//
// L'EMPRISE EST CELLE DU WALKER, pas une longueur devinee : `WalkKeyframeWorld` rend le bit
// de debut de chaque record (valide 249/250 entites, 8/8 bipedes), et un record va donc du
// sien jusqu'a celui du suivant. Une fenetre est attribuee au record qui contient son
// DEBUT — meme regle que `familiesByRecord`, pour que les deux lectures restent comparables.
func objScanKeyframeBipeds(dir string) ([]objRecord, int, error) {
	n := filmdec.CountFilmChunks(dir)
	var out []objRecord
	lus, images := 0, 0
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		lus++
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			images++
			out = append(out, objBipedRecordsOf(p.Payload(data), p.TimestampUS)...)
		}
	}
	if lus == 0 {
		return nil, 0, fmt.Errorf("aucun chunk lisible dans %s", dir)
	}
	return out, images, nil
}

// objBipedRecordsOf extrait les records de bipede d'un seul payload d'image-cle. PUR.
func objBipedRecordsOf(pay []byte, ts uint64) []objRecord {
	recs := filmdec.WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	total := len(pay) * 8
	var out []objRecord
	for i, r := range recs {
		if r.TI != objBipedTI || r.Slot < 0 {
			continue
		}
		fin := total
		if i+1 < len(recs) {
			fin = recs[i+1].Bit
		}
		out = append(out, objRecord{TS: ts, Slot: uint32(r.Slot), Vals: objWindows32(pay, r.Bit, fin)})
	}
	return out
}

// objWindows32 rend les valeurs DISTINCTES des fenetres de 32 bits dont le debut tombe dans
// [from, to). La fenetre peut deborder sur le record suivant : c'est deliberement la meme
// convention que `familiesByRecord`, qui attribue une occurrence au record contenant son
// premier bit.
func objWindows32(pay []byte, from, to int) []uint32 {
	total := len(pay) * 8
	if from < 0 {
		from = 0
	}
	if to > total {
		to = total
	}
	if to <= from {
		return nil
	}
	vues := make(map[uint32]struct{}, to-from)
	var w uint32
	for b := from; b < total && b < to+31; b++ {
		w = w<<1 | uint32(pay[b>>3]>>(7-uint(b&7))&1)
		if debut := b - 31; debut >= from && debut < to {
			vues[w] = struct{}{}
		}
	}
	out := make([]uint32, 0, len(vues))
	for v := range vues {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// objWindow est une FENETRE DE PORTAGE : de la prise a la fin du portage, sur l'horloge du
// match.
type objWindow struct {
	XUID   uint64
	Kind   string
	T0, T1 int64
}

// objPortageWindows construit les fenetres de portage de l'oracle CTF.
//
// LA FIN DE PORTAGE EST LE PREMIER DES QUATRE FAITS QUI L'INTERROMPENT : la capture du
// porteur, sa mort (fil des morts — c'est aussi ce qui couvre « porteur tue », dont
// l'evenement `flag_carriers_killed` est credite au TUEUR et ne nomme donc pas sa victime),
// une nouvelle prise du meme joueur, ou la fin du match.
//
// LE LACHER VOLONTAIRE N'EST PAS OBSERVABLE et n'est donc PAS borne : une fenetre qui le
// contient est trop longue, ce qui ABAISSE le taux mesure. Le biais joue contre le signal,
// jamais en sa faveur — c'est le sens dans lequel on veut se tromper.
func objPortageWindows(evs []objectiveevents.IdentifiedEvent, deaths []Death, finMS int64) ([]objWindow, int) {
	prises := map[uint64][]objWindow{}
	captures, morts := map[uint64][]int64{}, map[uint64][]int64{}
	for _, e := range evs {
		x, err := strconv.ParseUint(e.XUID, 10, 64)
		if err != nil {
			continue
		}
		switch e.Stat {
		case objectiveevents.StatFlagGrabs:
			prises[x] = append(prises[x], objWindow{XUID: x, Kind: "prise", T0: int64(e.TimeMS)})
		case objectiveevents.StatFlagSteals:
			prises[x] = append(prises[x], objWindow{XUID: x, Kind: "vol", T0: int64(e.TimeMS)})
		case objectiveevents.StatFlagCaptures:
			captures[x] = append(captures[x], int64(e.TimeMS))
		}
	}
	for _, d := range deaths {
		morts[d.XUID] = append(morts[d.XUID], d.TimeMS)
	}
	var out []objWindow
	fusions := 0
	for x, ps := range prises {
		sort.Slice(ps, func(i, j int) bool { return ps[i].T0 < ps[j].T0 })
		ps, n := objFusionnePrises(ps)
		fusions += n
		sort.Slice(captures[x], func(i, j int) bool { return captures[x][i] < captures[x][j] })
		sort.Slice(morts[x], func(i, j int) bool { return morts[x][i] < morts[x][j] })
		for i, p := range ps {
			suivante := finMS
			if i+1 < len(ps) {
				suivante = ps[i+1].T0
			}
			p.T1 = objMinApres(p.T0, finMS, captures[x], morts[x], []int64{suivante})
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].XUID < out[j].XUID
	})
	return out, fusions
}

// objPriseFusionMS : deux prises separees de moins que cela sont LA MEME action (un vol
// incremente son propre compteur, et rien ne garantit que les deux emissions tombent sur la
// meme milliseconde). Sans cette fusion, la seconde fermerait la fenetre de la premiere a
// duree quasi nulle et la mesure perdrait une fenetre par vol.
const objPriseFusionMS = 250

// objFusionnePrises fusionne les prises trop proches ; rend le nombre de fusions.
func objFusionnePrises(ps []objWindow) ([]objWindow, int) {
	out := ps[:0:0]
	fusions := 0
	for _, p := range ps {
		if len(out) > 0 && p.T0-out[len(out)-1].T0 <= objPriseFusionMS {
			fusions++
			continue
		}
		out = append(out, p)
	}
	return out, fusions
}

// objMinApres rend le plus petit instant strictement posterieur a t0 parmi les series
// fournies, borne par defaut.
func objMinApres(t0, defaut int64, series ...[]int64) int64 {
	best := defaut
	for _, s := range series {
		for _, v := range s {
			if v > t0 && v < best {
				best = v
			}
		}
	}
	return best
}

// objCompte porte les deux comptes d'une valeur : records de portage et records hors
// portage qui la contiennent.
type objCompte struct{ Portage, Hors int }

// objTable est le resultat de la confrontation d'un film.
type objTable struct {
	Par           map[uint32]*objCompte
	Portage, Hors int
	SlotsInconnus int
	Fenetres      int
}

// objConfronte etiquette chaque record (portage / hors portage) et compte, par valeur de
// 32 bits, dans combien de records de chaque camp elle apparait.
func objConfronte(recs []objRecord, b objBridge, wins []objWindow) objTable {
	parXUID := map[uint64][]objWindow{}
	for _, w := range wins {
		parXUID[w.XUID] = append(parXUID[w.XUID], w)
	}
	t := objTable{Par: map[uint32]*objCompte{}, Fenetres: len(wins)}
	for _, r := range recs {
		x, ok := b.SlotXUID[r.Slot]
		if !ok {
			t.SlotsInconnus++
			continue
		}
		matchMS := int64(r.TS/1000) - b.OffsetMS
		porte := objDansFenetre(parXUID[x], matchMS)
		if porte {
			t.Portage++
		} else {
			t.Hors++
		}
		for _, v := range r.Vals {
			c := t.Par[v]
			if c == nil {
				c = &objCompte{}
				t.Par[v] = c
			}
			if porte {
				c.Portage++
			} else {
				c.Hors++
			}
		}
	}
	return t
}

// objDansFenetre dit si l'instant tombe dans une fenetre de portage.
func objDansFenetre(ws []objWindow, at int64) bool {
	for _, w := range ws {
		if at >= w.T0 && at <= w.T1 {
			return true
		}
	}
	return false
}

// objCandidat est une valeur retenue par la confrontation, avec ses deux taux.
type objCandidat struct {
	Val           uint32
	Portage, Hors int
	TauxP, TauxH  float64
}

// objSeuilHors : au-dela de ce taux hors portage, une valeur n'est pas un marqueur de
// portage. C'est le seuil TEMOIN du plan (<= 5 %), applique ici a la selection.
const objSeuilHors = 0.05

// objCandidats trie les valeurs par taux de portage decroissant, sous la contrainte du
// seuil hors portage.
func objCandidats(t objTable) []objCandidat {
	var out []objCandidat
	for v, c := range t.Par {
		if t.Portage == 0 {
			break
		}
		tp := float64(c.Portage) / float64(t.Portage)
		th := 0.0
		if t.Hors > 0 {
			th = float64(c.Hors) / float64(t.Hors)
		}
		if tp < 0.5 || th > objSeuilHors {
			continue
		}
		out = append(out, objCandidat{Val: v, Portage: c.Portage, Hors: c.Hors, TauxP: tp, TauxH: th})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TauxP != out[j].TauxP {
			return out[i].TauxP > out[j].TauxP
		}
		return out[i].Val < out[j].Val
	})
	return out
}

// TestObjectifsPhase0FamilleDrapeau — ITEM 0.1, sur les trois films CTF.
func TestObjectifsPhase0FamilleDrapeau(t *testing.T) {
	root := objRequireRoot(t)
	communs := map[uint32]int{}
	joues := 0
	for _, id := range objCTFFilms {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			t.Logf("film %s absent du cache — non confronte", id)
			continue
		}
		joues++
		cands := objMesureFilm(t, root, id, src)
		for _, c := range cands {
			communs[c.Val]++
		}
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", objFilmEnv, root)
	}
	var retenus []uint32
	for v, n := range communs {
		if n == joues {
			retenus = append(retenus, v)
		}
	}
	sort.Slice(retenus, func(i, j int) bool { return retenus[i] < retenus[j] })
	t.Logf("ITEM 0.1 — films confrontes : %d ; valeurs candidates COMMUNES aux %d films : %d %s",
		joues, joues, len(retenus), objHex(retenus))
}

// objMesureFilm joue la confrontation d'un film et publie ses chiffres.
func objMesureFilm(t *testing.T, root, id string, src *objDiskFilm) []objCandidat {
	t.Helper()
	f := objCorpus[id]
	evs, _, apparies := objIdentified(src, f)
	b, err := objBuildBridge(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : pont : %v", id, err)
	}
	recs, images, err := objScanKeyframeBipeds(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : balayage : %v", id, err)
	}
	wins, fusions := objPortageWindows(evs, b.Deaths, objFinMatch(evs, b.Deaths))
	tab := objConfronte(recs, b, wins)
	cands := objCandidats(tab)
	t.Logf("%s : %d images-cles, %d records bipede (%d sans pont), %d slots statborg apparies, "+
		"%d fenetres de portage (%d prises fusionnees) ; records ETIQUETES portage=%d hors=%d ; "+
		"valeurs distinctes=%d ; candidates=%d",
		id, images, len(recs), tab.SlotsInconnus, apparies, len(wins), fusions,
		tab.Portage, tab.Hors, len(tab.Par), len(cands))
	for i, c := range cands {
		if i >= 8 {
			t.Logf("%s : ... %d autres candidates", id, len(cands)-8)
			break
		}
		t.Logf("%s : candidate 0x%08X — portage %d/%d = %.1f %% ; hors portage %d/%d = %.2f %%",
			id, c.Val, c.Portage, tab.Portage, 100*c.TauxP, c.Hors, tab.Hors, 100*c.TauxH)
	}
	return cands
}

// objFinMatch borne la derniere fenetre : le dernier fait date du match.
func objFinMatch(evs []objectiveevents.IdentifiedEvent, deaths []Death) int64 {
	var fin int64
	for _, e := range evs {
		if int64(e.TimeMS) > fin {
			fin = int64(e.TimeMS)
		}
	}
	for _, d := range deaths {
		if d.TimeMS > fin {
			fin = d.TimeMS
		}
	}
	return fin
}

// objHex met en forme une liste de valeurs de 32 bits.
func objHex(vs []uint32) string {
	if len(vs) == 0 {
		return "[]"
	}
	s := "["
	for i, v := range vs {
		if i > 0 {
			s += " "
		}
		if i >= 12 {
			s += "..."
			break
		}
		s += fmt.Sprintf("0x%08X", v)
	}
	return s + "]"
}
