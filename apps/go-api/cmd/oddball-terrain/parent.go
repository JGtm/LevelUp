package main

// parent.go — LE PARENT : orchestre un enfant filmproc par film, capte son dump, puis aligne et
// confronte. Aucune sentinelle ici (le parent ne decode rien) ; aucune base (le gamertag vient du
// film). L'oracle vient d'un JSON fige, releve de `match_objective_stats_latest`.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// terrainFilm : le film calibrant (verite terrain).
const terrainFilm = "d9781168"

// oracleStat porte les colonnes de l'oracle utilisees ici.
type oracleStat struct {
	Time   float64 `json:"time_as_skull_carrier_seconds"`
	Grabs  int     `json:"skull_grabs"`
	Ticks  int     `json:"skull_scoring_ticks"`
	Killed int     `json:"skull_carriers_killed"`
}

// point est un point de serie (instant, valeur) sur l'horloge du match.
type point struct {
	t int
	v int64
}

// emplKey identifie un emplacement du statborg.
type emplKey struct {
	comp int
	side string
}

// filmDump : tout ce que l'enfant a rendu d'un film.
type filmDump struct {
	id        string
	nrec      int
	truncated bool
	gamertags map[uint64]string
	deaths    map[uint64][]int
	// ident est PAR MANCHE : round -> slot -> xuid. Le slot est reattribue d'une manche a
	// l'autre (cf. identity.go), l'identite ne peut donc pas etre un simple slot -> xuid.
	ident  map[int]map[int]string
	rounds []int
	series map[int]map[int]map[emplKey][]point // slot -> round -> empl -> points
}

// identOf rend le xuid du slot pour une manche donnee, ou "" s'il n'est pas resolu.
func identOf(d *filmDump, round, slot int) string {
	if m := d.ident[round]; m != nil {
		return m[slot]
	}
	return ""
}

func runParent(repoRoot, cache, outDir, oraclePath string) error {
	oracle, err := loadOracle(oraclePath)
	if err != nil {
		return err
	}
	films := oddballCorpus(cache, oracle)
	fmt.Printf("corpus Oddball (cache inter oracle) : %d film(s) — %s\n", len(films), strings.Join(films, ", "))

	runner := &runnerHolder{repoRoot: repoRoot}
	ctx := context.Background()

	// 1) Decoder tout le corpus (un enfant filmproc par film), garder les dumps.
	dumps := map[string]*filmDump{}
	for _, id := range films {
		d, err := runner.decode(ctx, cache, id)
		if err != nil {
			fmt.Printf("  %s : NON decode (%v)\n", id, err)
			continue
		}
		dumps[id] = d
		fmt.Printf("  %s : %d enregistrements, tronque=%v, %d slots nommes (manche 0), manches=%v\n",
			id, d.nrec, d.truncated, len(d.ident[0]), d.rounds)
	}

	// 2) L'identite slot -> xuid vient des INSTANTS DE MORT (identity.go), calculee dans decode() :
	// le seul pont fiable ici. Les totaux (triplet API, tics d'oracle) sont fausses par la
	// troncature Theater de la derniere manche ; les INSTANTS de mort, eux, sont complets dans le
	// fil du film et coincident a ~3 ms avec les increments du compteur de morts du statborg.

	// 3) Identifier les emplacements Oddball (grabs, tics) par l'oracle, films confondus.
	grabsKey, ticksKey, identLog := identifyEmplacements(dumps, oracle, films)

	// 4) Film calibrant : confrontation manche 1 (le gate central).
	td := dumps[terrainFilm]
	scoresLog, confLog, verdict3 := confrontTerrain(td, oracle[terrainFilm], grabsKey, ticksKey)
	diag := ""
	if td != nil {
		diag = identityDiagRound(td, 0) + "\n"
	}
	writeLog(outDir+"/TERRAIN_scores.log", identLog+"\n"+diag+scoresLog)
	writeLog(outDir+"/TERRAIN_confrontation.log", confLog)
	fmt.Print(verdict3)

	// 5) Gate oracle sur tout le corpus (recouvrement du temps de portage).
	gateLog := gateOracle(dumps, films, oracle, ticksKey)
	writeLog(outDir+"/TERRAIN_gate_oracle.log", gateLog)
	return nil
}

func writeLog(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "ecriture %s : %v\n", path, err)
	}
}

// oddballCorpus rend les films de l'oracle dont le cache porte les chunks, ordonnes.
func oddballCorpus(cache string, oracle map[string]map[string]oracleStat) []string {
	var out []string
	for id := range oracle {
		if _, err := os.Stat(filmcache.ChunkDir(cache, id)); err == nil {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// loadOracle lit l'oracle fige film -> xuid -> stats.
func loadOracle(path string) (map[string]map[string]oracleStat, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("oracle : %w", err)
	}
	var out map[string]map[string]oracleStat
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("oracle JSON : %w", err)
	}
	return out, nil
}

// runnerHolder porte la racine du depot et re-cree un tampon de capture par film.
type runnerHolder struct {
	repoRoot string
}

// decode lance l'enfant filmproc sur -id, capte sa sortie et la parse.
func (h *runnerHolder) decode(ctx context.Context, cache, id string) (*filmDump, error) {
	var buf bytes.Buffer
	r, err := filmproc.NewRunner(h.repoRoot, &buf)
	if err != nil {
		return nil, err
	}
	res := r.Run(ctx, []string{"-child", "-match", id, "-cache", cache})
	if res.Issue != filmproc.IssueOK {
		return nil, fmt.Errorf("enfant %s : issue=%s code=%d", id, res.Issue, res.Code)
	}
	d := parseDump(id, buf.String())
	// L'identite est resolue PAR MANCHE (le slot est reattribue d'une manche a l'autre), a partir
	// des series comp 2 B et du fil des morts deja captes.
	d.ident = roundIdentities(d)
	return d, nil
}

// parseDump reconstruit le filmDump a partir des lignes taguees de l'enfant.
func parseDump(id, out string) *filmDump {
	d := &filmDump{
		id: id, gamertags: map[uint64]string{}, deaths: map[uint64][]int{},
		series: map[int]map[int]map[emplKey][]point{},
	}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 1<<16), 1<<24)
	for sc.Scan() {
		// Les lignes taguees portent l'id du film en champ 1 ; le reste (slog) est ignore.
		parseLine(d, strings.Split(sc.Text(), "\t"))
	}
	return d
}

// parseLine range une ligne taguee dans le dump.
func parseLine(d *filmDump, f []string) {
	if len(f) < 2 || f[1] != d.id {
		return
	}
	switch f[0] {
	case "NREC":
		d.nrec = atoi(f[2])
		d.truncated = strings.Contains(f[3], "true")
	case "GT":
		d.gamertags[atou(f[2])] = f[3]
	case "DEATH":
		d.deaths[atou(f[2])] = splitInts(f[3])
	case "ROUNDS":
		d.rounds = splitInts(f[2])
	case "SERIES":
		parseSeries(d, f)
	}
}

// parseSeries range une ligne SERIES : slot, round, comp, side, points.
func parseSeries(d *filmDump, f []string) {
	if len(f) < 7 {
		return
	}
	slot, round, comp := atoi(f[2]), atoi(f[3]), atoi(f[4])
	key := emplKey{comp: comp, side: f[5]}
	if d.series[slot] == nil {
		d.series[slot] = map[int]map[emplKey][]point{}
	}
	if d.series[slot][round] == nil {
		d.series[slot][round] = map[emplKey][]point{}
	}
	d.series[slot][round][key] = parsePoints(f[6])
}

func parsePoints(s string) []point {
	var out []point
	for _, tok := range strings.Split(s, ";") {
		if tok == "" || tok[0] == '+' {
			continue
		}
		kv := strings.SplitN(tok, ":", 2)
		if len(kv) != 2 {
			continue
		}
		out = append(out, point{t: atoi(kv[0]), v: int64(atoi(kv[1]))})
	}
	return out
}

func splitInts(s string) []int {
	if s == "" {
		return nil
	}
	var out []int
	for _, tok := range strings.Split(s, ",") {
		if tok != "" {
			out = append(out, atoi(tok))
		}
	}
	return out
}

func atoi(s string) int    { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
func atou(s string) uint64 { n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64); return n }
