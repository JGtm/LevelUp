//go:build research

package grammar

// ecart108_residus_research_test.go — complements de `ecart108_research_test.go` : les entrees lues
// comme le jeu dans les ecarts qui ne sont pas des multiples de -108, et la queue de la table
// (0x1fff entrees) apres le dernier record de chaque paquet.
//
//	RI27C_FILMS=... RI27C_RACINE=... RI27C_OUT=... RI27C_CARTES=... \n//	  go test -tags=research -count=1 -run '^TestEcart108Residus$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// e108Decrire lit au plus `n` entrees depuis `pos` comme `FUN_142e2bfd0` et rend, pour chacune, sa
// classe et sa largeur ; elle s arrete a la premiere entree illisible (archetype hors table, corps
// desynchronise) en la nommant.
func e108Decrire(pay []byte, pos, n, slot0 int, reg *Registry, ctx ContexteDeLecture) (int, []string) {
	var out []string
	for j := range n {
		e, ok := e108Lire(pay, pos)
		if !ok {
			return pos, append(out, "hors_payload")
		}
		c := e.classe(slot0 + j)
		if e.arch == keyframeArchetypeNone {
			out = append(out, c+":108")
			pos += e108EnTete
			continue
		}
		if e.arch >= kfArchMax {
			return pos, append(out, c+":illisible")
		}
		tr := WalkKeyframeFullState(pay, pos, reg, ctx)
		if tr.DesyncAt >= 0 {
			return pos, append(out, fmt.Sprintf("%s:desync_i%d", c, tr.DesyncAt))
		}
		out = append(out, fmt.Sprintf("%s:%d", c, tr.EndBit-pos))
		pos = tr.EndBit
	}
	return pos, out
}

// TestEcart108Residus : (1) pour les records bipedes d ecart « autre_sous » des formats 24 a 27, les
// entrees lues comme le jeu entre la fin de la traversee et l ancre suivante ; (2) pour chaque paquet,
// les entrees des slots 0 a (slot de la premiere ancre - 1) lues depuis le bit 1. Sortie :
// ecart108_residus.tsv (lignes U et H).
func TestEcart108Residus(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre : %v", court, err)
		}
		vf, _ := FilmFormatVersion(fc.film)
		h := map[string]int{}
		for p, err := range fc.ImagesCles() {
			if err != nil {
				t.Fatalf("%s : %v", court, err)
			}
			ctx := fc.ContexteDeLecture()
			ctx.Obs = nil
			if len(p.Records) > 0 {
				r0 := &p.Records[0]
				pos, d := e108Decrire(p.Payload, keyframePrefixBits, int(r0.Vie.Slot), 0, reg, ctx)
				h[fmt.Sprintf("tete_atteinte=%v", pos == int(r0.Debut) && len(d) == int(r0.Vie.Slot))]++
				for _, s := range d {
					h["tete_entree:"+s]++
				}
			}
			e108Queue(h, p, reg, ctx)
			for i := 0; vf >= 24 && i+1 < len(p.Records); i++ {
				r, s := &p.Records[i], &p.Records[i+1]
				fin, want := int(r.Debut+r.Bits), int(s.Debut)
				if int(r.TI) != keyframeBipedTI || r.Desync < 0 && fin == want || r.Desync < 0 && fin < want && (want-fin)%e108EnTete == 0 {
					continue
				}
				if r.Desync >= 0 || fin > want {
					continue
				}
				es := int(s.Vie.Slot) - int(r.Vie.Slot) - 1
				pos, d := e108Decrire(p.Payload, fin, es, int(r.Vie.Slot)+1, reg, ctx)
				lignes = append(lignes, fmt.Sprintf("U\t%s\tf%d\t%d\t%d\t%d\t%d\t%d\t%v\t%s", court, vf, p.TS, r.Vie.Slot,
					s.Vie.Slot, want-fin, es, pos == want, strings.Join(d, ";")))
			}
		}
		lignes = append(lignes, ri27d0Trier(fmt.Sprintf("%s\tf%d", court, vf), "H", h)...)
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "ecart108_residus.tsv"), lignes)
}

// e108Queue lit, depuis la fin de la traversee du DERNIER record du paquet, les entrees des slots
// suivants jusqu au dernier de la table (0x1fff entrees, slots 0 a 0x1ffe, `FUN_1408be074`) comme le
// jeu, et dit ou elle finit par rapport a la fin du payload.
func e108Queue(h map[string]int, p *lecture.Paquet, reg *Registry, ctx ContexteDeLecture) {
	if len(p.Records) == 0 {
		return
	}
	r := &p.Records[len(p.Records)-1]
	if r.Desync != -1 {
		h["queue:dernier_non_traverse"]++
		return
	}
	n := 0x1fff - int(r.Vie.Slot) - 1
	pos, d := e108Decrire(p.Payload, int(r.Debut+r.Bits), n, int(r.Vie.Slot)+1, reg, ctx)
	reste := len(p.Payload)*8 - pos
	h[fmt.Sprintf("queueQ:dernier_slot=%d|lues=%d|n=%d|reste=%d|ff_en_tete=%v", r.Vie.Slot, len(d), n, reste, e108ToutesLiberees(d))]++
	switch {
	case len(d) == n && reste >= 0 && reste < 8:
		h["queue:fin_du_payload_a_moins_d_un_octet"]++
	case len(d) == n:
		h[fmt.Sprintf("queue:toutes_lues_reste_%d", reste)]++
	default:
		h["queue:arret_avant_la_fin"]++
	}
	for _, s := range d {
		if i := strings.LastIndexByte(s, ':'); i >= 0 {
			if _, err := fmt.Sscanf(s[i+1:], "%d", new(int)); err == nil {
				s = s[:i] + ":N"
			}
		}
		h["queue_entree:"+s]++
	}
}

// e108ToutesLiberees dit si toutes les entrees decrites sont des entrees liberees (id, archetype et
// mot +0xc a 0xffffffff, 108 bits).
func e108ToutesLiberees(d []string) bool {
	for _, s := range d {
		if !strings.HasPrefix(s, "idFF,archFF,motFF,f4=0,f8=00:") {
			return false
		}
	}
	return true
}
