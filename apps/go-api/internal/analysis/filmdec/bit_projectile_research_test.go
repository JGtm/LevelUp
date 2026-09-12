package filmdec

// bit_projectile_research_test.go — INSTRUMENT BB.1 : du point publié aberrant jusqu'aux BITS.
//
// LA QUESTION. Le lot B a mesuré que 947 trajectoires de projectile sur 15 735 portent un pas
// impossible, et que ce pas vaut l'étendue de la carte sur un axe DIVISÉE PAR UNE PUISSANCE DE
// DEUX — le poids d'UN bit du champ quantifié. Il restait à dire QUEL bit, et pourquoi.
//
// CE QUE L'INSTRUMENT FAIT. Il rejoue le balayage des records d'objet du monde en gardant, pour
// chaque échantillon, la POSITION DE BIT du début d'i0, les bits de PORTE et le quantum BRUT de
// chaque axe. Il le fait DEUX FOIS sur le même film : avec la porte ANCIENNE (3 bits, le
// littéral de `decodeWorldObjectPos` AVANT le correctif BB.2) et avec la porte que le CATALOGUE impose
// (2 + regionIndexBits). Puis il compte les pas impossibles de chaque lecture et publie, pour
// les premiers d'entre eux, les bits bruts des deux points consécutifs.
//
// POURQUOI CETTE COMPARAISON TRANCHE. Les deux lectures partent du MÊME bit et lisent le MÊME
// flux : la seule différence est le nombre de bits de porte consommés avant l'axe X. Si la
// porte de production est trop courte d'un bit, chaque axe est lu un bit trop tôt, et le bit de
// POIDS FORT de chaque axe devient le bit de POIDS FAIBLE du champ qui le précède — un bit qui
// bascule d'une image à l'autre. Le pas vaut alors exactement la moitié de l'étendue de l'axe.
// C'est une prédiction chiffrée, pas une impression : elle se vérifie ou elle tombe.
//
// USAGE (lecture seule, aucun film = saut propre) :
//
//	CGO_ENABLED=0 \
//	BITPROJ_PARC=<racine portant data/> \
//	BITPROJ_FILMS='0797ce72=Live Fire,21ece4d8=Live Fire' \
//	  go test ./internal/analysis/filmdec -run TestBancBitProjectile -v -timeout 1800s
//
// Réglages : BITPROJ_DETAIL (nombre de pas détaillés par film, défaut 20).

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

const (
	bitProjParcEnv   = "BITPROJ_PARC"
	bitProjFilmsEnv  = "BITPROJ_FILMS"
	bitProjDetailEnv = "BITPROJ_DETAIL"
	// bitProjPasMaxM est le seuil de pas impossible du lot B (`projectileMaxStepM`).
	bitProjPasMaxM = 10.0
)

// bitProjFilm : un film du banc et la carte qui porte ses bornes.
type bitProjFilm struct{ id, carte string }

// bpEchantillon est un record d'objet du monde AVEC ses bits — c'est ce que le décodeur de
// production jette et dont la preuve a besoin.
type bpEchantillon struct {
	ts        uint64
	chunk     int
	bit       int // position de bit du premier bit d'i0 (après le masque)
	slot, gen uint32
	porte     uint64    // les bits de porte, à la largeur lue
	q         [3]uint64 // quanta bruts
	v         [3]float32
}

func TestBancBitProjectile(t *testing.T) {
	parc := os.Getenv(bitProjParcEnv)
	films := bitProjFilmsDeEnv(os.Getenv(bitProjFilmsEnv))
	if parc == "" || len(films) == 0 {
		t.Skipf("banc désactivé : %s et %s requis", bitProjParcEnv, bitProjFilmsEnv)
	}
	cat, err := LoadMapQuantCatalog(filepath.Join(
		parc, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	for _, f := range films {
		t.Run(f.id, func(t *testing.T) { bitProjMesureFilm(t, parc, cat, f) })
	}
}

func bitProjFilmsDeEnv(s string) []bitProjFilm {
	var out []bitProjFilm
	for _, part := range strings.Split(s, ",") {
		id, carte, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || id == "" || carte == "" {
			continue
		}
		out = append(out, bitProjFilm{id: id, carte: carte})
	}
	return out
}

func bitProjDetail() int {
	if n, err := strconv.Atoi(os.Getenv(bitProjDetailEnv)); err == nil && n > 0 {
		return n
	}
	return 20
}

// bitProjMesureFilm cuit un film sous DEUX portes et publie le verdict chiffré.
func bitProjMesureFilm(t *testing.T, parc string, cat *MapQuantCatalog, f bitProjFilm) {
	t.Helper()
	entry, err := cat.Lookup(f.carte)
	if err != nil {
		t.Fatalf("bornes de %q : %v", f.carte, err)
	}
	film, err := filmsource.LoadDir(filepath.Join(parc, "data", "cache", "film_chunks", f.id), nil)
	if err != nil {
		t.Skipf("film %s absent : %v", f.id, err)
	}
	rng := entry.Range()
	porteAncienne := 3                                    // ancien littéral de decodeWorldObjectPos, corrigé au BB.2 (porte = 2 + IndexW)
	porteCat := 2 + int(entry.EffectiveRegionIndexBits()) // precHigh + index-sel + index de région

	t.Logf("=== %s / %s (module %s) ===", f.id, f.carte, entry.Module)
	t.Logf("bornes X[%.3f;%.3f] Y[%.3f;%.3f] Z[%.3f;%.3f]  étendues %.3f / %.3f / %.3f",
		entry.Min[0], entry.Max[0], entry.Min[1], entry.Max[1], entry.Min[2], entry.Max[2],
		entry.Max[0]-entry.Min[0], entry.Max[1]-entry.Min[1], entry.Max[2]-entry.Min[2])
	t.Logf("largeurs d'axe %d/%d/%d · région attendue %d sur %d bits · porte ancienne %d bits (litteral 3, corrige au BB.2), porte catalogue %d bits",
		entry.AxisWidths[0], entry.AxisWidths[1], entry.AxisWidths[2],
		entry.Region, entry.EffectiveRegionIndexBits(), porteAncienne, porteCat)

	band := worldObjectSlotBand(film, ProjectileTypeIndex)
	if len(band) == 0 {
		t.Skipf("aucun slot ti=%d dans les images-clés de %s", ProjectileTypeIndex, f.id)
	}
	// PORTE ATTENDUE. L'ancienne porte exigeait les 3 bits NULS. À la porte du catalogue, les deux
	// premiers bits (precHigh, index-sel) restent nuls mais l'index de région vaut la région
	// JOUÉE de la carte : sur Live Fire, 1. Exiger zéro y écarterait tous les records.
	prod := bitProjBalaye(film, band, entry.AxisWidths, rng, porteAncienne, 0)
	cata := bitProjBalaye(film, band, entry.AxisWidths, rng, porteCat, uint64(entry.Region))
	t.Logf("records retenus : porte %d bits -> %d · porte %d bits -> %d",
		porteAncienne, len(prod), porteCat, len(cata))

	pProd := bitProjPas(prod)
	pCata := bitProjPas(cata)
	t.Logf("PAS IMPOSSIBLES (> %.0f m) : porte %d bits -> %d/%d (%.2f %%) · porte %d bits -> %d/%d (%.2f %%)",
		bitProjPasMaxM,
		porteAncienne, bitProjCompte(pProd), len(pProd), 100*bitProjPart(pProd),
		porteCat, bitProjCompte(pCata), len(pCata), 100*bitProjPart(pCata))

	bitProjDetaille(t, entry, prod, porteAncienne, porteCat)
}

// bitProjBalaye rejoue `scanProjectileRecords` avec une porte de largeur LIBRE, et garde les
// bits. Aucune I/O : le film est déjà chargé.
func bitProjBalaye(
	film *filmsource.Film, band map[uint32]bool, w [3]uint, rng Vec3Range,
	porte int, attendue uint64,
) []bpEchantillon {
	posBits := porte + int(w[0]+w[1]+w[2]) + 2
	var out []bpEchantillon
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(chunk)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok || rec.Idx[0] != 0 {
					continue
				}
				e, ok := bitProjLis(pay, rec, w, rng, porte, attendue)
				if !ok {
					continue
				}
				e.ts, e.chunk = pk.TimestampUS, c
				out = append(out, e)
				p += posBits
			}
		}
	}
	return out
}

// bitProjLis lit un i0 d'objet du monde à une porte donnée et rend l'échantillon AVEC ses bits.
// Mêmes règles de rejet que la production : porte nulle, quanta saturés écartés.
func bitProjLis(
	pay []byte, rec WorldObjectRecord, w [3]uint, rng Vec3Range, porte int, attendue uint64,
) (bpEchantillon, bool) {
	var e bpEchantillon
	e.porte = PeekBits(pay, rec.After, porte)
	if e.porte != attendue {
		return e, false
	}
	off := rec.After + porte
	for a := 0; a < 3; a++ {
		q := PeekBits(pay, off, int(w[a]))
		if q == 0 || q == (uint64(1)<<w[a])-1 {
			return e, false
		}
		lo, hi := rng[a].Min, rng[a].Max
		e.q[a] = q
		e.v[a] = lo + (float32(q)+0.5)*(hi-lo)/float32(uint64(1)<<w[a])
		off += int(w[a])
	}
	e.slot, e.gen, e.bit = rec.Slot, rec.Gen, rec.After
	return e, true
}

// bpPas est un pas entre deux échantillons consécutifs d'une même vie.
type bpPas struct {
	a, b bpEchantillon
	d    float64
}

// bitProjPas regroupe les échantillons par vie (slot, génération, trou de 250 ms) et rend les
// pas consécutifs. Même découpage que `splitLives` — sans quoi on compterait comme un pas la
// frontière entre deux vols du même slot.
func bitProjPas(ech []bpEchantillon) []bpPas {
	par := map[[2]uint32][]bpEchantillon{}
	for _, e := range ech {
		k := [2]uint32{e.slot, e.gen}
		par[k] = append(par[k], e)
	}
	var out []bpPas
	for _, v := range par {
		sort.Slice(v, func(i, j int) bool {
			if v[i].ts != v[j].ts {
				return v[i].ts < v[j].ts
			}
			return v[i].bit < v[j].bit
		})
		for i := 1; i < len(v); i++ {
			if v[i].ts-v[i-1].ts > projectileGapUS {
				continue // frontière de vie, pas un pas
			}
			dx := float64(v[i].v[0] - v[i-1].v[0])
			dy := float64(v[i].v[1] - v[i-1].v[1])
			out = append(out, bpPas{a: v[i-1], b: v[i], d: math.Hypot(dx, dy)})
		}
	}
	return out
}

func bitProjCompte(ps []bpPas) int {
	n := 0
	for _, p := range ps {
		if p.d > bitProjPasMaxM {
			n++
		}
	}
	return n
}

func bitProjPart(ps []bpPas) float64 {
	if len(ps) == 0 {
		return 0
	}
	return float64(bitProjCompte(ps)) / float64(len(ps))
}

// bitProjDetaille publie, pour les premiers pas impossibles de la lecture de PRODUCTION, les
// quanta bruts des deux points et le bit qui les sépare. C'est la pièce de la preuve.
func bitProjDetaille(t *testing.T, e MapQuantEntry, prod []bpEchantillon, porteAncienne, porteCat int) {
	t.Helper()
	ps := bitProjPas(prod)
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].a.ts != ps[j].a.ts {
			return ps[i].a.ts < ps[j].a.ts
		}
		return ps[i].a.bit < ps[j].a.bit
	})
	max := bitProjDetail()
	t.Logf("--- pas impossibles détaillés (porte %d bits ; la porte catalogue est %d) ---",
		porteAncienne, porteCat)
	t.Logf("%-6s %-5s %-10s | %-22s | %-22s | %s",
		"slot", "gen", "bit i0", "quanta A (x/y/z)", "quanta B (x/y/z)", "XOR par axe")
	n := 0
	for _, p := range ps {
		if p.d <= bitProjPasMaxM || n >= max {
			continue
		}
		n++
		var xor [3]uint64
		for a := 0; a < 3; a++ {
			xor[a] = p.a.q[a] ^ p.b.q[a]
		}
		t.Logf("%-6d %-5d %-10d | %-22s | %-22s | %s  pas %.2f m (dx %.2f dy %.2f)",
			p.a.slot, p.a.gen, p.a.bit,
			fmt.Sprintf("%d/%d/%d", p.a.q[0], p.a.q[1], p.a.q[2]),
			fmt.Sprintf("%d/%d/%d", p.b.q[0], p.b.q[1], p.b.q[2]),
			fmt.Sprintf("%s/%s/%s",
				bitProjRangDuBit(xor[0], e.AxisWidths[0]),
				bitProjRangDuBit(xor[1], e.AxisWidths[1]),
				bitProjRangDuBit(xor[2], e.AxisWidths[2])),
			p.d, float64(p.b.v[0]-p.a.v[0]), float64(p.b.v[1]-p.a.v[1]))
	}
	if n == 0 {
		t.Logf("aucun pas impossible sur ce film à la porte de production")
	}
}

// bitProjRangDuBit nomme le bit de poids le plus fort qui a basculé, compté depuis le MSB du
// champ (« msb » = le bit de poids fort, « - » = aucune bascule).
func bitProjRangDuBit(xor uint64, w uint) string {
	if xor == 0 {
		return "-"
	}
	for k := int(w) - 1; k >= 0; k-- {
		if xor&(uint64(1)<<uint(k)) != 0 {
			if k == int(w)-1 {
				return "msb"
			}
			return fmt.Sprintf("msb-%d", int(w)-1-k)
		}
	}
	return "-"
}
