//go:build research

package grammar

// r_nais_cadre_research_test.go — CHANTIER « NAIS » DE LA CAMPAGNE DE GRAMMAIRE (R-L1 (d),
// 2026-10-02) : LE BIT DE CONFIGURATION DU PAQUET ET LE MODELE D ALLOCATEUR QU IL DESIGNE.
//
// Lu (Ghidra, `FUN_142f2fc08`, `FUN_142f2f0cc`, `FUN_142f2f634`) : quand `DAT_144706104` vaut 0,
// l allocateur n a plus qu UN pool `[0, DAT_144706100)`, parcouru en next-fit depuis le curseur
// du pool 0 (`table+0x160`) ; il bascule a 0 quand un pool borne est plein. Le paquet delta ECRIT
// ce bit en tete (`FUN_14299d2c8` : `MOV R8B,[0x144706104]`), et le lecteur le relit
// (`FUN_142987460` : `DAT_144706104 = FUN_1406cf008(lecteur)`). C est donc une propriete LISIBLE
// de chaque paquet, pas un build.
//
// Mesures, par film : la repartition du bit 0 des paquets delta ; et, pour chaque NEW lu
// proprement par la marche de reference, son rang dans la suite predite par DEUX modeles
// d allocateur — cinq pools (`cmBlocs.predictions`, MESURES §T1-3) et pool unique (curseur du
// pool 0) — tete predite comprise.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestRNaisCadre$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// rnPredictionsUnique : le modele « pool unique » de l allocateur, next-fit sur `[0, n)` depuis
// le curseur du pool 0 (`FUN_142f2fc08` quand `DAT_144706104 == 0`), sans liberation pendant le
// chunk. Nil sans bloc.
func rnPredictionsUnique(b *cmBlocs, c int) map[uint32]int {
	e := b.entrees[c]
	if e == nil {
		return nil
	}
	out := map[uint32]int{}
	cur := int(b.queue[c][0])
	if cur < 0 || cur >= len(e) {
		cur = 0
	}
	rang := 0
	for k := 0; k < len(e) && rang < cmProfondeurPrediction; k++ {
		s := (cur + k) % len(e)
		if d := e[s].Drapeaux; d&1 == 0 || d&2 != 0 {
			out[uint32(s)] = rang //nolint:gosec // slot < 8191
			rang++
		}
	}
	return out
}

// rnClasseRang : « rang 0 », « rang 1-63 », « hors des 64 » ; tete predite ou non.
func rnClasseRang(b *cmBlocs, c int, pred map[uint32]int, eid uint32) string {
	if pred == nil {
		return "sans bloc"
	}
	slot, gen := eid&0x3fffffff, uint8(eid>>30) //nolint:gosec // deux bits
	r, ok := pred[slot]
	if !ok {
		return "hors des 64"
	}
	e, _ := b.entree(c, slot)
	tete := "tete predite"
	if (e.Gen+1)&3 != gen {
		tete = "tete autre"
	}
	if r == 0 {
		return "rang 0 · " + tete
	}
	return "rang 1-63 · " + tete
}

// rnCadre ecoute la marche de reference.
type rnCadre struct {
	b              *cmBlocs
	t              cmTables
	chunk          int
	bitChunk       string
	premierUnique  bool
	premierParPool map[int]bool
}

func (x *rnCadre) debutDeChunk(c int, data []byte, pks []FilmPacket, _ *World) {
	x.chunk, x.premierUnique, x.premierParPool = c, false, map[int]bool{}
	uns, zeros := 0, 0
	for _, pk := range pks {
		if pk.Type != PacketTypeDelta || pk.Size < 1 {
			continue
		}
		if pay := pk.Payload(data); len(pay) > 0 && pay[0]&0x80 != 0 {
			uns++
		} else {
			zeros++
		}
	}
	x.t.add("bit_cfg_paquets", "1", cmCompte{n: uns})
	x.t.add("bit_cfg_paquets", "0", cmCompte{n: zeros})
	switch {
	case uns > 0 && zeros > 0:
		x.bitChunk = "mixte"
	case uns > 0:
		x.bitChunk = "1"
	default:
		x.bitChunk = "0"
	}
	x.t.un("bit_cfg_chunks", x.bitChunk)
	q := x.b.queue[c]
	x.t.un("queue_pool0_dans_pool0", fmt.Sprintf("%v", int(q[0]) < cmBasesDePool[1]))
}

func (x *rnCadre) paquet(c int, p *cmPaquet, _ *World) {
	refuses := map[uint32]bool{}
	for _, rf := range p.refuses {
		refuses[rf.slot] = true
	}
	p5, p1 := x.b.predictions(c), rnPredictionsUnique(x.b, c)
	for _, r := range p.recs {
		if r.Type != recNew || r.DesyncAt >= 0 || refuses[r.Slot] {
			continue
		}
		k5, k1 := rnClasseRang(x.b, c, p5, r.ID), rnClasseRang(x.b, c, p1, r.ID)
		x.t.un("neufs_cinq_pools", cmJoindre("bit chunk "+x.bitChunk, k5))
		x.t.un("neufs_pool_unique", cmJoindre("bit chunk "+x.bitChunk, k1))
		if !x.premierUnique {
			x.premierUnique = true
			x.t.un("premier_neuf_pool_unique", cmJoindre("bit chunk "+x.bitChunk, k1))
		}
		if pool := cmPoolDe(r.Slot); !x.premierParPool[pool] {
			x.premierParPool[pool] = true
			x.t.un("premier_neuf_par_pool_cinq_pools", cmJoindre("bit chunk "+x.bitChunk, fmt.Sprintf("pool %d", pool), k5))
		}
	}
}

func (x *rnCadre) finDeFilm() {}

// TestRNaisCadre : le bit de configuration et les deux modeles d allocateur, film par film.
func TestRNaisCadre(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	out := []string{"film\tbuild\ttable\tcle\tn"}
	for _, id := range films {
		out = append(out, rnCadreUnFilm(t, racine, id, utiles)...)
	}
	b2Ecrire(t, sortie, "r_nais_cadre.tsv", out)
}

func rnCadreUnFilm(t *testing.T, racine, id string, utiles UsagesProduit) []string {
	garde := filmproc.Arm("r_nais/cadre", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	debut := time.Now()
	f, ok := cmOuvrir(t, racine, id, utiles)
	if !ok {
		return nil
	}
	b := cmLireBlocs(f)
	x := &rnCadre{b: b, t: cmTables{}}
	cmMarcher(f, cmVariante{}, x)
	var out []string
	for nom, m := range x.t {
		for _, k := range cmCles(m) {
			out = append(out, fmt.Sprintf("%s\t%s\t%s\t%s\t%d", id, f.build, nom, k, m[k].n))
		}
	}
	t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
	return out
}
