//go:build research

package grammar

// mouvement_5_26_handlers_research_test.go — LES TROIS HANDLERS QUE LE DEPOT NE LIT PAS
// (types 6, 0xb, 0xc) : RECENSEMENT, POSITION ET CONTENU (lot 5.26.1, mesure seule).
//
// CE QUE L ECRIVAIN DIT (Ghidra, lecture seule, `.ai/V7.5/film_re/NOTE_5_26_*`) :
//
//	type 6   `FUN_142988084` : lit le bloc entier (`FUN_142988338`, taille = en-tete + 4), pose un
//	         lecteur de bits (`FUN_1424c7b4c`, `FUN_1406d5cc0(.,3)` a zero bit) et lit R(32) ;
//	         la valeur passe a `FUN_1410de6c4` (un INDICE DE JOUEUR : borne par `+0x83c`, table de
//	         pas 0x458) et le repartiteur `FUN_1428e22c0` la range en `session+0x114`
//	         (`DAT_144c2328c`), avec `session+0x118 = 0`.
//	type 0xb `FUN_1429882c8` : alloue un tampon de la taille du bloc, le LIT, le LIBERE. Rien
//	         n est ecrit dans la session hors du curseur d octets.
//	type 0xc `FUN_1429875e4` : R(32) = un COMPTE, puis par entree R(32) x 3 et `FUN_1407eeba4`
//	         (largeur non lue ici : on s arrete a la premiere entree) ; l application passe par
//	         `FUN_142c26748` (« botHandle », table de participants de pas 0x610).
//
// La largeur du contenu n est lue QUE la ou l ecrivain la donne (R(32) des types 6 et 0xc) ;
// ailleurs on compte les octets.
//
// LA POSITION : pour chaque eid rejete (liste de `TestTicks525`, sous la marche de production
// du calque, UN decodage), combien de paquets 6 / 0xb / 0xc tombent ENTRE la derniere image-cle
// qui precede son premier rejet et ce premier rejet ; et, pour chaque paquet 6 / 0xb / 0xc, le
// paquet delta qui le suit est-il le premier rejet d un eid ?
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestHandlers526$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

// Les trois types de bloc que le repartiteur `FUN_1428e22c0` sert et que le depot ne lit pas.
const (
	h526TypeJoueur uint16 = 6
	h526TypeJete   uint16 = 0xb
	h526TypeBots   uint16 = 0xc
)

// h526Mot est la largeur lue par l ecrivain pour le mot de tete des types 6 et 0xc : R(32).
const h526Mot = 32

// h526Cle ordonne un paquet dans le flux : (chunk, rang dans le chunk).
type h526Cle struct{ chunk, idx int }

func (a h526Cle) avant(b h526Cle) bool {
	return a.chunk < b.chunk || (a.chunk == b.chunk && a.idx < b.idx)
}

// h526Bloc est un paquet de type 6 / 0xb / 0xc, mesure.
type h526Bloc struct {
	cle    h526Cle
	typ    uint16
	taille int
	ts     uint64
	// rangIC est le rang du paquet par rapport a l image-cle du chunk (negatif = avant elle,
	// 0 = pas d image-cle dans le chunk).
	rangIC int
	mot    int64 // R(32) de tete (types 6 et 0xc), -1 sinon
	hex    string
}

// h526Flux porte la population de paquets du film.
type h526Flux struct {
	parType   map[int][2]int // type -> (paquets, octets)
	blocs     []h526Bloc
	imagesCle []h526Cle
	deltas    int
}

// TestHandlers526 recense les paquets 6 / 0xb / 0xc et les place par rapport aux rejets.
func TestHandlers526(t *testing.T) {
	tc := t516Cadre(t)
	f := h526Recenser(tc)
	h526TableauTypes(t, f)
	h526TableauBlocs(t, f)
	p := t525Marcher(tc) // LE decodage de la mesure, sous la marche de production du calque
	h526Position(t, f, p)
}

// h526Recenser lit la population de paquets du film, sans decoder une trame.
func h526Recenser(tc t516Temoin) *h526Flux {
	f := &h526Flux{parType: map[int][2]int{}}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		ic := -1
		for _, pk := range pks {
			if pk.Type == PacketTypeKeyframe {
				ic = pk.Index
				f.imagesCle = append(f.imagesCle, h526Cle{c, pk.Index})
			}
		}
		for _, pk := range pks {
			n := f.parType[int(pk.Type)]
			f.parType[int(pk.Type)] = [2]int{n[0] + 1, n[1] + pk.Size}
			if pk.Type == PacketTypeDelta {
				f.deltas++
			}
			if pk.Type != h526TypeJoueur && pk.Type != h526TypeJete && pk.Type != h526TypeBots {
				continue
			}
			f.blocs = append(f.blocs, h526LireBloc(c, pk, pk.Payload(data), ic))
		}
	}
	return f
}

// h526LireBloc mesure un paquet 6 / 0xb / 0xc : le mot de tete la ou l ecrivain en donne la
// largeur, les octets sinon.
func h526LireBloc(c int, pk FilmPacket, pay []byte, ic int) h526Bloc {
	b := h526Bloc{cle: h526Cle{c, pk.Index}, typ: pk.Type, taille: pk.Size, ts: pk.TimestampUS,
		mot: -1}
	if ic >= 0 {
		b.rangIC = pk.Index - ic
	}
	if pk.Type != h526TypeJete && len(pay)*8 >= h526Mot {
		b.mot = int64(readBitsAt(pay, 0, h526Mot))
	}
	n := len(pay)
	if n > 16 {
		n = 16
	}
	b.hex = fmt.Sprintf("% x", pay[:n])
	return b
}

// h526TableauTypes publie la population de TOUS les types (controle contre le 5.19.3).
func h526TableauTypes(t *testing.T, f *h526Flux) {
	t.Helper()
	cles := make([]int, 0, len(f.parType))
	for k := range f.parType {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	t.Logf("(a) POPULATION DES TYPES DE PAQUET (controle 5.19.3) :")
	for _, k := range cles {
		v := f.parType[k]
		t.Logf("      type %2d : %6d paquets · %9d octets", k, v[0], v[1])
	}
}

// h526TableauBlocs publie, par type, les tailles, les rangs par rapport a l image-cle et les
// valeurs du mot de tete.
func h526TableauBlocs(t *testing.T, f *h526Flux) {
	t.Helper()
	for _, typ := range []uint16{h526TypeJoueur, h526TypeJete, h526TypeBots} {
		tailles, rangs, mots := map[int]int{}, map[int]int{}, map[int64]int{}
		n := 0
		var exemple string
		for _, b := range f.blocs {
			if b.typ != typ {
				continue
			}
			n++
			tailles[b.taille]++
			rangs[b.rangIC]++
			mots[b.mot]++
			if exemple == "" {
				exemple = b.hex
			}
		}
		t.Logf("(b) TYPE %d : %d paquets · tailles %s · rang par rapport a l image-cle %s · "+
			"R(32) de tete %s · premier contenu [%s]", typ, n, h526Hist(tailles), h526Hist(rangs),
			h526Hist64(mots), exemple)
	}
	for _, b := range f.blocs {
		t.Logf("      chunk %2d paquet %5d type %2d taille %4d ts %12d rangIC %+d mot %d",
			b.cle.chunk, b.cle.idx, b.typ, b.taille, b.ts, b.rangIC, b.mot)
	}
}

// h526Position place chaque premier rejet d eid par rapport aux paquets 6 / 0xb / 0xc.
func h526Position(t *testing.T, f *h526Flux, p *t525Passe) {
	t.Helper()
	premiers := map[uint32]h526Cle{}
	premierPaquet := map[h526Cle]int{}
	for _, tr := range p.tr {
		if !tr.rejet {
			continue
		}
		if _, vu := premiers[tr.rejetID]; vu {
			continue
		}
		k := h526Cle{tr.chunk, tr.paquet}
		premiers[tr.rejetID] = k
		premierPaquet[k]++
	}
	entre, sans, sansIC := 0, 0, 0
	for _, k := range premiers {
		ic, ok := h526ImageCleAvant(f.imagesCle, k)
		if !ok {
			sansIC++
			continue
		}
		if h526BlocsEntre(f.blocs, ic, k) > 0 {
			entre++
		} else {
			sans++
		}
	}
	t.Logf("(c) %d eid rejetes (premier rejet) : un paquet 6/0xb/0xc ENTRE la derniere "+
		"image-cle et le premier rejet %d · aucun %d · pas d image-cle avant %d",
		len(premiers), entre, sans, sansIC)
	h526Suivant(t, f, p, premierPaquet)
}

// h526Suivant dit, pour chaque paquet 6 / 0xb / 0xc, si le paquet delta qui le suit porte le
// premier rejet d un eid — et le taux de base sur tous les paquets delta.
func h526Suivant(t *testing.T, f *h526Flux, p *t525Passe, premierPaquet map[h526Cle]int) {
	t.Helper()
	ordre := make([]h526Cle, 0, len(p.tr))
	for _, tr := range p.tr {
		ordre = append(ordre, h526Cle{tr.chunk, tr.paquet})
	}
	suivi, suiviMemeChunk := 0, 0
	for _, b := range f.blocs {
		i := sort.Search(len(ordre), func(j int) bool { return b.cle.avant(ordre[j]) })
		if i >= len(ordre) {
			continue
		}
		if premierPaquet[ordre[i]] > 0 {
			suivi++
		}
		for j := i; j < len(ordre) && ordre[j].chunk == b.cle.chunk; j++ {
			if premierPaquet[ordre[j]] > 0 {
				suiviMemeChunk++
				break
			}
		}
	}
	t.Logf("    paquets 6/0xb/0xc : %d · suivis IMMEDIATEMENT (paquet delta suivant) d un premier "+
		"rejet %d · suivis d un premier rejet dans le meme chunk %d",
		len(f.blocs), suivi, suiviMemeChunk)
	t.Logf("    TAUX DE BASE : %d paquets delta portent un premier rejet sur %d (%.2f %%)",
		len(premierPaquet), len(ordre), m533bPart(len(premierPaquet), len(ordre)))
}

// h526ImageCleAvant rend la derniere image-cle strictement avant k.
func h526ImageCleAvant(ics []h526Cle, k h526Cle) (h526Cle, bool) {
	var out h526Cle
	ok := false
	for _, ic := range ics {
		if ic.avant(k) {
			out, ok = ic, true
		}
	}
	return out, ok
}

// h526BlocsEntre compte les paquets 6 / 0xb / 0xc strictement entre a et b.
func h526BlocsEntre(bs []h526Bloc, a, b h526Cle) int {
	n := 0
	for _, x := range bs {
		if a.avant(x.cle) && x.cle.avant(b) {
			n++
		}
	}
	return n
}

// h526Hist rend un histogramme trie, en ligne.
func h526Hist(m map[int]int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	s := ""
	for _, k := range cles {
		s += fmt.Sprintf(" %d:%d", k, m[k])
	}
	return "{" + s + " }"
}

// h526Hist64 : idem pour des cles 64 bits.
func h526Hist64(m map[int64]int) string {
	cles := make([]int64, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(a, b int) bool { return cles[a] < cles[b] })
	s := ""
	for _, k := range cles {
		s += fmt.Sprintf(" %d:%d", k, m[k])
	}
	return "{" + s + " }"
}
