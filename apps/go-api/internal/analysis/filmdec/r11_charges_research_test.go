package filmdec

// r11_charges_research_test.go — LES CHARGES CONSOMMEES, PAR EQUIPEMENT NOMME.
//
// CE QUE CET INSTRUMENT MESURE. Le journal (`r11_journal_research_test.go`) a montre sur
// `1cd3848a` une serie i56 4 -> 3 -> 2 -> 1 -> 0 dont les cinq instants tombent exactement sur
// les cinq usages de propulseur releves par l'utilisateur au Theater. Cet instrument
// generalise : il compte les BAISSES DE CHARGE par equipement NOMME PAR LA PALETTE, avec les
// denominateurs (vies, segments de port, minutes d'exposition) sans lesquels un compte ne se
// juge pas.
//
// LA REGLE D'UNE BAISSE, ECRITE AVANT LA MESURE. Pour un slot et un emplacement k du masque
// R(3) : la valeur pleine PAR DEFAUT est 0x7F (`ability_energy.go` : le moteur pose 0x7F quand
// le bit de masque vaut 0, et le film ne transmet alors RIEN). On tient donc, par (slot, k),
// un quartet courant initialise a 7 ; une lecture ARMEE dont le quartet est STRICTEMENT
// INFERIEUR au quartet courant est UNE baisse — un evenement, pas une difference : une chute
// de 7 a 4 compte pour UN usage, parce que la valeur pleine n'est jamais transmise et que le
// premier usage revele donc « ce qui reste ». Un emplacement non arme remet le quartet
// courant a 7 (l'equipement a disparu de la main).
//
// L'ATTRIBUTION. Un usage est attribue au rang lu par i48 le plus recemment AVANT lui, DANS LA
// MEME VIE (les pistes de l'artefact : le slot MIGRE aux reapparitions). La porte ouverte
// (`AbilitySetNoRank`) est une valeur et non un trou : elle sert de CONTROLE NEGATIF — un
// joueur qui ne porte rien ne doit pas consommer de charge.
//
// LES TEMOINS POSITIFS, ELIMINATOIRES. Le PROPULSEUR (verite terrain de l'utilisateur,
// `1cd3848a`) et le GRAPPIN doivent ressortir. Un instrument qui ne les retrouve pas ne
// prononce aucun negatif sur le repulseur.
//
// GARDES : `R9_FILMS`, `R9_ARTIFACTS`, `R8_BOUNDS`, `R11_IDS` (obligatoire), `R11_DETAIL`
// (nom EN d'un equipement dont on veut le detail des instants, defaut `Repulsor`).
// Aucune ecriture, aucune DuckDB, `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R9_FILMS=<repo>/data/cache/film_chunks \
//	  R9_ARTIFACTS=<repo>/data/cache/replays/halo_infinite \
//	  R8_BOUNDS=<wt>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R11_IDS=1cd3848a,72b0a25e go test ./internal/analysis/filmdec/ \
//	  -run '^TestR11Charges$' -count=1 -timeout 120m -v

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

const (
	r11DetailEnv = "R11_DETAIL"
	// r11FullNibble : le quartet de la valeur pleine 0x7F, jamais transmis par le film.
	r11FullNibble = 7
	// r11MatchUS : fenetre d'appariement entre une baisse de charge et une impulsion i57/i59.
	// 1,5 s — le geste dure une demi-seconde, les deux canaux sont dans le meme paquet ou dans
	// le suivant.
	r11MatchUS = 1_500_000
)

// r11Use est UNE baisse de charge : un usage date, attribue a un equipement nomme.
type r11Use struct {
	Slot   uint32
	TSUS   uint64
	KSlot  int // emplacement du masque R(3) ou la baisse a eu lieu
	From   int // quartet precedent (7 = plein par defaut, jamais transmis)
	To     int // quartet lu
	Rank   int
	Name   string // gamertag du porteur, par la piste de l'artefact
	Imp    bool   // une impulsion i57/i59 tag==1 a moins de r11MatchUS
	InLife bool   // le rang vient bien de la meme vie
	// AgeUS est l'age de la lecture i48 qui nomme cet usage. i48 n'emet qu'environ une fois
	// par vie : un rang lu 120 s plus tot peut avoir ete remplace sans que le film le dise.
	// C'est la QUALITE de l'attribution, et elle doit voyager avec elle.
	AgeUS uint64
}

// r11FreshUS : au-dela, le rang qui nomme un usage est trop vieux pour etre digne de foi.
// 20 s — l'ordre de grandeur de la periode d'emission d'i48 (une fois par vie environ, la
// fiche de creneaux du 2026-09-03 mesure « une fois toutes les 20 secondes »).
const r11FreshUS = 20_000_000

// r11Seg est UN segment de port : « ce slot porte ce rang, de tel a tel instant ».
type r11Seg struct {
	Slot     uint32
	Rank     int
	From, To uint64
}

// r11Row agrege ce qu'un equipement a produit sur un film.
type r11Row struct {
	Rank          int
	Label         string
	Lives, Segs   int
	ExposUS       uint64
	Uses, WithImp int
	// UsesFresh : ceux des usages dont la lecture i48 qui les nomme a moins de r11FreshUS.
	// C'est la colonne a lire : au-dela, l'attribution est une supposition (mesure du
	// 084a804d : six baisses nommees « repulseur » par une lecture vieille de deux minutes
	// tombent en fait sur les cinq accroches de grappin de l'artefact).
	UsesFresh       int
	Imps            int
	FirstNibbleHist map[int]int
	// Reads / Armed : les lectures d'i56 tombant dans un segment de port de cet equipement,
	// et celles dont au moins un emplacement est ARME. Ce sont elles qui distinguent
	// « le canal ne dit rien de cet equipement » de « le canal en parle et ne baisse jamais ».
	Reads, Armed int
	// Consumed : la somme des DECROISSANCES de quartet, hors premiere lecture (partant du
	// plein 0x7F, jamais transmis, l'ecart ne mesure rien).
	Consumed int
	// KHist : les emplacements du masque R(3) ou les valeurs armees ont ete vues.
	KHist map[int]int
}

func TestR11Charges(t *testing.T) {
	detail := strings.TrimSpace(os.Getenv(r11DetailEnv))
	if detail == "" {
		detail = "Repulsor"
	}
	for _, dir := range r11FilmDirs(t) {
		r11ChargesOneFilm(t, dir, detail)
	}
}

func r11ChargesOneFilm(t *testing.T, dir, detail string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r11Prepare(t, dir)
	rd := r11Collect(s.scan)
	r11LogHeader(t, s, rd.Stat)

	segs := r11Segments(s, rd.Ranks)
	uses := r11Uses(s, rd, segs)
	rows := r11Rows(s, segs, uses, rd)
	t.Logf("  segments de port %d sur %d lectures i48 (les autres tombent hors piste connue : "+
		"nommer le joueur y serait deviner)", len(segs), len(rd.Ranks))
	r11LogRows(t, rows)
	r11LogArmed(t, s, rd)
	r11LogDetail(t, s, uses, detail)
}

// r11LogArmed publie la MEME question sous une attribution plus large, et c'est une garde
// contre l'illusion de couverture : les segments de port commencent a la premiere lecture
// d'i48 de la vie, si bien qu'une lecture d'i56 faite avant elle n'appartient a aucun segment.
// Ici chaque lecture ARMEE est rattachee au rang lu par i48 le plus proche DANS LA MEME VIE,
// avant OU apres. Lecture plus faible (un echange a pu avoir lieu entre les deux), publiee a
// part pour cette raison — mais elle ne peut pas manquer un equipement qui armerait i56.
func r11LogArmed(t *testing.T, s r11Setup, rd r11Reads) {
	t.Helper()
	bySlot := map[uint32][]r11RankRead{}
	for _, r := range rd.Ranks {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	tot, armed := 0, 0
	byRank := map[int]int{}
	kHist := map[int]int{}
	orphan := 0
	for _, e := range rd.Energy {
		tot++
		if e.Mask == 0 {
			continue
		}
		armed++
		for k := 0; k < AbilityEnergyCharges; k++ {
			if e.Ch[k] != AbilityEnergyUnarmed {
				kHist[k]++
			}
		}
		rank, ok := r11NearestRank(s, bySlot[e.Slot], e)
		if !ok {
			orphan++
			continue
		}
		byRank[rank]++
	}
	t.Logf("  ATTRIBUTION LARGE (rang i48 le plus proche dans la vie) — i56 lu %d, arme %d "+
		"(e0=%d e1=%d e2=%d), sans aucune lecture i48 dans la vie : %d",
		tot, armed, kHist[0], kHist[1], kHist[2], orphan)
	var ranks []int
	for r := range byRank {
		ranks = append(ranks, r)
	}
	sort.Slice(ranks, func(a, b int) bool { return byRank[ranks[a]] > byRank[ranks[b]] })
	for _, r := range ranks {
		t.Logf("    %-28s %d lectures armees", r11RankLabel(s.art, r), byRank[r])
	}
}

// r11NearestRank rend le rang i48 le plus proche de cette lecture DANS LA MEME VIE.
func r11NearestRank(s r11Setup, rs []r11RankRead, e r11EnergyRead) (int, bool) {
	tr, ok := r11TrackAt(s, e.Slot, e.TSUS)
	if !ok {
		return 0, false
	}
	best, bestD, found := AbilitySetNoRank, uint64(1)<<62, false
	for _, r := range rs {
		f := r11Frame(s, r.TSUS)
		if f < tr.Points[0].T || f > tr.EndFrame {
			continue
		}
		d := uint64(0)
		if r.TSUS > e.TSUS {
			d = r.TSUS - e.TSUS
		} else {
			d = e.TSUS - r.TSUS
		}
		if d < bestD {
			best, bestD, found = r.Rank, d, true
		}
	}
	return best, found
}

// r11Segments construit les segments de port a partir des lectures d'i48. Un segment court
// de la lecture jusqu'a la lecture SUIVANTE du meme slot dans la MEME VIE, ou jusqu'a la fin
// de la vie. Hors vie connue (aucune piste ne couvre l'instant), la lecture est ignoree :
// attribuer un rang a un slot sans piste nommerait le joueur precedent.
func r11Segments(s r11Setup, ranks []r11RankRead) []r11Seg {
	bySlot := map[uint32][]r11RankRead{}
	for _, r := range ranks {
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	var out []r11Seg
	for slot, rs := range bySlot {
		sort.Slice(rs, func(a, b int) bool { return rs[a].TSUS < rs[b].TSUS })
		for i, r := range rs {
			tr, ok := r11TrackAt(s, slot, r.TSUS)
			if !ok {
				continue
			}
			end := r11FrameUS(s, tr.EndFrame)
			if i+1 < len(rs) && rs[i+1].TSUS < end {
				end = rs[i+1].TSUS
			}
			if end > r.TSUS {
				out = append(out, r11Seg{Slot: slot, Rank: r.Rank, From: r.TSUS, To: end})
			}
		}
	}
	return out
}

// r11TrackAt rend la piste de l'artefact qui couvre cet instant pour ce slot.
func r11TrackAt(s r11Setup, slot uint32, ts uint64) (*r9TrackLite, bool) {
	frame := r11Frame(s, ts)
	for _, tr := range s.art.BySlot[slot] {
		if frame >= tr.Points[0].T && frame <= tr.EndFrame {
			return tr, true
		}
	}
	return nil, false
}

// r11Frame convertit un horodatage moteur en numero de frame de l'artefact.
func r11Frame(s r11Setup, ts uint64) int {
	if s.art.FrameIntervalMs <= 0 {
		return 0
	}
	return int((s.ms(ts) - s.art.OriginMs) / int64(s.art.FrameIntervalMs))
}

// r11FrameUS convertit un numero de frame en horodatage moteur.
func r11FrameUS(s r11Setup, frame int) uint64 {
	ms := s.art.OriginMs + int64(frame)*int64(s.art.FrameIntervalMs)
	return uint64(int64(s.origin) + ms*1000)
}

// r11Uses applique la regle de baisse, emplacement par emplacement, et attribue chaque usage.
func r11Uses(s r11Setup, rd r11Reads, segs []r11Seg) []r11Use {
	bySlot := map[uint32][]r11EnergyRead{}
	for _, e := range rd.Energy {
		bySlot[e.Slot] = append(bySlot[e.Slot], e)
	}
	var out []r11Use
	for slot, es := range bySlot {
		sort.Slice(es, func(a, b int) bool { return es[a].TSUS < es[b].TSUS })
		cur := [AbilityEnergyCharges]int{r11FullNibble, r11FullNibble, r11FullNibble}
		for _, e := range es {
			for k := 0; k < AbilityEnergyCharges; k++ {
				if e.Ch[k] == AbilityEnergyUnarmed {
					cur[k] = r11FullNibble
					continue
				}
				nib := (e.Ch[k] >> 4) & 0xF
				if nib < cur[k] {
					out = append(out, r11Use{Slot: slot, TSUS: e.TSUS, KSlot: k,
						From: cur[k], To: nib})
				}
				cur[k] = nib
			}
		}
	}
	for i := range out {
		r11Attribute(s, &out[i], segs, rd.Imp)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].TSUS < out[b].TSUS })
	return out
}

// r11Attribute nomme l'usage : le rang du segment de port qui le couvre, le gamertag de la
// piste, et l'existence d'une impulsion i57/i59 a moins de r11MatchUS.
func r11Attribute(s r11Setup, u *r11Use, segs []r11Seg, imps []r11Imp) {
	u.Rank = AbilitySetNoRank
	for _, g := range segs {
		if g.Slot == u.Slot && u.TSUS >= g.From && u.TSUS < g.To {
			u.Rank, u.InLife, u.AgeUS = g.Rank, true, u.TSUS-g.From
			break
		}
	}
	if tr, ok := r11TrackAt(s, u.Slot, u.TSUS); ok {
		if n := s.art.Names[tr.XUID]; n != "" {
			u.Name = n
		}
	}
	for _, im := range imps {
		if im.Slot != u.Slot {
			continue
		}
		d := int64(im.TSUS) - int64(u.TSUS)
		if d < 0 {
			d = -d
		}
		if d <= r11MatchUS {
			u.Imp = true
			break
		}
	}
}

// r11Rows agrege par rang de palette.
func r11Rows(s r11Setup, segs []r11Seg, uses []r11Use, rd r11Reads) []r11Row {
	rows := map[int]*r11Row{}
	lives := map[int]map[string]bool{}
	get := func(rank int) *r11Row {
		if rows[rank] == nil {
			rows[rank] = &r11Row{Rank: rank, Label: r11RankLabel(s.art, rank),
				FirstNibbleHist: map[int]int{}, KHist: map[int]int{}}
			lives[rank] = map[string]bool{}
		}
		return rows[rank]
	}
	for _, g := range segs {
		r := get(g.Rank)
		r.Segs++
		r.ExposUS += g.To - g.From
		lives[g.Rank][fmt.Sprintf("%d/%d", g.Slot, r11Frame(s, g.From)/600)] = true
		for _, im := range rd.Imp {
			if im.Slot == g.Slot && im.TSUS >= g.From && im.TSUS < g.To {
				r.Imps++
			}
		}
		r11CountEnergy(get(g.Rank), g, rd.Energy)
	}
	for _, u := range uses {
		r := get(u.Rank)
		r.Uses++
		if u.AgeUS <= r11FreshUS {
			r.UsesFresh++
		}
		r.FirstNibbleHist[u.To]++
		if u.From != r11FullNibble {
			r.Consumed += u.From - u.To
		}
		if u.Imp {
			r.WithImp++
		}
	}
	var out []r11Row
	for rank, r := range rows {
		r.Lives = len(lives[rank])
		out = append(out, *r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Uses > out[b].Uses })
	return out
}

// r11CountEnergy compte les lectures d'i56 qui tombent dans un segment de port : le
// denominateur qui separe « le canal se tait sur cet equipement » de « il en parle sans
// jamais baisser ». Sans lui, un zero d'usages est illisible.
func r11CountEnergy(r *r11Row, g r11Seg, energy []r11EnergyRead) {
	for _, e := range energy {
		if e.Slot != g.Slot || e.TSUS < g.From || e.TSUS >= g.To {
			continue
		}
		r.Reads++
		for k := 0; k < AbilityEnergyCharges; k++ {
			if e.Ch[k] != AbilityEnergyUnarmed {
				r.KHist[k]++
			}
		}
		if e.Mask != 0 {
			r.Armed++
		}
	}
}

// r11LogRows publie le tableau par equipement.
func r11LogRows(t *testing.T, rows []r11Row) {
	t.Helper()
	t.Logf("  BAISSES DE CHARGE i56 par equipement (rang NOMME PAR LA PALETTE)")
	t.Logf("    %-28s %5s %5s %8s %6s %9s %6s %7s %9s %8s %6s",
		"equipement", "vies", "segs", "expo_min", "i56_lu", "i56_arme", "usages",
		"dont_<20s", "usages/min", "avec_imp", "imps")
	for _, r := range rows {
		mn := float64(r.ExposUS) / 60e6
		perMin := 0.0
		if mn > 0 {
			perMin = float64(r.UsesFresh) / mn
		}
		t.Logf("    %-28s %5d %5d %8.2f %6d %9d %6d %7d %9.3f %8d %6d",
			r.Label, r.Lives, r.Segs, mn, r.Reads, r.Armed, r.Uses, r.UsesFresh, perMin,
			r.WithImp, r.Imps)
	}
	for _, r := range rows {
		if r.Armed > 0 {
			t.Logf("    emplacements armes — %-24s e0=%d e1=%d e2=%d (charges consommees %d)",
				r.Label, r.KHist[0], r.KHist[1], r.KHist[2], r.Consumed)
		}
		if r.Uses == 0 {
			continue
		}
		var ks []int
		for k := range r.FirstNibbleHist {
			ks = append(ks, k)
		}
		sort.Ints(ks)
		var parts []string
		for _, k := range ks {
			parts = append(parts, fmt.Sprintf("%d:%d", k, r.FirstNibbleHist[k]))
		}
		t.Logf("    charges restantes apres usage — %s : %s", r.Label, strings.Join(parts, " "))
	}
}

// r11LogDetail publie chaque usage d'un equipement donne : c'est la liste de creneaux que
// l'utilisateur pourra confronter au Theater.
func r11LogDetail(t *testing.T, s r11Setup, uses []r11Use, detail string) {
	t.Helper()
	want := s.art.r9RankOf(detail)
	t.Logf("  DETAIL des usages de %q (rang %d dans ce film)", detail, want)
	if want < 0 {
		t.Logf("    ce film ne porte pas cet equipement dans sa palette")
		return
	}
	n := 0
	for _, u := range uses {
		if u.Rank != want {
			continue
		}
		n++
		t.Logf("    %-7s slot=%-4d %-17s e%d %d->%d  impulsion=%v  rang lu il y a %.0f s",
			r9MMSS(s.ms(u.TSUS)), u.Slot, u.Name, u.KSlot, u.From, u.To, u.Imp,
			float64(u.AgeUS)/1e6)
	}
	t.Logf("    -> %d usages", n)
}
