package filmdec

// mesure_entete_ti9_identite_test.go — MESURE M, VOLET 3 : UNE ENTITE ti=9 PORTE-T-ELLE DE
// QUOI LA RATTACHER A UN JOUEUR ?
//
// POURQUOI CE VOLET. Le volet 1 (mesure_entete_ti9_test.go) rend huit designateurs d'equipe
// stables repartis 4-4 — mais HUIT EQUIPES SANS IDENTITE ne servent a rien : sans le lien
// entite -> joueur, on ne peut pas dire QUI est dans l'equipe 0. Le fork n'a pas resolu ce
// lien. Ce banc regarde les deux seuls endroits ou il pourrait se trouver sans nouvelle
// retro-ingenierie : l'etat par defaut de ti=9, et l'ordre des slots.
//
// CE QU'IL LIT. `consumeDefaultStateTI9` porte FUN_1410d7540 = V ; R(6) ; R(6) ; R(1). Les
// deux champs de 6 bits sont assez larges pour un index de joueur (0..63) et assez etroits
// pour ne pas etre autre chose. Le banc les relit BIT A BIT au meme offset que la marche
// (en-tete 47), sans passer par le deserialiseur : il veut les VALEURS, que le deserialiseur
// de production jette.
//
// LANCEMENT : memes gardes que le volet 1.
//
//	MESURE_TI9_ROOT=<...>/data/cache/film_chunks MESURE_TI9_IDS=a,b,c \
//	  go test ./internal/analysis/filmdec/ -run MesureIdentiteTI9 -v -timeout 60m

import (
	"sort"
	"testing"
)

// mesureIdentiteChamps porte les champs de l'etat par defaut de ti=9 pour UN record.
type mesureIdentiteChamps struct {
	version bool
	f1, f2  uint64 // les deux R(6)
	flag    bool   // le R(1) terminal
}

// mesureLitDefautTI9 relit l'etat par defaut de ti=9 a partir du premier bit qui le suit
// l'en-tete de `mesureTI9These` bits. Grammaire : V [R(8) si V] R(6) R(6) R(1).
func mesureLitDefautTI9(pay []byte, recBit int) mesureIdentiteChamps {
	p := recBit + mesureTI9These
	var c mesureIdentiteChamps
	c.version = kfReadBits(pay, p, 1) == 1
	p++
	if c.version {
		p += 8
	}
	c.f1 = kfReadBits(pay, p, 6)
	c.f2 = kfReadBits(pay, p+6, 6)
	c.flag = kfReadBits(pay, p+12, 1) == 1
	return c
}

// mesureIdentiteEntite rassemble ce qu'on sait d'une entite ti=9 sur tout un film.
type mesureIdentiteEntite struct {
	lectures int
	equipes  map[uint64]int
	f1       map[uint64]int
	f2       map[uint64]int
	flags    map[bool]int
}

func nouvelleIdentite() *mesureIdentiteEntite {
	return &mesureIdentiteEntite{
		equipes: map[uint64]int{}, f1: map[uint64]int{}, f2: map[uint64]int{},
		flags: map[bool]int{},
	}
}

func TestMesureIdentiteTI9(t *testing.T) {
	films := mesureOuvre(t)
	defer LockProcessDecode()()

	for _, f := range films {
		ents := map[int]*mesureIdentiteEntite{}
		voisins := map[int]map[int]int{} // slot -> TI -> occurrences
		mesureParcours(f, func(_ int, pay []byte, r KeyframeRec) {
			if voisins[r.Slot] == nil {
				voisins[r.Slot] = map[int]int{}
			}
			voisins[r.Slot][r.TI]++
			if r.TI != 9 {
				return
			}
			_, vals, ok := mesureMarche(pay, r, f.reg, mesureTI9These)
			if !ok || len(vals) == 0 {
				return
			}
			e := ents[r.Slot]
			if e == nil {
				e = nouvelleIdentite()
				ents[r.Slot] = e
			}
			e.lectures++
			e.equipes[vals[0]]++
			c := mesureLitDefautTI9(pay, r.Bit)
			e.f1[c.f1]++
			e.f2[c.f2]++
			e.flags[c.flag]++
		})
		mesureRapportIdentite(t, f, ents)
		mesureRapportVoisins(t, f, ents, voisins)
	}
}

// mesureRapportVoisins publie le VOISINAGE de slots des entites ti=9. Les huit slots sont
// consecutifs de deux en deux : ce qui occupe les slots intercales est le premier candidat
// au lien entite -> joueur, et se lit sans nouvelle retro-ingenierie.
func mesureRapportVoisins(t *testing.T, f mesureFilm, ents map[int]*mesureIdentiteEntite,
	voisins map[int]map[int]int) {
	t.Helper()
	lo, hi := -1, -1
	for s := range ents {
		if lo < 0 || s < lo {
			lo = s
		}
		if s > hi {
			hi = s
		}
	}
	if lo < 0 {
		return
	}
	t.Logf("=== F. VOISINAGE DE SLOTS DES ENTITES ti=9 — film %s ===", f.id)
	t.Logf("%8s %10s %s", "slot", "records", "types d'entite vus")
	vus := map[int]bool{}
	for s := lo - 4; s <= hi+4; s++ {
		if voisins[s] == nil {
			continue
		}
		tis := make([]uint64, 0, len(voisins[s]))
		hist := map[uint64]int{}
		for ti, n := range voisins[s] {
			tis = append(tis, uint64(ti))
			hist[uint64(ti)] = n
		}
		total := 0
		for _, n := range voisins[s] {
			total += n
		}
		t.Logf("%8d %10d ti %s", s, total, mesureHisto(hist))
		for _, ti := range tis {
			vus[int(ti)] = true
		}
	}
	types := make([]int, 0, len(vus))
	for ti := range vus {
		types = append(types, ti)
	}
	sort.Ints(types)
	t.Logf("archetypes du voisinage (premiers composants, pour les nommer) :")
	for _, ti := range types {
		arch, ok := f.reg.Archetype(ti)
		if !ok {
			continue
		}
		n := len(arch.Components)
		if n > 3 {
			n = 3
		}
		t.Logf("  ti=%2d (%d composants) : %v", ti, len(arch.Components), arch.Components[:n])
	}
}

func mesureRapportIdentite(t *testing.T, f mesureFilm, ents map[int]*mesureIdentiteEntite) {
	t.Helper()
	t.Logf("")
	t.Logf("=== E. CHAMPS DE L'ETAT PAR DEFAUT ti=9 — film %s ===", f.id)
	if arch, ok := f.reg.Archetype(9); ok {
		t.Logf("archetype ti=9 : %d composants nommes, i0=%q", len(arch.Components), arch.Components[0])
	}
	t.Logf("%8s %10s %8s %18s %18s %14s", "slot", "lectures", "equipe", "champ1 R(6)",
		"champ2 R(6)", "drapeau R(1)")
	slots := make([]int, 0, len(ents))
	for s := range ents {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	for _, s := range slots {
		e := ents[s]
		t.Logf("%8d %10d %8s %18s %18s %14s", s, e.lectures, mesureHisto(e.equipes),
			mesureHisto(e.f1), mesureHisto(e.f2), mesureHistoBool(e.flags))
	}
}

func mesureHistoBool(m map[bool]int) string {
	out := map[uint64]int{}
	for k, v := range m {
		if k {
			out[1] = v
			continue
		}
		out[0] = v
	}
	return mesureHisto(out)
}
