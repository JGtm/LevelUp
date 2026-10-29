//go:build research

package replay

// score_koth_emissions_research_test.go — E8.2 : LES EMISSIONS BRUTES DU SCORE DE MODE D'UN FILM
// KOTH, SLOT D'EQUIPE PAR SLOT D'EQUIPE, avant et apres les filtres de serie.
//
// LECTURE SEULE : les enregistrements viennent de la section statborg du fichier de FAITS
// (`data/cache/film_facts`), aucun octet de film n'est decode.
//
//	L1_DEPOT=<checkout qui porte data/cache> SCORE_FILMS=26602661,7de0b91d \
//	  go test -tags research -count=1 -run '^TestScoreKothEmissions$' -v ./internal/games/halo_infinite/film/replay/

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/facts/fallback"
	"github.com/JGtm/LevelUp/libs/film/internal/facts/objectives"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// chargerStatborgDesFaits lit la section statborg d'un film depuis le cache de faits du poste.
func chargerStatborgDesFaits(t *testing.T, depot, film string) *FilmFactsFile {
	t.Helper()
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(depot, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	blob, err := os.ReadFile(filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite", film+".filmfacts.bin")) //nolint:gosec // mesure
	if err != nil {
		t.Fatalf("faits de %s : %v", film, err)
	}
	head, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete de %s : %v", film, err)
	}
	var entry profile.MapQuantEntry
	for _, c := range cat.Maps {
		if c.Module == head.MapModule {
			entry = c
			break
		}
	}
	ff, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("faits de %s : %v", film, err)
	}
	return ff
}

func TestScoreKothEmissions(t *testing.T) {
	depot := os.Getenv("L1_DEPOT")
	films := os.Getenv("SCORE_FILMS")
	if depot == "" || films == "" {
		t.Skip("L1_DEPOT et SCORE_FILMS requis")
	}
	for _, film := range strings.Split(films, ",") {
		ff := chargerStatborgDesFaits(t, depot, film)
		recs := ff.Statborg.Records
		t.Logf("== %s : %d enregistrements ; manches reelles %v ; ecrites %v ; bornes posees=%v",
			film, len(recs), objectives.RealRounds(recs), objectives.ResolveRounds(recs).Written,
			objectives.ResolveRoundBounds(recs).Posed())
		type cle struct{ slot, round int }
		var ordre []cle
		vus := map[cle][]types.StatRecord{}
		for _, r := range recs {
			if !objectives.IsTeamSlot(r.Slot) {
				continue
			}
			if _, ok := r.Comps[0]; !ok {
				continue
			}
			k := cle{r.Slot, r.Round}
			if _, ok := vus[k]; !ok {
				ordre = append(ordre, k)
			}
			vus[k] = append(vus[k], r)
		}
		for _, k := range ordre {
			var sb strings.Builder
			prev := types.StatValue{A: -1, B: -1}
			n := 0
			flush := func(last types.StatRecord) {
				if n > 1 {
					sb.WriteString(" x" + strconv.Itoa(n))
				}
				_ = last
			}
			var derniere types.StatRecord
			for _, r := range vus[k] {
				v := r.Comps[0]
				if v != prev {
					flush(derniere)
					sb.WriteString("\n      t=" + strconv.Itoa(r.TimeMS) + " A=" + strconv.Itoa(int(v.A)) + " B=" + strconv.Itoa(int(v.B)))
					prev, n = v, 0
				}
				n++
				derniere = r
			}
			flush(derniere)
			t.Logf("  slot %d manche %d (%d emissions) :%s", k.slot, k.round, len(vus[k]), sb.String())
		}
		for _, teams := range []bool{true} {
			tot := objectives.SeriesTotal(recs, objectives.ModeScoreComponent(), teams, nil)
			byRound := objectives.SeriesByRound(recs, objectives.ModeScoreComponent(), teams, nil)
			for slot, pts := range tot {
				t.Logf("  SeriesTotal slot %d : %v", slot, pts)
			}
			for slot, br := range byRound {
				for round, pts := range br {
					t.Logf("  SeriesByRound slot %d manche %d : %v", slot, round, pts)
				}
			}
		}
	}
}

// TestScoreKothPopulationB — OU VIT UN CANAL B NON NUL SUR LES SLOTS D'EQUIPE (composant 0) ?
//
// Parcourt, un par un et sans decoder de film, les faits du cache du poste.
func TestScoreKothPopulationB(t *testing.T) {
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		t.Skip("L1_DEPOT requis")
	}
	dir := filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cache de faits : %v", err)
	}
	films, avecB, total, totalB := 0, 0, 0, 0
	paires := map[[2]int64]int{}
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".filmfacts.bin") {
			continue
		}
		film := strings.TrimSuffix(e.Name(), ".filmfacts.bin")
		blob, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // mesure
		if err != nil {
			continue
		}
		if _, err := DecodeFilmFactsEntete(blob); err != nil {
			continue
		}
		ff := chargerStatborgDesFaits(t, depot, film)
		films++
		nB := 0
		for _, r := range ff.Statborg.Records {
			if !objectives.IsTeamSlot(r.Slot) {
				continue
			}
			v, ok := r.Comps[0]
			if !ok {
				continue
			}
			total++
			if v.B != 0 {
				nB++
				totalB++
				paires[[2]int64{v.A, v.B}]++
			}
		}
		if nB > 0 {
			avecB++
			t.Logf("  %s : %d emission(s) d'equipe avec B != 0", film, nB)
		}
	}
	t.Logf("%d films ; %d emissions d'equipe comp 0 ; %d avec B != 0 dans %d films ; paires (A,B) : %v",
		films, total, totalB, avecB, paires)
}

// TestScoreKothParc — LES SERIES D'EQUIPE DU SCORE DE MODE DE TOUS LES FILMS DU CACHE DE FAITS,
// ecrites une par film dans SCORE_SORTIE pour comparer deux etats du code (diff avant/apres).
//
// Sans decoder de film : la section statborg des faits suffit. SCORE_REGISTRE (TSV
// `film t0 t1 variante ...`) ajoute le score du registre, la verite terrain de l'affichage.
func TestScoreKothParc(t *testing.T) {
	depot, sortie := os.Getenv("L1_DEPOT"), os.Getenv("SCORE_SORTIE")
	if depot == "" || sortie == "" {
		t.Skip("L1_DEPOT et SCORE_SORTIE requis")
	}
	registre := map[string][]string{}
	if f := os.Getenv("SCORE_REGISTRE"); f != "" {
		blob, err := os.ReadFile(f) //nolint:gosec // mesure
		if err != nil {
			t.Fatalf("registre : %v", err)
		}
		for _, l := range strings.Split(string(blob), "\n")[1:] {
			if c := strings.Split(strings.TrimRight(l, "\r"), "\t"); len(c) >= 4 {
				registre[c[0]] = c
			}
		}
	}
	dir := filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cache de faits : %v", err)
	}
	var sb strings.Builder
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".filmfacts.bin") {
			continue
		}
		film := strings.TrimSuffix(e.Name(), ".filmfacts.bin")
		blob, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // mesure
		if err != nil {
			continue
		}
		if _, err := DecodeFilmFactsEntete(blob); err != nil {
			continue
		}
		recs := chargerStatborgDesFaits(t, depot, film).Statborg.Records
		nB := 0
		for _, r := range recs {
			if v, ok := r.Comps[0]; ok && objectives.IsTeamSlot(r.Slot) && v.B != 0 {
				nB++
			}
		}
		reg := registre[film]
		if len(reg) < 4 {
			reg = []string{film, "?", "?", "?"}
		}
		sb.WriteString(film + "\t" + reg[3] + "\t" + reg[1] + "-" + reg[2] + "\tB=" + strconv.Itoa(nB))
		tot := objectives.SeriesTotal(recs, objectives.ModeScoreComponent(), true, nil)
		rnd := objectives.SeriesByRound(recs, objectives.ModeScoreComponent(), true, nil)
		for _, slot := range []int{6, 8} {
			sb.WriteString("\ts" + strconv.Itoa(slot) + " total=")
			for _, p := range tot[slot] {
				sb.WriteString(strconv.Itoa(p.TimeMS) + ":" + strconv.FormatInt(p.Value, 10) + " ")
			}
			sb.WriteString("rounds=")
			for r := 0; r < 10; r++ {
				pts := rnd[slot][r]
				if len(pts) == 0 {
					continue
				}
				sb.WriteString("r" + strconv.Itoa(r) + "[")
				for _, p := range pts {
					sb.WriteString(strconv.Itoa(p.TimeMS) + ":" + strconv.FormatInt(p.Value, 10) + " ")
				}
				sb.WriteString("]")
			}
		}
		sb.WriteString("\n")
	}
	if err := os.WriteFile(sortie, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	t.Logf("%d lignes ecrites dans %s", strings.Count(sb.String(), "\n"), sortie)
}

// TestScoreKothSeuil — LE CALQUE DE SCORE D'UN FILM KOTH, ASSEMBLE COMME LA CUISSON, depuis les
// faits : courbes d'equipe, seuil de garde lu dans le film, contradiction avec la table.
//
// Les lignes de joueurs et le score du registre viennent de SCORE_JOUEURS (TSV `film xuid team
// kills deaths assists t0 t1 variante`, extrait de la copie de la base) ; l'horloge, de
// l'artefact deja cuit quand il existe (`originMs`, `frameCount`), sinon d'une grille a origine
// nulle. Rien n'est ecrit hors de SCORE_SORTIE.
func TestScoreKothSeuil(t *testing.T) {
	depot, sortie, joueurs := os.Getenv("L1_DEPOT"), os.Getenv("SCORE_SORTIE"), os.Getenv("SCORE_JOUEURS")
	if depot == "" || sortie == "" || joueurs == "" {
		t.Skip("L1_DEPOT, SCORE_SORTIE et SCORE_JOUEURS requis")
	}
	type fiche struct {
		lines    []types.PlayerLine
		teams    map[string]int
		t0, t1   int
		variante string
	}
	fiches := map[string]*fiche{}
	blob, err := os.ReadFile(joueurs) //nolint:gosec // mesure
	if err != nil {
		t.Fatalf("joueurs : %v", err)
	}
	for _, l := range strings.Split(string(blob), "\n")[1:] {
		c := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(c) < 9 {
			continue
		}
		f := fiches[c[0]]
		if f == nil {
			f = &fiche{teams: map[string]int{}, variante: c[8]}
			f.t0, _ = strconv.Atoi(c[6])
			f.t1, _ = strconv.Atoi(c[7])
			fiches[c[0]] = f
		}
		k, _ := strconv.Atoi(c[3])
		d, _ := strconv.Atoi(c[4])
		a, _ := strconv.Atoi(c[5])
		team, _ := strconv.Atoi(c[2])
		f.lines = append(f.lines, types.PlayerLine{XUID: c[1], Kills: k, Deaths: d, Assists: a})
		f.teams[c[1]] = team
	}
	tables := map[string]int{"KOTH:Arena": 35, "Doubles:King of the Hill": 35, "Ranked:King of the Hill": 40}
	var films []string
	for film := range fiches {
		films = append(films, film)
	}
	sort.Strings(films)
	var sb strings.Builder
	for _, film := range films {
		if _, err := os.Stat(filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite", film+".filmfacts.bin")); err != nil {
			continue
		}
		blobF, err := os.ReadFile(filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite", film+".filmfacts.bin")) //nolint:gosec // mesure
		if err != nil {
			continue
		}
		if _, err := DecodeFilmFactsEntete(blobF); err != nil {
			sb.WriteString(film + "\tFAITS ILLISIBLES (schema)\n")
			continue
		}
		f := fiches[film]
		recs := chargerStatborgDesFaits(t, depot, film).Statborg.Records
		origine, frames := 0, 200000
		if art, err := os.ReadFile(filepath.Join(depot, "data", "cache", "replays", "halo_infinite", film+".json")); err == nil { //nolint:gosec // mesure
			var h struct {
				OriginMs   int `json:"originMs"`
				FrameCount int `json:"frameCount"`
			}
			if json.Unmarshal(art, &h) == nil && h.FrameCount > 0 {
				origine, frames = h.OriginMs, h.FrameCount
			}
		}
		in := &ScoreInput{
			Records: recs, Lines: f.lines, TeamByXUID: f.teams, TeamScores: &[2]int{f.t0, f.t1},
			HillScoring: true, HoldTicksPerPointTable: tables[f.variante],
		}
		fb := fallback.NouveauCompteur()
		tl, _, lu := assembleScoreTimeline(in, nil, scoreClock{intervalMS: 100, frames: frames, originMS: origine}, fb)
		sb.WriteString(film + "\t" + f.variante + "\t" + strconv.Itoa(f.t0) + "-" + strconv.Itoa(f.t1))
		if tl == nil {
			sb.WriteString("\tpas de calque\n")
			continue
		}
		for _, tm := range tl.Teams {
			id := "?"
			if tm.TeamID != nil {
				id = strconv.Itoa(*tm.TeamID)
			}
			sb.WriteString("\tcamp" + id + "=")
			for _, k := range tm.Total {
				sb.WriteString(strconv.Itoa(k.T) + ":" + strconv.Itoa(k.V) + " ")
			}
		}
		seuil := 0
		if tl.HoldTicksPerPoint != nil {
			seuil = *tl.HoldTicksPerPoint
		}
		sb.WriteString("\tgains=")
		if tl.HoldTicks != nil {
			g, _ := holdGainsAtPoints(tl.HoldTicks, tl.Teams)
			sb.WriteString(fmt.Sprint(g))
		}
		sb.WriteString("\tSEUIL=" + strconv.Itoa(seuil))
		if lu != nil {
			sb.WriteString(" film=" + strconv.Itoa(lu.film) + " points=" + strconv.Itoa(lu.points) +
				" ecartes=" + strconv.Itoa(lu.rejected) + " table=" + strconv.Itoa(lu.table) +
				" contredite=" + strconv.FormatBool(lu.tableDisagrees) + " source=" + lu.source)
		}
		sb.WriteString(" compteur_contredite=" + strconv.Itoa(fb.Compte(fallback.NomSeuilGardeTableContredite)) + "\n")
	}
	if err := os.WriteFile(sortie, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	t.Logf("%s", sb.String())
}

// TestScoreKothDetailGarde — POUR UN FILM (SCORE_DETAIL), le detail des tics de garde de chaque
// camp autour de chaque point : image du point, gain, premier et dernier tic de l'intervalle, et
// ses ecarts anormaux (> 11 images entre deux tics, un tic attendu par seconde).
func TestScoreKothDetailGarde(t *testing.T) {
	depot, joueurs, film := os.Getenv("L1_DEPOT"), os.Getenv("SCORE_JOUEURS"), os.Getenv("SCORE_DETAIL")
	if depot == "" || joueurs == "" || film == "" {
		t.Skip("L1_DEPOT, SCORE_JOUEURS et SCORE_DETAIL requis")
	}
	lines, teams := []types.PlayerLine{}, map[string]int{}
	t0, t1 := 0, 0
	blob, err := os.ReadFile(joueurs) //nolint:gosec // mesure
	if err != nil {
		t.Fatalf("joueurs : %v", err)
	}
	for _, l := range strings.Split(string(blob), "\n")[1:] {
		c := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(c) < 9 || c[0] != film {
			continue
		}
		k, _ := strconv.Atoi(c[3])
		d, _ := strconv.Atoi(c[4])
		a, _ := strconv.Atoi(c[5])
		team, _ := strconv.Atoi(c[2])
		t0, _ = strconv.Atoi(c[6])
		t1, _ = strconv.Atoi(c[7])
		lines = append(lines, types.PlayerLine{XUID: c[1], Kills: k, Deaths: d, Assists: a})
		teams[c[1]] = team
	}
	recs := chargerStatborgDesFaits(t, depot, film).Statborg.Records
	origine, frames := 0, 200000
	if art, err := os.ReadFile(filepath.Join(depot, "data", "cache", "replays", "halo_infinite", film+".json")); err == nil { //nolint:gosec // mesure
		var h struct {
			OriginMs   int `json:"originMs"`
			FrameCount int `json:"frameCount"`
		}
		if json.Unmarshal(art, &h) == nil && h.FrameCount > 0 {
			origine, frames = h.OriginMs, h.FrameCount
		}
	}
	in := &ScoreInput{Records: recs, Lines: lines, TeamByXUID: teams, TeamScores: &[2]int{t0, t1},
		HillScoring: true, HoldTicksPerPointTable: 40}
	tl, _, _ := assembleScoreTimeline(in, nil, scoreClock{intervalMS: 100, frames: frames, originMS: origine}, fallback.NouveauCompteur())
	if tl == nil || tl.HoldTicks == nil {
		t.Fatalf("pas de barre de garde")
	}
	serie := map[int][]ScoreTick{}
	for _, h := range tl.HoldTicks {
		if h.TeamID != nil {
			serie[*h.TeamID] = h.Ticks
		}
	}
	since := -1
	for _, p := range scoredPoints(tl.Teams) {
		own := serie[p.team]
		var ts []int
		for _, k := range own {
			if k.T > since && k.T <= p.t+30 {
				ts = append(ts, k.T)
			}
		}
		gain, ok := coherentGain(serie, p, since)
		t.Logf("point camp %d image %d (depuis %d) : coherent=%v gain=%d ; tics (image) = %v", p.team, p.t, since, ok, gain, ts)
		since = p.t
	}
}
