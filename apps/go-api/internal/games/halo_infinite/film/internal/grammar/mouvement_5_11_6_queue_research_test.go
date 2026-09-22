//go:build research

package grammar

// mouvement_5_11_6_queue_research_test.go — CE QUE LA MARCHE NE LIT PAS (lot 5.11.6).
//
// # POURQUOI CE FICHIER EXISTE
//
// Le lot 5.11 a conclu que la rafale du saut de `dad793c7` ne porte que `i0`, `i1` et `i25`.
// L UTILISATEUR REFUSE LA CONCLUSION : « un saut est un saut, je ne me contenterai pas d un
// derive ». Il fait autorite, et sa lecture est juste sur un point de METHODE : la mesure du
// 5.11 a regarde CE QUE LA MARCHE LIT, pas ce qu elle LAISSE. Or elle laisse trois choses, et
// chacune est un endroit ou un champ peut vivre :
//
//	LA QUEUE DE PAQUET — 6,3 bits par paquet en moyenne sur ce film, non lus apres le dernier
//	  record. Des bits ECRITS par le jeu que personne ne regarde.
//	LES PAQUETS NON LOCALISES — ceux dont la liste d evenements n est pas cadree et que la
//	  marche ABANDONNE en entier.
//	LES AUTRES ARCHETYPES — le 5.11.2 a croise tous les archetypes par COMPOSANT DECLARE, mais
//	  n a pas dit quels SLOTS ecrivent dans la fenetre, ni si une image-cle y tombe.
//
// Chaque test rend « trouve : champ X » ou « non trouve, voici ce qui a ete lu ». Jamais un
// negatif nu.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_CARTE=<carte> MOUV511_BORNES=<catalogue> \
//	  MOUV511_T0=26.0 MOUV511_T1=27.2 \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement5116' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m5116Paquet est UN paquet delta marche, avec ce que la marche en a lu et ce qu elle a laisse.
type m5116Paquet struct {
	chunk, index int
	rel          float64
	bitsTotal    int
	debut        int // premier bit du premier record (2, ou la localisation de la liste)
	finRecords   int // EndBit du dernier record rendu
	nRecords     int
	nBiped       int
	evType       int
	evPresent    bool
	localise     bool
	queue        string // les bits NON LUS, en binaire MSB-first
	queueBits    int
}

// m5116Passe marche le film et rend un descripteur par paquet delta. Elle N ABANDONNE PAS les
// paquets non localises : elle les garde avec `localise=false`, parce que c est exactement la
// population que le lot 5.11 n a pas regardee.
func m5116Passe(t *testing.T) []m5116Paquet {
	t.Helper()
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	var out []m5116Paquet
	var t0 uint64
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if t0 == 0 {
				t0 = pk.TimestampUS
			}
			out = append(out, m5116UnPaquet(pk, pk.Payload(data), c, t0, w, cfg))
		}
	}
	return out
}

// m5116UnPaquet marche UN paquet et rend son descripteur.
func m5116UnPaquet(pk FilmPacket, pay []byte, c int, t0 uint64, w *World,
	cfg FrameConfig) m5116Paquet {
	p := m5116Paquet{chunk: c, index: pk.Index,
		rel: float64(pk.TimestampUS-t0) / 1e6, bitsTotal: len(pay) * 8, debut: 2,
		localise: true}
	p.evType, p.evPresent = PacketHeadEventType(pay)
	if p.evPresent {
		if p.debut = marchLocateStrict(pay, w, cfg); p.debut < 0 {
			p.localise, p.debut = false, 0
			p.queueBits = p.bitsTotal
			p.queue = m511Bits(pay, 0, 160)
			return p
		}
	}
	recs, _ := DecodeFrameViews(pay, w, cfg, 3, p.debut)
	p.nRecords = len(recs)
	p.finRecords = p.debut
	for _, r := range recs {
		if r.Trace.EndBit > p.finRecords {
			p.finRecords = r.Trace.EndBit
		}
		if r.TypeIndex == BipedTypeIndex {
			p.nBiped++
		}
	}
	p.queueBits = p.bitsTotal - p.finRecords
	if p.queueBits > 0 {
		p.queue = m511Bits(pay, p.finRecords, p.queueBits)
	}
	return p
}

// TestMouvement5116Queue — POINT 1 : LES BITS NON LUS EN QUEUE DE PAQUET.
//
// Un champ vit dans la queue s il VARIE dans la fenetre du saut et pas dehors. Le test publie la
// distribution des largeurs et des VALEURS de queue, dedans et dehors, puis dumpe la fenetre.
func TestMouvement5116Queue(t *testing.T) {
	paquets := m5116Passe(t)
	a, b := m511Bornes()
	larDedans, larHors := map[int]int{}, map[int]int{}
	valDedans, valHors := map[string]int{}, map[string]int{}
	var dedans, hors int
	for _, p := range paquets {
		if !p.localise {
			continue
		}
		if p.rel >= a && p.rel <= b {
			dedans++
			larDedans[p.queueBits]++
			valDedans[p.queue]++
		} else {
			hors++
			larHors[p.queueBits]++
			valHors[p.queue]++
		}
	}
	t.Logf("FENETRE [%.3f ; %.3f] s — %d paquets localises dedans, %d dehors", a, b, dedans, hors)
	t.Logf("LARGEUR DE LA QUEUE (bits non lus apres le dernier record) :")
	m5116Histo(t, "  dedans", larDedans, dedans)
	m5116Histo(t, "  hors  ", larHors, hors)
	var seulDedans []string
	for v, n := range valDedans {
		if valHors[v] == 0 {
			seulDedans = append(seulDedans, fmt.Sprintf("%q x%d", v, n))
		}
	}
	sort.Strings(seulDedans)
	if len(seulDedans) == 0 {
		t.Logf("VALEURS DE QUEUE : AUCUNE n est exclusive a la fenetre (%d valeurs distinctes "+
			"dedans, %d dehors). Voici ce qui a ete lu :", len(valDedans), len(valHors))
	} else {
		t.Logf("VALEURS DE QUEUE : %d EXCLUSIVE(S) a la fenetre — CANDIDATS : %s",
			len(seulDedans), strings.Join(seulDedans, " · "))
	}
	m5116Top(t, "  valeurs dedans", valDedans, 8)
	m5116Top(t, "  valeurs hors  ", valHors, 8)
	t.Logf("DUMP DE LA FENETRE, paquet par paquet :")
	var n int
	for _, p := range paquets {
		if p.rel < a || p.rel > b || !p.localise {
			continue
		}
		n++
		if n > 40 {
			t.Logf("  ... (%d paquets de plus)", dedans-40)
			break
		}
		t.Logf("  t=%8.3f s · %4d bits · debut %3d · fin records %4d · %d records (%d bipede) "+
			"· queue %d bits = %q", p.rel, p.bitsTotal, p.debut, p.finRecords, p.nRecords,
			p.nBiped, p.queueBits, p.queue)
	}
}

// m5116Histo publie un histogramme trie par cle.
func m5116Histo(t *testing.T, titre string, h map[int]int, total int) {
	t.Helper()
	cles := make([]int, 0, len(h))
	for k := range h {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var parts []string
	for _, k := range cles {
		parts = append(parts, fmt.Sprintf("%d bits : %d (%.1f %%)", k, h[k],
			m533bPart(h[k], total)))
	}
	t.Logf("%s : %s", titre, strings.Join(parts, " · "))
}

// m5116Top publie les `n` valeurs les plus frequentes.
func m5116Top(t *testing.T, titre string, h map[string]int, n int) {
	t.Helper()
	type kv struct {
		v string
		n int
	}
	l := make([]kv, 0, len(h))
	for v, c := range h {
		l = append(l, kv{v: v, n: c})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].v < l[j].v
	})
	var parts []string
	for i, x := range l {
		if i >= n {
			break
		}
		parts = append(parts, fmt.Sprintf("%q x%d", x.v, x.n))
	}
	t.Logf("%s (%d distinctes) : %s", titre, len(h), strings.Join(parts, " · "))
}

// TestMouvement5116NonLocalises — POINT 2 : LES PAQUETS QUE LA MARCHE ABANDONNE EN ENTIER.
func TestMouvement5116NonLocalises(t *testing.T) {
	paquets := m5116Passe(t)
	a, b := m511Bornes()
	var dedans, hors []m5116Paquet
	var totalLoc, totalNon int
	for _, p := range paquets {
		if p.localise {
			totalLoc++
			continue
		}
		totalNon++
		if p.rel >= a && p.rel <= b {
			dedans = append(dedans, p)
		} else {
			hors = append(hors, p)
		}
	}
	t.Logf("PAQUETS : %d localises, %d NON localises (%d dans la fenetre [%.3f ; %.3f], "+
		"%d dehors)", totalLoc, totalNon, len(dedans), a, b, len(hors))
	if len(dedans) == 0 {
		t.Logf("  AUCUN paquet non localise dans la fenetre : la marche a lu TOUS les paquets " +
			"du saut. Le champ cherche n est pas dans cette population.")
	}
	for _, p := range dedans {
		t.Logf("  DEDANS t=%8.3f s · chunk %d paquet %d · %d bits · evenement de tete type %d "+
			"· tete %s", p.rel, p.chunk, p.index, p.bitsTotal, p.evType, p.queue)
	}
	for i, p := range hors {
		if i >= 12 {
			t.Logf("  ... (%d de plus hors fenetre)", len(hors)-12)
			break
		}
		t.Logf("  hors   t=%8.3f s · chunk %d paquet %d · %d bits · evenement de tete type %d "+
			"· tete %s", p.rel, p.chunk, p.index, p.bitsTotal, p.evType, p.queue)
	}
}
