// tmp_findmatch — retrouver l'identifiant COMPLET d'un match a partir de sa carte et de sa
// date, pour pouvoir demander un manifeste de film frais a l'API.
//
// POURQUOI CET OUTIL EXISTE. Les manifestes de film sont nommes par le PREFIXE de
// l'identifiant de match (8 caracteres hexadecimaux). Pour rafraichir un manifeste expire il
// faut appeler `discovery-infiniteugc.../hi/films/matches/{matchID}/spectate`, qui exige
// l'identifiant ENTIER. Le prefixe ne suffit pas et rien dans le cache ne porte l'identifiant
// complet.
//
// PRECAUTION DE VERROU. La base partagee peut etre tenue en ecriture par le serveur (modele
// mono-process, ADR 0013/0016) : ouvrir le fichier vivant en lecture depuis un autre process
// est interdit. On travaille donc sur une COPIE, ce qui supprime la question — le cout est
// une copie de fichier, la contrepartie est zero risque de corruption.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
)

func main() {
	src := flag.String("db", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\warehouse\shared_matches_v2.duckdb`, "base partagee")
	tmp := flag.String("copy", `C:\Users\GUILLA~1\AppData\Local\Temp\claude\findmatch_copy.duckdb`, "chemin de la copie de travail")
	day := flag.String("day", "2026-07-24", "journee a lister (AAAA-MM-JJ)")
	mapLike := flag.String("map", "", "filtre sur le nom de carte (sous-chaine, insensible a la casse)")
	prefix := flag.String("prefix", "", "chercher un match dont l'identifiant commence par ce prefixe")
	schema := flag.Bool("schema", false, "afficher les colonnes de match_registry et sortir")
	reuse := flag.Bool("reuse", false, "reutiliser la copie existante au lieu de recopier")
	flag.Parse()

	if !*reuse {
		if err := copyFile(*src, *tmp); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	db, err := sql.Open("duckdb", *tmp+"?access_mode=read_only")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ouverture :", err)
		os.Exit(1)
	}
	defer db.Close()

	if *schema {
		dumpSchema(db)
		return
	}
	cols, err := columnsOf(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *prefix != "" {
		queryByPrefix(db, cols, *prefix)
		return
	}
	queryByDay(db, cols, *day, *mapLike)
}

// columnsOf rend l'ensemble des colonnes de match_registry. Le schema a bouge au fil des
// versions ; on ne code en dur aucun nom qu'on n'a pas verifie present.
func columnsOf(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns
	                        WHERE table_name = 'match_registry'`)
	if err != nil {
		return nil, fmt.Errorf("lecture du schema : %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out[c] = true
	}
	return out, rows.Err()
}

func dumpSchema(db *sql.DB) {
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns
	                        WHERE table_name = 'match_registry' ORDER BY ordinal_position`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer rows.Close()
	fmt.Println("match_registry —")
	for rows.Next() {
		var c, t string
		if err := rows.Scan(&c, &t); err != nil {
			continue
		}
		fmt.Printf("  %-34s %s\n", c, t)
	}
}

// timeExpr rend le fragment temporel canonique (regle projet n.8 : jamais start_time brut).
func timeExpr(cols map[string]bool) string {
	if cols["start_time_utc"] && cols["start_time"] {
		return analysis.SQLStartTimeCanonical("")
	}
	if cols["start_time_utc"] {
		return "start_time_utc"
	}
	return "start_time"
}

// pick rend la premiere colonne presente parmi les candidates, ou "NULL".
func pick(cols map[string]bool, cands ...string) string {
	for _, c := range cands {
		if cols[c] {
			return c
		}
	}
	return "NULL"
}

func queryByDay(db *sql.DB, cols map[string]bool, day, mapLike string) {
	t := timeExpr(cols)
	mapCol := pick(cols, "map_name", "map", "map_id")
	modeCol := pick(cols, "game_variant_name", "mode_name", "playlist_name", "game_variant_id")
	q := fmt.Sprintf(`SELECT match_id, %s AS carte, %s AS mode, %s AS quand
	                   FROM match_registry
	                   WHERE CAST(%s AS DATE) = CAST(? AS DATE)`, mapCol, modeCol, t, t)
	args := []any{day}
	if mapLike != "" {
		q += fmt.Sprintf(` AND lower(CAST(%s AS VARCHAR)) LIKE ?`, mapCol)
		args = append(args, "%"+strings.ToLower(mapLike)+"%")
	}
	q += fmt.Sprintf(` ORDER BY %s`, t)
	run(db, q, args, fmt.Sprintf("matchs du %s", day))
}

func queryByPrefix(db *sql.DB, cols map[string]bool, prefix string) {
	t := timeExpr(cols)
	mapCol := pick(cols, "map_name", "map", "map_id")
	modeCol := pick(cols, "game_variant_name", "mode_name", "playlist_name", "game_variant_id")
	q := fmt.Sprintf(`SELECT match_id, %s AS carte, %s AS mode, %s AS quand
	                   FROM match_registry WHERE match_id LIKE ? ORDER BY %s`,
		mapCol, modeCol, t, t)
	run(db, q, []any{prefix + "%"}, "match par prefixe "+prefix)
}

func run(db *sql.DB, q string, args []any, title string) {
	rows, err := db.Query(q, args...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "requete :", err)
		os.Exit(1)
	}
	defer rows.Close()
	fmt.Printf("%s —\n\n", title)
	n := 0
	for rows.Next() {
		var id string
		var carte, mode, quand sql.NullString
		if err := rows.Scan(&id, &carte, &mode, &quand); err != nil {
			fmt.Fprintln(os.Stderr, "scan :", err)
			continue
		}
		n++
		fmt.Printf("  %s  %-26s  %-30s  %s\n",
			id, carte.String, mode.String, quand.String)
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Printf("\n  %d match(s)\n", n)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("ouverture source %s : %w", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creation copie %s : %w", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copie : %w", err)
	}
	return out.Sync()
}
