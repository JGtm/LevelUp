package objectiveevents

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// score_measure_oracle_test.go — la lecture de l ORACLE de la phase 0 du lot A : le TSV
// exporte une fois de `match_registry` + `match_participants` en LECTURE SEULE.
//
// La base ne doit etre ouverte QUE pour cet export (plan, D13 : aucune ecriture, un seul
// essai en cas de verrou) ; l instrument travaille ensuite sur le TSV, jamais sur la base.
// Separe de score_measure_test.go pour tenir le seuil de 500 lignes par fichier.

// oracleMatch est la ligne de match de l'oracle et son roster.
type oracleMatch struct {
	MatchID, Variant, MapName string
	Team0, Team1, DurationS   int
	Lines                     []PlayerLine
	// Teams donne le team_id canonique par xuid.
	Teams map[string]int
}

// loadOracle lit le TSV de `match_registry` et celui des participants (meme dossier, suffixe
// `_participants`), et en extrait le match dont le match_id commence par short.
func loadOracle(t *testing.T, path, short string) oracleMatch {
	t.Helper()
	reg := readTSV(t, path)
	out := oracleMatch{Teams: map[string]int{}}
	for _, r := range reg.rows {
		id := reg.col(r, "match_id")
		if !strings.HasPrefix(id, short) {
			continue
		}
		out.MatchID = id
		out.Variant = reg.col(r, "game_variant_name")
		out.MapName = reg.col(r, "map_name")
		out.Team0, out.Team1 = atoi(reg.col(r, "team_0_score")), atoi(reg.col(r, "team_1_score"))
		out.DurationS = atoi(reg.col(r, "duration_seconds"))
	}
	if out.MatchID == "" {
		t.Fatalf("aucune ligne de %s ne commence par %q", path, short)
	}
	part := readTSV(t, strings.TrimSuffix(path, ".tsv")+"_participants.tsv")
	for _, r := range part.rows {
		if !strings.HasPrefix(part.col(r, "match_id"), short) {
			continue
		}
		xuid := part.col(r, "xuid")
		out.Lines = append(out.Lines, PlayerLine{XUID: xuid,
			Kills:   atoi(part.col(r, "kills")),
			Deaths:  atoi(part.col(r, "deaths")),
			Assists: atoi(part.col(r, "assists"))})
		out.Teams[xuid] = atoi(part.col(r, "team_id"))
	}
	if len(out.Lines) == 0 {
		t.Fatalf("aucun participant pour %q dans l'oracle", short)
	}
	return out
}

// tsvTable est un TSV lu en memoire, indexe par nom de colonne.
type tsvTable struct {
	head map[string]int
	rows [][]string
}

func readTSV(t *testing.T, path string) tsvTable {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("oracle illisible (%s) : %v", path, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("oracle vide (%s)", path)
	}
	out := tsvTable{head: map[string]int{}}
	for i, name := range strings.Split(lines[0], "\t") {
		out.head[strings.TrimSpace(name)] = i
	}
	for _, ln := range lines[1:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		out.rows = append(out.rows, strings.Split(ln, "\t"))
	}
	return out
}

func (tt tsvTable) col(row []string, name string) string {
	i, ok := tt.head[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// atoi rend 0 pour une cellule vide ou NULL — l'absence est ecrite telle quelle par les
// mesures, jamais devinee.
func atoi(s string) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return v
}
