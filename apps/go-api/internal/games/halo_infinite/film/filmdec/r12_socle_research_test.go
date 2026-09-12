package filmdec

// r12_socle_research_test.go — LE SOCLE DU LOT R12 : un film SANS ARTEFACT ET SANS BORNES.
//
// POURQUOI IL EXISTE. Le film ancre de R12 (`215e7022`, Argyle) n'a NI artefact de rejeu
// (sa carte n'est pas au catalogue de bornes, la cuisson echoue volontairement) NI entree
// dans `map_quant_bounds.json`. Tous les instruments R8/R9/R11 exigent les deux :
// `r9LoadArt` fait `t.Fatalf` sans artefact, `r8MapEntry` fait `t.Fatalf` quand aucune carte
// du catalogue ne porte les largeurs du film. Ce fichier est le socle qui s'en passe.
//
// CE QUE L'ABSENCE DE BORNES NE COUTE PAS, ET C'EST LE POINT.
// `SetWorldObjectPrecisionFromLayout` (traverse.go:183) ne lit que `AxisW` et `GateBits` de
// l'`I0Layout`, et `DetectI0Layout(dir)` les rend DEPUIS LE FILM. Les bornes metriques ne
// servent qu'a DEQUANTIFIER une position en metres. Aucun canal vise par R12 (i48, i56,
// i57/i59, masque, evenements, ti=37) n'a besoin de metres : seules les POSITIONS en
// auraient besoin, et R12 n'en publie aucune.
//
// CE QUE L'ABSENCE D'ARTEFACT COUTE, ET COMMENT C'EST REMPLACE.
//   - la PALETTE (`abilityLabels`) : remplacee par `r12ClassifyPalette`, qui applique la
//     regle de `replay.classifyAbilityPalette` (majorite a 90 %) sur les rangs i48 observes,
//     avec les marqueurs et les noms de `config/titles/halo_infinite/mappings/replay_labels.toml`
//     RECOPIES ICI (un test de recherche ne charge pas la config du titre). La recopie est
//     signalee comme telle et bornee aux deux familles connues.
//   - les GAMERTAGS par slot : NON remplaces ici. R12 n'en a pas besoin — il travaille sur
//     des SLOTS, et l'identite des porteurs vient du canal i48 (le rang porte), pas d'un nom.
//     Le pont vers les gamertags est fait a part, par `killsource` (instrument
//     `r12_ancre_kill_research_test.go`).
//
// PIEGE HERITE DE R8, RECOPIE EXPRES : `WorldObjectPrecision` est un GLOBAL DE PAQUET. Un
// instrument qui oublie `SetWorldObjectPrecisionFromLayout` desaligne les desers sans lever
// la moindre erreur (13 poses au lieu de 537, mesure de R8).
//
// GARDES : `R12_FILMS`, `R12_IDS`. Aucune ecriture, aucune DuckDB, `CGO_ENABLED=0`.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	r12FilmsEnv = "R12_FILMS"
	r12IDsEnv   = "R12_IDS"
)

// --- LE SETUP -----------------------------------------------------------------------------

// r12Setup porte un film prepare : le decoupage, l'archetype, la bande de slots, l'origine
// d'horloge. AUCUN artefact, AUCUNE borne metrique.
type r12Setup struct {
	id     string
	dir    string
	chunks []int
	slots  SlotBand
	lay    I0Layout
	arch   Archetype
	origin uint64
}

// ms convertit un horodatage moteur en temps du visionneur (millisecondes depuis le debut du
// film). Meme convention que `r11Setup.ms` et que `killsource.Kill.TimeMS`.
func (s r12Setup) ms(ts uint64) int64 { return (int64(ts) - int64(s.origin)) / 1000 }

// r12Prepare resout un film. L'appelant DOIT detenir `LockProcessDecode` et restaurer
// `WorldObjectPrecision` en sortie.
func r12Prepare(t *testing.T, dir string) r12Setup {
	t.Helper()
	id := filepath.Base(dir)
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("%s : aucun chunk film", id)
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBandDir(dir, chunks)
	if slots.Count() == 0 {
		t.Fatalf("%s : aucun slot biped dans les keyframes", id)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("%s : decoupage i0 illisible : %v", id, err)
	}
	SetWorldObjectPrecisionFromLayout(lay)
	arch, err := bipedArchetypeDir(dir)
	if err != nil {
		t.Fatalf("%s : archetype biped illisible : %v", id, err)
	}
	origin, ok := r12FirstPacketUS(dir, 1)
	if !ok {
		t.Fatalf("%s : chunk 1 illisible, aucune origine d'horloge", id)
	}
	return r12Setup{id: id, dir: dir, chunks: chunks, slots: slots, lay: lay,
		arch: arch, origin: origin}
}

// r12FirstPacketUS rend l'horodatage du PREMIER paquet du chunk demande. Le manifeste du film
// donne `start_ms = 0` au chunk 1 : c'est l'origine de l'horloge du visionneur.
func r12FirstPacketUS(dir string, chunk int) (uint64, bool) {
	data, err := ReadFilmChunk(dir, chunk)
	if err != nil {
		return 0, false
	}
	for _, pk := range WalkPackets(data) {
		if pk.TimestampUS != 0 {
			return pk.TimestampUS, true
		}
	}
	return 0, false
}

// r12MMSS met un instant en millisecondes sous la forme du curseur du visionneur.
func r12MMSS(ms int64) string {
	sign := ""
	if ms < 0 {
		sign, ms = "-", -ms
	}
	return fmt.Sprintf("%s%d:%02d.%01d", sign, ms/60000, ms/1000%60, ms%1000/100)
}

// r12FilmDirs rend les dossiers a traiter. `R12_IDS` (liste explicite) OU `R12_LIMIT` (les N
// premiers films de la racine, par ordre alphabetique) — l'un des deux est OBLIGATOIRE : un
// film coute des dizaines de secondes de decodage, un balayage non borne serait un piege.
func r12FilmDirs(t *testing.T) []string {
	t.Helper()
	root := os.Getenv(r12FilmsEnv)
	if root == "" {
		t.Skipf("%s absent : instrument saute", r12FilmsEnv)
	}
	var out []string
	for _, s := range strings.Split(os.Getenv(r12IDsEnv), ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, filepath.Join(root, s))
		}
	}
	if len(out) > 0 {
		return out
	}
	lim, _ := strconv.Atoi(os.Getenv("R12_LIMIT"))
	if lim <= 0 {
		t.Fatalf("%s ou R12_LIMIT obligatoire avec %s", r12IDsEnv, r12FilmsEnv)
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("racine %s illisible : %v", root, err)
	}
	skip, _ := strconv.Atoi(os.Getenv("R12_SKIP"))
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		out = append(out, filepath.Join(root, e.Name()))
		if len(out) >= lim {
			break
		}
	}
	return out
}

// --- LES LECTURES ---------------------------------------------------------------------------

// r12RankRead est UNE lecture d'i48 : le compteur de rotation et le rang de palette.
type r12RankRead struct {
	Slot    uint32
	MS      int64
	Counter uint32
	Rank    int
}

// r12EnergyRead est UNE lecture d'i56 (le canal des charges trouve par R11).
type r12EnergyRead struct {
	Slot uint32
	MS   int64
	Mask uint32
	Ch   [AbilityEnergyCharges]int
}

// r12TagRead est UNE lecture de tag d'i57 ou d'i59.
type r12TagRead struct {
	Slot uint32
	MS   int64
	Src  string
	Tag  uint64
}

// r12Reads est la recolte d'un passage.
type r12Reads struct {
	Ranks  []r12RankRead
	Energy []r12EnergyRead
	Tags   []r12TagRead
	Stat   map[string]int
}

var (
	r12I48Names = []string{"biped-desired-ability-set-component", "biped-desired-ability-set"}
	r12I56Names = []string{
		"biped-spartan-ability-energy-component", "biped-spartan-ability-energy",
	}
	r12I57Names = []string{"biped-spartan-ability-component", "biped-spartan-ability"}
	r12I59Names = []string{
		"biped-spartan-ability-non-predicted-state-component",
		"biped-spartan-ability-non-predicted-state",
	}
)

// r12Collect fait UN passage sur les paquets delta et recolte les quatre canaux de capacite
// par les hooks des desers de PRODUCTION. Aucune relecture posee a cote d'eux.
// L'appelant detient `LockProcessDecode` : les hooks sont des globaux de paquet.
func r12Collect(s r12Setup) r12Reads {
	idx := map[string]int{
		"i48": r8IndexOfAny(s.arch, r12I48Names),
		"i56": r8IndexOfAny(s.arch, r12I56Names),
		"i57": r8IndexOfAny(s.arch, r12I57Names),
		"i59": r8IndexOfAny(s.arch, r12I59Names),
	}
	out := r12Reads{Stat: map[string]int{}}
	var cur struct {
		slot uint32
		ms   int64
	}
	prev48, prev56 := abilitySetHook, abilityEnergyHook
	prev57, prev59 := spartanAbilityHook, abilityNonPredictedHook
	SetAbilitySetHook(func(counter uint64, rank, _ int) {
		out.Stat["i48"]++
		out.Ranks = append(out.Ranks, r12RankRead{cur.slot, cur.ms, uint32(counter), rank})
	})
	SetAbilityEnergyHook(func(mask uint32, ch [AbilityEnergyCharges]int) {
		out.Stat["i56"]++
		out.Energy = append(out.Energy, r12EnergyRead{cur.slot, cur.ms, mask, ch})
	})
	SetSpartanAbilityHook(func(tag, _, _ uint64, _ bool) {
		out.Stat[fmt.Sprintf("i57tag%d", tag)]++
		out.Tags = append(out.Tags, r12TagRead{cur.slot, cur.ms, "i57", tag})
	})
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) {
		out.Stat[fmt.Sprintf("i59tag%d", st.Tag)]++
		out.Tags = append(out.Tags, r12TagRead{cur.slot, cur.ms, "i59", uint64(st.Tag)})
	})
	defer func() {
		SetAbilitySetHook(prev48)
		SetAbilityEnergyHook(prev56)
		SetSpartanAbilityHook(prev57)
		SetAbilityNonPredictedHook(prev59)
	}()
	r12Walk(s, func(slot uint32, ms int64, ids []int) bool {
		any := false
		for _, k := range []string{"i48", "i56", "i57", "i59"} {
			if idx[k] >= 0 && maskHas(ids, idx[k]) {
				out.Stat["masque"+k]++
				any = true
			}
		}
		if any {
			cur.slot, cur.ms = slot, ms
		}
		return any
	})
	return out
}

// r12Walk marche les records delta de bipede. `want` decide, sur le masque, si le record doit
// etre deroule ; il publie aussi le slot et l'instant a l'appelant.
func r12Walk(s r12Setup, want func(slot uint32, ms int64, ids []int) bool) {
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
			ms := s.ms(pk.TimestampUS)
			for p := 0; p+minRecord <= total; {
				i0, slot, ids, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				if want(slot, ms, ids) {
					walkRecordComponents(pay, i0, total, ids, s.lay, s.arch,
						func(int) bool { return true })
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
}

// --- L'INSTRUMENT D'ANCRAGE (mesure A2) -----------------------------------------------------

// r12Ancre est un instant du releve Theater.
type r12Ancre struct {
	nom string
	ms  int64
}

// r12AncresUsage : les CINQ usages de repulseur releves au Theater sur `215e7022`, en temps
// de FILM (convention etablie au par. 0.1 du rapport R12, confirmee par `TestR12AncreKill`).
var r12AncresUsage = []r12Ancre{
	{"U1 Elmo910 (kill)", 5*60000 + 25000},
	{"U2 Bot ziker", 8*60000 + 14000},
	{"U3 Bot ziker", 8*60000 + 20000},
	{"U4 Elmo910", 9*60000 + 54000},
	{"U5 Elmo910", 10*60000 + 5000},
}

// r12AncresPrise : les QUATRE ramassages releves.
var r12AncresPrise = []r12Ancre{
	{"P1 JGtm", 3*60000 + 48000},
	{"P2 Elmo910", 4*60000 + 55000},
	{"P3 Bot ziker", 8*60000 + 14000},
	{"P4 Elmo910", 9*60000 + 50000},
}

// TestR12Ancrage publie le journal complet d'i48 (le rang porte, tous slots) et juge la
// mesure A2 pre-inscrite : au moins 2 des 4 ramassages apparies a +/- 5 s, et strictement
// plus que le temoin decale de +30 s.
func TestR12Ancrage(t *testing.T) {
	for _, dir := range r12FilmDirs(t) {
		r12AncrageOneFilm(t, dir)
	}
}

func r12AncrageOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r12Prepare(t, dir)
	rd := r12Collect(s)

	pal := r12ClassifyPalette(rd.Ranks)
	palID := "non classee"
	if pal != nil {
		palID = pal.id
	}
	rep := pal.r12RankOf("Repulsor")
	t.Logf("=== FILM %s ===", s.id)
	t.Logf("  layout AxisW=%v region=%d gate=%d | %d chunks | %d slots biped",
		s.lay.AxisW, s.lay.Region, s.lay.GateBits, len(s.chunks), s.slots.Count())
	t.Logf("  denominateurs : %v", r12SortedStat(rd.Stat))
	t.Logf("  palette=%s rangRepulseur=%d", palID, rep)

	hist := map[int]int{}
	for _, r := range rd.Ranks {
		hist[r.Rank]++
	}
	t.Logf("  histogramme des rangs i48 : %s", r12HistTxt(hist, pal))

	t.Logf("  JOURNAL i48 (%d lectures) :", len(rd.Ranks))
	sort.SliceStable(rd.Ranks, func(a, b int) bool { return rd.Ranks[a].MS < rd.Ranks[b].MS })
	for _, r := range rd.Ranks {
		mark := ""
		if r.Rank == rep && rep >= 0 {
			mark = "   <<<< REPULSEUR"
		}
		t.Logf("    %-9s slot=%-5d compteur=%d rang=%-3d %s%s",
			r12MMSS(r.MS), r.Slot, r.Counter, r.Rank, pal.r12LabelOf(r.Rank), mark)
	}

	if rep < 0 {
		t.Logf("  MESURE A2 : la palette ne nomme aucun rang « Repulsor » — appariement impossible")
		return
	}
	var repTimes []int64
	for _, r := range rd.Ranks {
		if r.Rank == rep {
			repTimes = append(repTimes, r.MS)
		}
	}
	t.Logf("  MESURE A2 — appariement des ramassages (tolerance +/- 5 s) :")
	hit := r12Pair(t, r12AncresPrise, repTimes, 5000, 0)
	dec := r12Pair(t, r12AncresPrise, repTimes, 5000, 30000)
	// TEMOIN RESSERRE : a +/- 5 s avec 11 lectures de rang 6 dans un film de 11 minutes, le
	// temoin decale peut apparier par hasard. A +/- 2,5 s il ne le peut presque plus, et c'est
	// la BANDE des ecarts (leur dispersion) qui porte la preuve.
	hit2 := r12Pair(t, r12AncresPrise, repTimes, 2500, 0)
	dec2 := r12Pair(t, r12AncresPrise, repTimes, 2500, 30000)
	dec3 := r12Pair(t, r12AncresPrise, repTimes, 2500, -30000)
	t.Logf("  MESURE A2 — VERDICT : %d/4 apparies a +/-5 s (seuil pre-inscrit 2), "+
		"temoin decale +30 s : %d/4", hit, dec)
	t.Logf("  MESURE A2 — resserre a +/-2,5 s : %d/4 ; temoins decales +30 s : %d/4, "+
		"-30 s : %d/4", hit2, dec2, dec3)
}

// r12Pair apparie des ancres a des instants, avec un decalage optionnel (le temoin).
func r12Pair(t *testing.T, anc []r12Ancre, times []int64, tol, shift int64) int {
	t.Helper()
	hit := 0
	for _, a := range anc {
		best, bestD := int64(0), int64(1<<62)
		for _, x := range times {
			d := x - (a.ms + shift)
			if d < 0 {
				d = -d
			}
			if d < bestD {
				best, bestD = x, d
			}
		}
		ok := bestD <= tol
		if ok {
			hit++
		}
		if shift == 0 {
			t.Logf("    %-16s ancre=%-9s plus proche=%-9s ecart=%+6d ms  %v",
				a.nom, r12MMSS(a.ms), r12MMSS(best), best-a.ms, ok)
		}
	}
	return hit
}

func r12SortedStat(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%d ", k, m[k])
	}
	return b.String()
}

func r12HistTxt(h map[int]int, pal *r12Palette) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%d(%s):%d  ", k, pal.r12LabelOf(k), h[k])
	}
	return b.String()
}
