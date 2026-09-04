package filmdec

// r12_corpus_research_test.go — LES TETES SEULES, SUR UN CORPUS : le cadrage CERTAIN.
//
// POURQUOI CET INSTRUMENT EXISTE. La marche de liste peut deriver, et R7 a montre que la
// derive FABRIQUE des occurrences des types rares (104, 105, 119, 14) : sur `084a804d`, film
// dont il mesure la marche la plus mauvaise, on en compte 11, 27, 19 et 21. Un compte tire de
// la marche ne peut donc pas, a lui seul, etablir qu'un type existe.
//
// LA TETE, ELLE, NE PEUT PAS DERIVER. Le premier evenement d'une liste se lit
// `[1 bit config][1 bit continuation][R(7) type]` : rien ne le precede, aucune largeur de
// charge n'intervient. C'est l'epreuve D de R7, et c'est elle qui a tranche : les types 104,
// 42, 43 n'y apparaissent JAMAIS sur 12 films (0 pour 42 attendues), alors que le type 105 y
// apparait 8 fois. **Le type 105 est donc le seul de la famille de poussee dont R7 lui-meme
// etablit qu'il EXISTE.**
//
// CE QUE CET INSTRUMENT MESURE, ET QUE PERSONNE N'AVAIT MESURE : les tetes de type 105
// tombent-elles la ou un REPULSEUR est porte ? Il croise, film par film :
//   - le nombre de vies de repulseur (rang i48 nomme par la palette) ;
//   - le nombre de tetes de chaque type de la famille ;
//   - et, pour chaque tete, l'ECART A LA LECTURE i48 « repulseur » LA PLUS PROCHE, contre le
//     meme ecart aux lectures « grappin » (le temoin de specificite) et contre un temoin
//     decale de +30 s.
//
// GARDES : `R12_FILMS`, `R12_IDS`. Aucune ecriture, aucune DuckDB, `CGO_ENABLED=0`. USAGE :
//
//	CGO_ENABLED=0 R12_FILMS=<repo>/data/cache/film_chunks R12_IDS=a,b,c \
//	  go test ./internal/analysis/filmdec/ -run '^TestR12Corpus$' -count=1 -timeout 180m -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

func filepathBase(p string) string { return filepath.Base(p) }

// r12TeteOcc : une tete de liste, datee.
type r12TeteOcc struct {
	Typ int
	MS  int64
}

// r12Ecarts agrege des ecarts signes a une reference.
type r12Ecarts struct {
	n      int
	proche int // |ecart| <= 3000 ms
	vals   []int64
}

func (e *r12Ecarts) ajoute(d int64) {
	e.n++
	if d < 0 {
		if -d <= 3000 {
			e.proche++
		}
	} else if d <= 3000 {
		e.proche++
	}
	e.vals = append(e.vals, d)
}

func (e *r12Ecarts) med() int64 {
	if len(e.vals) == 0 {
		return 0
	}
	v := append([]int64(nil), e.vals...)
	sort.Slice(v, func(a, b int) bool { return v[a] < v[b] })
	return v[len(v)/2]
}

func TestR12Corpus(t *testing.T) {
	t.Logf("%-10s %-11s %7s %7s %8s %5s %5s %5s %5s %5s %5s %5s %5s %5s",
		"film", "palette", "viesRep", "viesGra", "paquets",
		"h104", "h105", "h117", "h103", "h9", "h100", "h116", "csRep", "csGra")
	var totRep, totGra r12Ecarts
	var totRepDec r12Ecarts
	var nTetes105, nTetes104 int
	var detail []string
	for _, dir := range r12FilmDirs(t) {
		// UN SOUS-TEST PAR FILM : sur un corpus de cent films, un film illisible ne doit pas
		// emporter la campagne entiere. `r12Prepare` fait `t.Fatalf` ; isole ici, il n'arrete
		// que ce film-la.
		var r, g, rd r12Ecarts
		var n104, n105 int
		var lignes []string
		t.Run(filepathBase(dir), func(st *testing.T) {
			r, g, rd, n104, n105, lignes = r12CorpusOneFilm(st, dir)
		})
		totRep.n += r.n
		totRep.proche += r.proche
		totRep.vals = append(totRep.vals, r.vals...)
		totGra.n += g.n
		totGra.proche += g.proche
		totGra.vals = append(totGra.vals, g.vals...)
		totRepDec.n += rd.n
		totRepDec.proche += rd.proche
		totRepDec.vals = append(totRepDec.vals, rd.vals...)
		nTetes104 += n104
		nTetes105 += n105
		detail = append(detail, lignes...)
	}
	t.Logf("")
	t.Logf("=== CORPUS — %d tetes de type 104, %d tetes de type 105 ===", nTetes104, nTetes105)
	t.Logf("  ecart a la lecture i48 REPULSEUR la plus proche : n=%d, |d|<=3 s : %d (%.1f %%), "+
		"mediane %+d ms", totRep.n, totRep.proche,
		100*float64(totRep.proche)/float64(max(1, totRep.n)), totRep.med())
	t.Logf("  TEMOIN de specificite — meme ecart aux lectures GRAPPIN : n=%d, |d|<=3 s : %d "+
		"(%.1f %%), mediane %+d ms", totGra.n, totGra.proche,
		100*float64(totGra.proche)/float64(max(1, totGra.n)), totGra.med())
	t.Logf("  TEMOIN decale +30 s (repulseur) : n=%d, |d|<=3 s : %d (%.1f %%), mediane %+d ms",
		totRepDec.n, totRepDec.proche,
		100*float64(totRepDec.proche)/float64(max(1, totRepDec.n)), totRepDec.med())
	t.Logf("  DETAIL des tetes (film, type, instant, ecart au repulseur, ecart au grappin) :")
	for _, l := range detail {
		t.Logf("    %s", l)
	}
}

func r12CorpusOneFilm(t *testing.T, dir string) (rep, gra, repDec r12Ecarts,
	n104, n105 int, lignes []string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r12Prepare(t, dir)
	rd := r12Collect(s)
	pal := r12ClassifyPalette(rd.Ranks)
	rRep, rGra := pal.r12RankOf("Repulsor"), pal.r12RankOf("Grappleshot")
	var tRep, tGra []int64
	vies := map[int]int{}
	for _, r := range rd.Ranks {
		vies[r.Rank]++
		switch r.Rank {
		case rRep:
			tRep = append(tRep, r.MS)
		case rGra:
			tGra = append(tGra, r.MS)
		}
	}
	// LES USAGES CERTAINS : une PORTE OUVERTE d'i48 (`AbilitySetNoRank`) precedee, POUR LE
	// MEME SLOT, d'un rang de repulseur, est la DERNIERE CHARGE CONSOMMEE — le film declare
	// lui-meme que le joueur ne porte plus rien. C'est l'usage certain que R11 par. 5 a
	// identifie et qu'aucun releve Theater n'est necessaire pour dater.
	spentRep := r12Consommations(rd.Ranks, rRep)
	spentGra := r12Consommations(rd.Ranks, rGra)

	tetes, nDelta := r12TetesDuFilm(s)
	h := map[int]int{}
	for _, o := range tetes {
		h[o.Typ]++
	}
	vRep, vGra := -1, -1
	if rRep >= 0 {
		vRep = vies[rRep]
	}
	if rGra >= 0 {
		vGra = vies[rGra]
	}
	t.Logf("%-10s %-11s %7d %7d %8d %5d %5d %5d %5d %5d %5d %5d %5d %5d", s.id, r12PalID(pal),
		vRep, vGra, nDelta, h[104], h[105], h[117], h[103], h[9], h[100], h[116],
		len(spentRep), len(spentGra))
	_ = tGra

	for _, o := range tetes {
		if o.Typ != 104 && o.Typ != 105 {
			continue
		}
		if o.Typ == 104 {
			n104++
		} else {
			n105++
		}
		// LA MESURE : l'ecart a une CONSOMMATION CERTAINE de repulseur, contre le meme
		// ecart a une consommation de grappin (temoin de specificite) et contre le meme
		// ecart decale de +30 s (temoin de hasard).
		dR, okR := r12PlusProche(spentRep, o.MS)
		dG, okG := r12PlusProche(spentGra, o.MS)
		dD, okD := r12PlusProche(spentRep, o.MS+30000)
		if okR {
			rep.ajoute(dR)
		}
		if okG {
			gra.ajoute(dG)
		}
		if okD {
			repDec.ajoute(dD)
		}
		dPrise, okP := r12PlusProche(tRep, o.MS)
		lignes = append(lignes, fmt.Sprintf(
			"%-10s type=%-4d %-9s consoRepulseur=%s consoGrappin=%s priseRepulseur=%s",
			s.id, o.Typ, r12MMSS(o.MS), r12EcartTxt(dR, okR), r12EcartTxt(dG, okG),
			r12EcartTxt(dPrise, okP)))
	}
	return rep, gra, repDec, n104, n105, lignes
}

// r12Consommations rend les instants ou un slot a CONSOMME sa derniere charge de l'equipement
// de rang `rank` : une lecture i48 `AbilitySetNoRank` (porte ouverte) dont la lecture
// PRECEDENTE, pour LE MEME SLOT, portait ce rang. Le film declare lui-meme, a cet instant,
// que le joueur ne porte plus rien : c'est un USAGE certain, sans releve Theater.
func r12Consommations(rs []r12RankRead, rank int) []int64 {
	if rank < 0 {
		return nil
	}
	tl := r12BuildTimeline(rs)
	var out []int64
	for _, v := range tl {
		for i := 1; i < len(v); i++ {
			if v[i].Rank == AbilitySetNoRank && v[i-1].Rank == rank {
				out = append(out, v[i].MS)
			}
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

func r12EcartTxt(d int64, ok bool) string {
	if !ok {
		return "     (aucune)"
	}
	return fmt.Sprintf("%+9d ms", d)
}

// r12PlusProche rend l'ecart signe (instant - lecture) le plus petit en valeur absolue.
func r12PlusProche(xs []int64, ms int64) (int64, bool) {
	if len(xs) == 0 {
		return 0, false
	}
	best, bd := int64(0), int64(1<<62)
	for _, x := range xs {
		d := ms - x
		a := d
		if a < 0 {
			a = -a
		}
		if a < bd {
			best, bd = d, a
		}
	}
	return best, true
}

// r12TetesDuFilm lit la tete de chaque paquet delta. Aucune grammaire, aucun deser.
func r12TetesDuFilm(s r12Setup) ([]r12TeteOcc, int) {
	var out []r12TeteOcc
	n := 0
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			n++
			if tp := r12Tete(pk.Payload(data)); tp >= 0 {
				out = append(out, r12TeteOcc{tp, s.ms(pk.TimestampUS)})
			}
		}
	}
	return out, n
}
