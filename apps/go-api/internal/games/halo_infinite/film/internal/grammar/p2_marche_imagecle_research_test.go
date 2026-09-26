//go:build research

package grammar

// p2_marche_imagecle_research_test.go — SONDE P2 des retours rejeu du 2026-09-23 : LA MARCHE
// D'IMAGE-CLE QUI PERD UN PREFIXE DE TABLE.
//
// Constat de depart (RAPPORT_fiche_armes.md §1.3) : a l'image-cle de 81c02726 t 1494 (morceau 9),
// aucun record sous le slot 536 n'est reconnu, les bipedes vivants 519, 528, 530, 533, 534, 535
// manquent. Question : les records sont-ils DANS le payload, et par quel sous-mecanisme
// `WalkKeyframeWorld` les perd-il (depart refuse puis election lointaine, fausse ancre, entree
// sans archetype) ?
//
// CE QUE L'INSTRUMENT FAIT, SUR UN SEUL MORCEAU A LA FOIS (P2_FILM + P2_CHUNKS) :
//
//	A  RECENSEMENT des en-tetes EXACTS `[(gen<<30)|slot][35]` (deux mots pleins de 32 bits) a
//	   chaque position de bit du payload d'image-cle, avec `n1` (mot a +108) comme controle :
//	   n1 est la taille de tampon de l'archetype, constante par build.
//	B  TRACE du balayeur : copie INSTRUMENTEE de `walkKeyframeWorldFenetre` (depart, sauts de
//	   largeur, balayages, elections, arret), ETALONNEE : sa sortie doit egaler bit a bit celle
//	   de `WalkKeyframeWorld`, sinon le test echoue.
//	C  CROISEMENT : chaque en-tete bipede du recensement est-il une ancre du balayeur ? Sinon,
//	   quel pas l'a enjambe.
//	D  MARCHE DETERMINISTE (cadre de l'ecrivain, 108 bits, entrees sans archetype enchainees)
//	   depuis le bit 1 : ou s'arrete-t-elle, combien d'entrees sans archetype traverse-t-elle,
//	   atteint-elle les en-tetes bipedes.
//
// Lecture seule : chunk_00 et les morceaux demandes sont COPIES dans un repertoire temporaire
// pour que le contexte de film (profil du build) ne charge rien d'autre. Aucun artefact ecrit.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const p2BipedeTI = 35

// p2Cand est un candidat vu par un balayage (ordre des bits).
type p2Cand struct{ bit, slot, gen int }

// p2Scan est le resultat INSTRUMENTE de `kfScanNext`.
type p2Scan struct {
	at     int
	rapide bool     // retour immediat sur le voisin (slot prev+1, gen 1)
	cands  int      // candidats valides vus
	premie []p2Cand // les premiers candidats vus, par bit croissant
	elu    p2Cand
}

// p2ScanNext est `kfScanNext` a l'identique, plus le releve des candidats.
func p2ScanNext(buf []byte, from, prevSlot, total, maxWin int) p2Scan {
	r := p2Scan{at: -1}
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q+64 <= end; q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				break
			}
			continue
		}
		sentStreak = 0
		s, _, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		r.cands++
		if len(r.premie) < 8 {
			r.premie = append(r.premie, p2Cand{q, s, g})
		}
		if s == prevSlot+1 && g == 1 {
			r.at, r.rapide, r.elu = q, true, p2Cand{q, s, g}
			return r
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if r.at < 0 || cand.betterThan(best) {
			r.at, best = q, cand
			r.elu = p2Cand{q, s, g}
		}
	}
	return r
}

// p2Pas est un pas du balayeur trace.
type p2Pas struct {
	pos, slot, ti, gen int
	methode            string // "saut-largeur" | "balayage"
	nat                int
	scan               *p2Scan
}

// p2Trace rejoue `walkKeyframeWorldFenetre` (fenetre de production) en journalisant.
func p2Trace(buf []byte) (recs []KeyframeRec, depart *p2Scan, pas []p2Pas) {
	maxWin, total := kfScanFenetreBits, len(buf)*8
	width, seen := map[int]int{}, map[int]int{}
	pos, prev := 1, -1
	if _, _, _, ok := kfValidAnchor(buf, pos, prev, total); !ok {
		s := p2ScanNext(buf, pos, prev, total, maxWin)
		depart, pos = &s, s.at
	}
	for pos >= 0 {
		slot, ti, gen, ok := kfValidAnchor(buf, pos, prev, total)
		if !ok {
			break
		}
		startState := pos + 64
		p := p2Pas{pos: pos, slot: slot, ti: ti, gen: gen, methode: "balayage"}
		w, has := width[ti]
		if _, _, jg, vok := kfValidAnchor(buf, startState+w, slot, total); has && vok && jg == 1 {
			p.methode, p.nat = "saut-largeur", startState+w
		} else {
			s := p2ScanNext(buf, startState, slot, total, maxWin)
			p.scan, p.nat = &s, s.at
		}
		recs = append(recs, KeyframeRec{Slot: slot, TI: ti, Gen: gen, Bit: pos})
		pas = append(pas, p)
		prev = slot
		if p.nat < 0 {
			break
		}
		nw := p.nat - startState
		if prevW, s := seen[ti]; s {
			if prevW == nw {
				width[ti] = nw
			} else {
				delete(width, ti)
			}
		} else {
			seen[ti] = nw
		}
		pos = p.nat
	}
	return recs, depart, pas
}

// p2Entete est un en-tete exact trouve par le recensement.
type p2Entete struct {
	bit, slot, gen int
	n1             uint64
}

// p2Recenser rend les en-tetes exacts `[(gen<<30)|slot][ti]` du payload (gen 1..3).
func p2Recenser(pay []byte, ti uint64) []p2Entete {
	total := len(pay) * 8
	var out []p2Entete
	for q := 0; q+172 <= total; q++ {
		if kfReadBits(pay, q+32, 32) != ti {
			continue
		}
		id := kfReadBits(pay, q, 32)
		gen, slot := int(id>>30), int(id&0x3FFFFFFF)
		if gen == 0 || slot >= kfTableCap {
			continue
		}
		out = append(out, p2Entete{bit: q, slot: slot, gen: gen, n1: kfReadBits(pay, q+108, 32)})
	}
	return out
}

// p2PasCouvrant rend l'index du pas dont l'emprise [pos, nat) contient `bit`, -1 si aucun
// (bit avant la premiere ancre).
func p2PasCouvrant(pas []p2Pas, bit int) int {
	for i, p := range pas {
		fin := p.nat
		if fin < 0 {
			fin = 1 << 62
		}
		if bit >= p.pos && bit < fin {
			return i
		}
	}
	return -1
}

func p2Env(t *testing.T) (film string, chunks []int) {
	t.Helper()
	film = os.Getenv("P2_FILM")
	if film == "" {
		t.Skip("P2_FILM absent : sonde sautee")
	}
	for _, s := range strings.Split(os.Getenv("P2_CHUNKS"), ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			chunks = append(chunks, n)
		}
	}
	if len(chunks) == 0 {
		t.Fatal("P2_CHUNKS vide")
	}
	return film, chunks
}

// p2MiniFilm copie chunk_00 et UN morceau dans un repertoire temporaire et rend son contexte.
func p2MiniFilm(t *testing.T, film string, c int) (*FilmContext, *Registry, []byte) {
	t.Helper()
	tmp := t.TempDir()
	for _, n := range []int{0, c} {
		nom := fmt.Sprintf("chunk_%02d.bin", n)
		b, err := os.ReadFile(filepath.Join(film, nom))
		if err != nil {
			t.Fatalf("lecture %s : %v", nom, err)
		}
		if err := os.WriteFile(filepath.Join(tmp, nom), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fc, _, err := ContexteDeFilm(tmp)
	if err != nil {
		t.Logf("decoupage i0 non detecte (%v) : contexte par defaut", err)
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	data, err := ReadFilmChunk(tmp, c)
	if err != nil {
		t.Fatal(err)
	}
	return fc, reg, data
}

func TestP2MarcheImageCle(t *testing.T) {
	film, chunks := p2Env(t)
	for _, c := range chunks {
		fc, reg, data := p2MiniFilm(t, film, c)
		if restore, err := InstallFilmFormatMPP(fc); err == nil {
			defer restore()
		} else {
			t.Logf("MPP du build non installe : %v", err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			t.Logf("=== %s morceau %d paquet %d : ts %d ms, %d bits ===",
				filepath.Base(film), c, pk.Index, pk.TimestampUS/1000, len(pay)*8)
			p2UnPayload(t, fc, reg, pay)
		}
	}
}

func p2UnPayload(t *testing.T, fc *FilmContext, reg *Registry, pay []byte) {
	t.Helper()
	recs, depart, pas := p2Trace(pay)
	prod := WalkKeyframeWorld(pay)
	if fmt.Sprint(prod) != fmt.Sprint(recs) {
		t.Fatalf("ETALONNAGE : la trace (%d) differe de WalkKeyframeWorld (%d)", len(recs), len(prod))
	}
	t.Logf("B. etalonnage OK : %d ancres identiques a WalkKeyframeWorld", len(recs))
	p2Depart(t, pay, depart, recs)
	bip := p2Recenser(pay, p2BipedeTI)
	p2Croiser(t, bip, recs, pas)
	p2Elections(t, pas)
	p2Deterministe(t, fc, reg, pay, bip, recs)
	p2Trous(t, fc, pay, pas)
	p2Prototype(t, fc, pay, bip, recs)
	p2Variantes(t, pay, bip, recs)
	if os.Getenv("P2_DETAIL") == "81c02726-9" { // fenetres propres au cas rapporte
		p2Ordre(t, pay, recs, 140000, 240000)
		p2Voisinage(t, reg, pay, pas, 110, 135)
		p2Fermeture(t, fc.ContexteDeLecture(), reg, pay, pas, 100, 135, 143972, 206882)
	} else {
		p2Ordre(t, pay, recs, 0, 0)
	}
}

// p2Depart publie le premier pas : l'en-tete au bit 1 et, s'il est refuse, l'election.
func p2Depart(t *testing.T, pay []byte, depart *p2Scan, recs []KeyframeRec) {
	t.Helper()
	h, ok := readKeyframeHeader(pay, 1, len(pay)*8)
	t.Logf("B. bit 1 : id=%#08x arch=%#08x (readKeyframeHeader ok=%v slot=%d gen=%d sansArch=%v)",
		kfReadBits(pay, 1, 32), kfReadBits(pay, 33, 32), ok, h.Slot, h.Gen, h.SansArchetype())
	if depart == nil {
		t.Logf("B. DEPART ACCEPTE au bit 1")
	} else {
		t.Logf("B. DEPART REFUSE au bit 1 -> election : elu bit %d slot %d gen %d parmi %d candidats ; premiers %v",
			depart.elu.bit, depart.elu.slot, depart.elu.gen, depart.cands, depart.premie)
	}
	n := len(recs)
	if n > 12 {
		n = 12
	}
	t.Logf("B. premieres ancres : %v", recs[:n])
}

// p2Croiser publie le recensement bipede et, pour chaque en-tete manque, le pas qui l'enjambe.
func p2Croiser(t *testing.T, bip []p2Entete, recs []KeyframeRec, pas []p2Pas) {
	t.Helper()
	ancre := map[int]bool{}
	for _, r := range recs {
		ancre[r.Bit] = true
	}
	n1 := map[uint64]int{}
	for _, e := range bip {
		n1[e.n1]++
	}
	t.Logf("A. %d en-tetes exacts [id][35] ; n1 : %v", len(bip), n1)
	var manques int
	for _, e := range bip {
		etat := "ANCRE"
		if !ancre[e.bit] {
			manques++
			etat = "MANQUE"
			if i := p2PasCouvrant(pas, e.bit); i < 0 {
				etat += " (avant la premiere ancre)"
			} else {
				p := pas[i]
				etat += fmt.Sprintf(" (enjambe par le pas %d : ancre slot %d ti %d bit %d -> %d, %s)",
					i, p.slot, p.ti, p.pos, p.nat, p.methode)
			}
		}
		t.Logf("A.   bit %8d slot %4d gen %d n1 %d : %s", e.bit, e.slot, e.gen, e.n1, etat)
	}
	t.Logf("C. bipedes : %d en-tetes, %d ancres, %d MANQUES", len(bip), len(bip)-manques, manques)
}

// p2Elections publie les pas qui ne sont ni un saut de largeur ni un voisin immediat.
func p2Elections(t *testing.T, pas []p2Pas) {
	t.Helper()
	var n, rapides, sauts int
	for i, p := range pas {
		switch {
		case p.scan == nil:
			sauts++
		case p.scan.rapide:
			rapides++
		default:
			n++
			t.Logf("B. ELECTION pas %d : depuis slot %d ti %d (bit %d) -> elu slot %d gen %d bit %d (+%d bits) parmi %d ; premiers %v",
				i, p.slot, p.ti, p.pos, p.scan.elu.slot, p.scan.elu.gen, p.scan.elu.bit, p.nat-p.pos-64,
				p.scan.cands, p.scan.premie)
		}
	}
	t.Logf("B. %d pas : %d sauts de largeur, %d voisins immediats, %d elections", len(pas), sauts, rapides, n)
}

// p2Deterministe marche la table au cadre de l'ecrivain depuis le bit 1 et publie ce qu'elle
// atteint.
func p2Deterministe(t *testing.T, fc *FilmContext, reg *Registry, pay []byte, bip []p2Entete,
	recs []KeyframeRec) {
	t.Helper()
	ctx := fc.ContexteDeLecture()
	det, stop := WalkKeyframeRecords(pay, reg, ctx)
	var sa int
	atteint := map[int]bool{}
	for _, r := range det {
		atteint[r.BitStart] = true
		if r.SansArchetype() {
			sa++
		}
	}
	fin := 0
	if len(det) > 0 {
		fin = det[len(det)-1].BitEnd
	}
	t.Logf("D. marche deterministe : %d records (%d SANS ARCHETYPE), arret %s, derniere fin bit %d",
		len(det), sa, stop, fin)
	for i, r := range det {
		if i >= 40 {
			break
		}
		t.Logf("D.   #%d bit %d-%d slot %d gen %d arch %#x desync %d", i, r.BitStart, r.BitEnd, r.Slot, r.Gen,
			r.Archetype, r.DesyncAt)
	}
	var nb int
	for _, e := range bip {
		if atteint[e.bit] {
			nb++
		}
	}
	t.Logf("D. en-tetes bipedes atteints par la marche deterministe : %d / %d", nb, len(bip))
	p2Fermetures(t, ctx, reg, pay, bip, recs)
}

// p2Fermetures : depuis chaque en-tete bipede, l'etat complet ferme-t-il sur un en-tete valide ?
// Et combien d'entrees sans archetype suivent (chainage de 108 bits) avant le record suivant ?
func p2Fermetures(t *testing.T, ctx ContexteDeLecture, reg *Registry, pay []byte, bip []p2Entete,
	recs []KeyframeRec) {
	t.Helper()
	total := len(pay) * 8
	starts := make([]int, 0, len(recs))
	for _, r := range recs {
		starts = append(starts, r.Bit)
	}
	sort.Ints(starts)
	for _, e := range bip {
		tr := WalkKeyframeFullState(pay, e.bit, reg, ctx)
		h, ok := readKeyframeHeader(pay, tr.EndBit, total)
		chaine := ""
		pos, prev, sa := tr.EndBit, e.slot, 0
		for ok && tr.DesyncAt < 0 && h.Slot > prev {
			if !h.SansArchetype() {
				chaine = fmt.Sprintf("-> record slot %d ti %d bit %d apres %d entree(s) sans archetype", h.Slot, h.TI, pos, sa)
				break
			}
			sa++
			prev, pos = h.Slot, pos+keyframeFullHeaderBits(ctx)
			h, ok = readKeyframeHeader(pay, pos, total)
		}
		t.Logf("D.   bipede slot %d bit %d : fin etat complet %d desync %d en-tete suivant ok=%v slot %d arch %#x %s",
			e.slot, e.bit, tr.EndBit, tr.DesyncAt, ok, h.Slot, h.Archetype, chaine)
	}
}
