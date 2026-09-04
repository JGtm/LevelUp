package filmdec

// r12_knockback_research_test.go — LES DEUX TYPES DE POUSSEE, MIS A L'EPREUVE.
//
// CE QUE LA MESURE ANCREE A TROUVE (par. 4 du rapport R12) : sur `215e7022`, le type 104
// `EquipmentKnockbackPlayer` n'apparait QUE DEUX FOIS dans tout le film, a 5:25,5 et 5:25,6 —
// c'est-a-dire a l'instant EXACT du seul usage de repulseur que le releve Theater date avec
// un kill (325 526 ms, mesure `TestR12AncreKill`). Et le type 105 `EquipmentObjectKnockedBack`
// y apparait 10 fois, dont 4 dans les fenetres d'usage.
//
// R7 A DECLARE CES TYPES ABSENTS, et il faut prendre son argument au serieux : sur 12 films
// il a mesure 108 occurrences de 104, ZERO en position de tete (42 attendues, p ~ 1e-23), une
// reference constante a 4224, et 75 % d'entre elles PRECEDEES par le type 0
// `damage_aftermath`, dont la grammaire de production est mesuree DOUTEUSE. Sa conclusion —
// « les occurrences de 104 sont la derive produite par cette largeur » — etait la seule
// disponible SANS ANCRE.
//
// CE FICHIER NE REFUTE PAS R7 PAR UNE OPINION : il pose les controles que R7 ne pouvait pas
// poser, parce qu'il n'avait aucun instant d'usage certain.
//
//	CONTROLE 1  POSITION, PREDECESSEUR, REFERENCE de chaque occurrence — les trois epreuves
//	            de R7, rejouees occurrence par occurrence au lieu d'un agregat de parc.
//	CONTROLE 2  ORACLE DE TRAME RESTREINT : la trame qui SUIT une liste contenant un 104
//	            va-t-elle aussi loin que la trame moyenne ? Une liste mal cadree derive et la
//	            trame s'effondre (R7 : facteur 5,46 mesure sur ce film entre cadrage juste et
//	            temoin decale).
//	CONTROLE 3  CORPUS : le compte de 104 par film suit-il le nombre de VIES DE REPULSEUR, ou
//	            le nombre de MORTS ? Si 104 est la derive de `damage_aftermath`, il suit les
//	            morts. S'il est le repulseur, il suit les vies de repulseur — et il vaut ZERO
//	            sur un film ou personne n'en porte.
//
// GARDES : `R12_FILMS`, `R12_IDS`, `R12_CORPUS=1` (mode corpus, resume par film seulement).
// Aucune ecriture, aucune DuckDB, `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R12_FILMS=<repo>/data/cache/film_chunks R12_IDS=215e7022 \
//	  go test ./internal/analysis/filmdec/ -run '^TestR12Knockback$' -count=1 -timeout 60m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// r12KnockTypes : les types de poussee suivis, et leurs voisins de famille.
var r12KnockTypes = []int{104, 105, 119, 116, 117, 103, 14, 93, 98, 30}

// r12Occ : UNE occurrence d'un type suivi, avec tout ce qui permet de la juger.
type r12Occ struct {
	MS      int64
	Typ     int
	Pos     int
	Long    int
	Pred    int
	Ref0    uint64
	HasRef0 bool
	Fin     bool
}

func TestR12Knockback(t *testing.T) {
	corpus := os.Getenv("R12_CORPUS") == "1"
	t.Logf("%-10s %-11s %6s %6s %6s %7s %6s %6s %6s %6s %6s", "film", "palette",
		"viesRep", "viesGra", "listes", "t0", "t104", "t105", "t119", "t117", "t14")
	for _, dir := range r12FilmDirs(t) {
		r12KnockOneFilm(t, dir, corpus)
	}
}

func r12KnockOneFilm(t *testing.T, dir string, corpus bool) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r12Prepare(t, dir)
	ctx := r12CtxDeLayout(s.lay)
	r12VerifieCtx(t, s.lay, ctx)
	rd := r12Collect(s)
	pal := r12ClassifyPalette(rd.Ranks)
	rep, gra := pal.r12RankOf("Repulsor"), pal.r12RankOf("Grappleshot")
	vies := map[int]int{}
	for _, r := range rd.Ranks {
		vies[r.Rank]++
	}

	occs, listes, nT0 := r12Occurrences(s, ctx)
	cnt := map[int]int{}
	for _, o := range occs {
		cnt[o.Typ]++
	}
	// `viesRep` vaut -1 quand la palette ne nomme AUCUN rang « Repulsor » : c'est INCONNU,
	// pas zero. La famille B ne nomme que le grappin et le propulseur.
	vRep, vGra := -1, -1
	if rep >= 0 {
		vRep = vies[rep]
	}
	if gra >= 0 {
		vGra = vies[gra]
	}
	t.Logf("%-10s %-11s %6d %6d %6d %7d %6d %6d %6d %6d %6d", s.id, r12PalID(pal),
		vRep, vGra, listes, nT0, cnt[104], cnt[105], cnt[119], cnt[117], cnt[14])
	if corpus {
		return
	}

	t.Logf("  palette=%s rangRepulseur=%d rangGrappin=%d", r12PalID(pal), rep, gra)
	t.Logf("  CONTROLE 1 — chaque occurrence, avec sa position, son predecesseur et sa reference :")
	t.Logf("    %-9s %-5s %-38s %4s %5s %5s %-30s %10s",
		"instant", "type", "nom", "pos", "long", "pred", "nomPredecesseur", "ref0")
	for _, o := range occs {
		ref := "(aucune)"
		if o.HasRef0 {
			ref = fmt.Sprintf("%d", o.Ref0)
		}
		t.Logf("    %-9s %-5d %-38s %4d %5d %5d %-30s %10s",
			r12MMSS(o.MS), o.Typ, r7Noms[o.Typ], o.Pos, o.Long, o.Pred, r7Noms[o.Pred], ref)
	}

	// CONTROLE 2 — l'oracle de trame RESTREINT aux listes qui portent le type.
	reg, chunks, err := r7Chargements(dir)
	if err != nil || len(chunks) == 0 {
		t.Logf("  CONTROLE 2 : registre illisible (%v) — oracle restreint non mesure", err)
		return
	}
	cfg := DefaultFrameConfig()
	cfg.IDLowBits, _ = r7CalibreIDLow(reg, chunks)
	toutes, _ := r7OracleFilm(reg, chunks, ctx, cfg, nil, 0)
	r7RapportTrame(t, "  CONTROLE 2 — toutes listes ", toutes)
	for _, tp := range []int{104, 105, 117} {
		if cnt[tp] == 0 {
			continue
		}
		garde := func(evs []r7Ev) bool {
			for _, e := range evs {
				if e.Typ == tp {
					return true
				}
			}
			return false
		}
		st, _ := r7OracleFilm(reg, chunks, ctx, cfg, garde, 0)
		r7RapportTrame(t, fmt.Sprintf("  CONTROLE 2 — listes avec %d ", tp), st)
	}
}

// r12Occurrences marche toutes les listes non vides et releve les occurrences des types
// suivis, avec leur position, leur predecesseur et leur reference. Rend aussi le nombre de
// listes non vides (le denominateur) et le nombre d'evenements de type 0 `damage_aftermath` —
// le predecesseur suspect de R7, dont le compte est LE regresseur concurrent du controle 3.
func r12Occurrences(s r12Setup, ctx r7Ctx) ([]r12Occ, int, int) {
	var out []r12Occ
	listes, nT0 := 0, 0
	suivi := map[int]bool{}
	for _, tp := range r12KnockTypes {
		suivi[tp] = true
	}
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
			if r12Tete(pay) < 0 {
				continue
			}
			listes++
			ms := s.ms(pk.TimestampUS)
			evs, stop, opq, _ := r7MarcheDecalee(pay, ctx, 0)
			for i, e := range evs {
				if e.Typ == 0 {
					nT0++
				}
				if !suivi[e.Typ] {
					continue
				}
				pred := -1
				if i > 0 {
					pred = evs[i-1].Typ
				}
				out = append(out, r12Occ{MS: ms, Typ: e.Typ, Pos: e.Pos, Long: len(evs),
					Pred: pred, Ref0: e.Ref0, HasRef0: e.HasRef0, Fin: stop == r7StopFin})
			}
			// Le type qui BLOQUE la marche est releve aussi : c'est le seul moyen de compter
			// un type opaque (le 14 `PlayEffectOnObject`, entre autres).
			if (stop == r7StopOpaque || stop == r7StopSansDomaine) && suivi[opq] {
				pred := -1
				if len(evs) > 0 {
					pred = evs[len(evs)-1].Typ
				}
				out = append(out, r12Occ{MS: ms, Typ: opq, Pos: len(evs) + 1,
					Long: len(evs) + 1, Pred: pred, Fin: false})
			}
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Typ != out[b].Typ {
			return out[a].Typ < out[b].Typ
		}
		return out[a].MS < out[b].MS
	})
	return out, listes, nT0
}
