package filmdec

// r9_masque_research_test.go — LE RECENSEMENT DU MASQUE BIPEDE PAR RANG PORTE
// (par. 7.5 du RAPPORT_R9_REPULSEUR_2026-09-03). Il traite la porte (c) du registre R8 (i56
// `biped-spartan-ability-energy`) dans sa forme la plus forte : au lieu d'instruire UN
// composant, il les interroge TOUS LES 64 d'un coup.
//
// POURQUOI CETTE FORME-LA. Le balayage du masque de R8 (par. 7.3) etait ancre sur des BOUFFEES
// DE VITESSE — et la mesure du par. 6 de ce rapport etablit que le repulseur n'en produit pas,
// ni chez son porteur ni chez sa victime. Cet ancrage etait donc aveugle a la cible. Le
// recensement, lui, N'A BESOIN D'AUCUNE ANCRE : si l'usage du repulseur est transmis par un
// composant quelconque du bipede, ce composant doit etre ANNONCE plus souvent quand un
// repulseur est porte — au moins a l'instant de l'usage.
//
// CE QUE LA MESURE NE PEUT PAS VOIR, ET C'EST DIT : un usage transmis comme une VALEUR a
// l'interieur d'un composant annonce en permanence (le cas du propulseur, tag 1 d'i57)
// n'apparait pas dans un recensement d'annonces. Le recensement ferme la question « un
// composant REAGIT-IL au port d'un repulseur », pas la question « une valeur le fait ».
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	candidat  un composant dont le taux d'annonce par record pour les porteurs de REPULSEUR
//	          vaut >= 3x le maximum des rangs temoins bien peuples (>= 10 vies).
//	temoin    i28 `unit-active-camo-state-component` DOIT ressortir sur le rang du camouflage
//	          quand ce rang est present : c'est le seul composant du bipede dont on sait
//	          d'avance qu'il depend de l'equipement porte. S'il ne ressort pas, le recensement
//	          n'a pas de puissance et aucun negatif ne sera publie.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=06dfe6d9 go test ./internal/analysis/filmdec/ \
//	  -run '^TestR9Masque$' -count=1 -timeout 120m -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// r9MasqueMin est le nombre minimal de records pour qu'un rang soit juge. Sous ce seuil, un
// taux d'annonce n'est pas une mesure.
const r9MasqueMin = 200

func TestR9Masque(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r9MasqueOneFilm(t, dir)
	}
}

func r9MasqueOneFilm(t *testing.T, dir string) {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	s := r8MobResolve(t, dir)
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	lives := r8Lives(r8BuildSpeeds(pos))
	ranks, _, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("rangs de capacite illisibles : %v", err)
	}
	recs, ann := r9MasqueScan(s, ranks, lives)
	r9LogMasque(t, filepath.Base(dir), s.arch, recs, ann)
}

// r9MasqueScan compte, par rang de capacite porte, le nombre de records et le nombre
// d'annonces de chaque composant. AUCUNE marche : seuls l'en-tete et le masque sont lus, donc
// le resultat ne depend d'aucun deser et ne souffre d'aucune desynchronisation.
func r9MasqueScan(
	s r8MobSetup, ranks []AbilityRank, lives map[uint32][]r8LifeSpan,
) (map[int]int, map[int]map[int]int) {
	recs := map[int]int{}
	ann := map[int]map[int]int{}
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, ids, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				rk := r8RankInLife(ranks, lives, slot, pk.TimestampUS)
				recs[rk]++
				for _, id := range ids {
					if ann[id] == nil {
						ann[id] = map[int]int{}
					}
					ann[id][rk]++
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
	return recs, ann
}

// r9LogMasque publie, pour chaque composant, le taux d'annonce par record et par rang, et
// designe les candidats selon le critere pre-inscrit.
func r9LogMasque(
	t *testing.T, film string, arch Archetype, recs map[int]int, ann map[int]map[int]int,
) {
	t.Helper()
	rks := make([]int, 0, len(recs))
	for k := range recs {
		if k >= 0 && recs[k] >= r9MasqueMin {
			rks = append(rks, k)
		}
	}
	sort.Ints(rks)
	t.Logf("%s : recensement du masque BIPEDE par rang porte (rangs a >= %d records)",
		film, r9MasqueMin)
	var head string
	for _, k := range rks {
		head += padRank(k, recs[k])
	}
	t.Logf("  %-52s %s", "composant", head)
	ids := make([]int, 0, len(ann))
	for id := range ann {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		var line string
		for _, k := range rks {
			line += padRate(float64(ann[id][k]) / float64(recs[k]))
		}
		t.Logf("  i%-2d %-48s %s", id, arch.component(id), line)
	}
}

func padRank(rang, n int) string {
	return fmt.Sprintf("%12s", fmt.Sprintf("r%d(%d)", rang, n))
}

func padRate(v float64) string {
	return fmt.Sprintf("%12.3f", v)
}
