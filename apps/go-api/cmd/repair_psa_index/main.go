// cmd/repair_psa_index — diagnostic et réparation one-shot des index ART de
// `personal_score_awards` sur les DB joueur (item B2.4 du plan
// .ai/PLAN_PERF_NOTE_OBJECTIFS.md).
//
// **Pourquoi** : découverte du lot 0, confirmée sur pièces le 2026-08-27 — sur la
// DB XxDaemonGamerxX, `WHERE match_id = '05fffb2a-...'` rend 2 lignes là où le
// scan complet en rend 4. L'index ART est désynchronisé de la table (bug DuckDB
// #23046). Tout lecteur qui interroge par prédicat indexé (PersonalScoreAwardsRepo,
// et le futur loader ospm du lot 3) sert donc des données AMPUTÉES, silencieusement.
//
// **Ce que l'outil fait** :
//  1. DIAGNOSTIC — pour chaque clé distincte de chaque axe indexé (match_id,
//     award_category, triplet match_id+xuid+generation_id), compare le comptage
//     par scan forcé au comptage par lookup indexé ;
//  2. RÉPARATION (option `-repair`) — DROP INDEX + CREATE INDEX à l'identique de
//     la DDL des migrations, sur les seuls index des axes en écart ;
//  3. RE-VÉRIFICATION — rejoue le diagnostic ; l'outil sort en erreur si un écart
//     subsiste.
//
// Aucune ligne de DONNÉES n'est modifiée : DDL d'index uniquement.
//
// **Mode par défaut : dry-run** (DB ouverte en `access_mode=read_only`). La DB
// n'est ouverte en écriture QUE si `-repair` est passé.
//
// **Pré-requis** : serveur arrêté (port 8000 libre) — DuckDB refuse l'ouverture
// RW d'une base déjà tenue par un autre process (ADR 0013/0016, un seul writer).
//
// Usage :
//
//	go run ./cmd/repair_psa_index -data <racine data>            # dry-run, 4 joueurs
//	go run ./cmd/repair_psa_index -data <racine data> -repair    # réparation
//	go run ./cmd/repair_psa_index -db <chemin stats.duckdb>      # une seule DB
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	titleSlug          = "halo_infinite"
	defaultDataRoot    = "data"
	maxListedDivergent = 20 // au-delà, la liste est tronquée (le total reste exact)
)

// defaultPlayers — les 4 joueurs suivis (mêmes gamertags que cmd/diag_perfsim).
var defaultPlayers = []string{"JGtm", "Chocoboflor", "Madina97294", "XxDaemonGamerxX"}

func main() {
	dataRoot := flagString("data", defaultDataRoot, "racine du dossier data/")
	dbPath := flagString("db", "", "chemin explicite d'une seule stats.duckdb (prioritaire sur -data)")
	players := flagString("players", strings.Join(defaultPlayers, ","), "gamertags séparés par des virgules")
	repair := flagBool("repair", "réparer les index en écart (sans ce drapeau : dry-run en lecture seule)")

	targets, err := resolveTargets(*dbPath, *dataRoot, *players)
	if err != nil {
		fatal("%v", err)
	}

	mode := "DRY-RUN (lecture seule)"
	if *repair {
		mode = "RÉPARATION (écriture DDL)"
	}
	fmt.Printf("== repair_psa_index — %s ==\n", mode)
	fmt.Printf("Cibles : %d DB joueur\n\n", len(targets))

	ctx := context.Background()
	failures := 0
	for _, t := range targets {
		if err := processTarget(ctx, t, *repair); err != nil {
			fmt.Printf("  ERREUR : %v\n\n", err)
			failures++
		}
	}

	if failures > 0 {
		fmt.Printf("== %d DB en erreur ==\n", failures)
		os.Exit(1)
	}
	fmt.Println("== Terminé ==")
}

// target — une DB joueur à traiter.
type target struct {
	label string
	path  string
}

func resolveTargets(dbPath, dataRoot, players string) ([]target, error) {
	if dbPath != "" {
		if _, err := os.Stat(dbPath); err != nil {
			return nil, fmt.Errorf("DB introuvable %s: %w", dbPath, err)
		}
		return []target{{label: filepath.Base(filepath.Dir(dbPath)), path: dbPath}}, nil
	}
	var out []target
	for _, gt := range strings.Split(players, ",") {
		gt = strings.TrimSpace(gt)
		if gt == "" {
			continue
		}
		p := filepath.Join(dataRoot, "titles", titleSlug, "players", gt, "stats.duckdb")
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("DB introuvable pour %s (%s): %w", gt, p, err)
		}
		out = append(out, target{label: gt, path: p})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune cible (données : -db=%q -data=%q -players=%q)", dbPath, dataRoot, players)
	}
	return out, nil
}

// processTarget applique les 3 phases (diagnostic, réparation, re-vérification)
// à une DB. Retourne une erreur si un écart subsiste après réparation, ou si la
// DB est en écart alors que le mode dry-run interdit d'y toucher (signalé, pas
// bloquant : c'est le résultat attendu d'un dry-run).
func processTarget(ctx context.Context, t target, repair bool) error {
	fmt.Printf("── %s ──\n%s\n", t.label, t.path)

	db, err := openDB(t.path, repair)
	if err != nil {
		return err
	}
	defer db.Close()

	var total int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personal_score_awards`).Scan(&total); err != nil {
		return fmt.Errorf("table personal_score_awards illisible: %w", err)
	}
	idx, err := existingIndexes(ctx, db)
	if err != nil {
		return err
	}
	fmt.Printf("lignes=%d  index=%v\n", total, idx)

	before, err := diagnoseAll(ctx, db)
	if err != nil {
		return err
	}
	printReports("AVANT", before)

	toRebuild := indexesToRebuild(before)
	if len(toRebuild) == 0 {
		fmt.Printf("VERDICT : aucun écart — rien à réparer.\n\n")
		return nil
	}
	if !repair {
		fmt.Printf("VERDICT : %d axe(s) en écart — réparation requise (relancer avec -repair).\n\n", countDivergentAxes(before))
		return nil
	}

	fmt.Printf("Réparation : DROP + CREATE %v ...\n", toRebuild)
	if err := repairIndexes(ctx, db, toRebuild); err != nil {
		return err
	}

	after, err := diagnoseAll(ctx, db)
	if err != nil {
		return err
	}
	printReports("APRÈS", after)

	var totalAfter int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM personal_score_awards`).Scan(&totalAfter); err != nil {
		return fmt.Errorf("recomptage après réparation: %w", err)
	}
	if totalAfter != total {
		return fmt.Errorf("ALERTE : lignes avant=%d après=%d — la réparation ne doit toucher AUCUNE donnée", total, totalAfter)
	}
	if n := countDivergentAxes(after); n > 0 {
		return fmt.Errorf("%d axe(s) TOUJOURS en écart après réparation — corruption au-delà de l'index, STOP", n)
	}
	fmt.Printf("VERDICT : réparé, 0 écart, %d lignes intactes.\n\n", totalAfter)
	return nil
}

// openDB ouvre la DB en lecture seule (dry-run) ou en écriture (réparation).
func openDB(path string, write bool) (*sql.DB, error) {
	dsn := path
	if !write {
		dsn += "?access_mode=read_only"
	}
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s (serveur encore actif ?): %w", path, err)
	}
	return db, nil
}

func countDivergentAxes(reports []axisReport) int {
	n := 0
	for _, r := range reports {
		if !r.ok() {
			n++
		}
	}
	return n
}

func printReports(phase string, reports []axisReport) {
	fmt.Printf("  [%s]\n", phase)
	for _, r := range reports {
		status := "OK"
		if !r.ok() {
			status = fmt.Sprintf("ÉCART (%d clés)", len(r.divergences))
		}
		fmt.Printf("   %-42s clés=%-6d scan=%-7d indexé=%-7d %s\n",
			r.axis, r.keys, r.scannedRows, r.indexedRows, status)
		if r.nullKeys > 0 {
			fmt.Printf("     (%d clé(s) NULL ignorée(s) — non interrogeables par égalité)\n", r.nullKeys)
		}
		for i, d := range r.divergences {
			if i >= maxListedDivergent {
				fmt.Printf("     ... et %d autre(s) clé(s) en écart\n", len(r.divergences)-maxListedDivergent)
				break
			}
			fmt.Printf("     %s : scan=%d indexé=%d\n", strings.Join(d.key, " | "), d.scanned, d.indexed)
		}
	}
}

// ── petits helpers de flags (dépendance zéro, style des cmd/diag_*) ─────────

func flagString(name, def, _ string) *string {
	v := def
	for i, arg := range os.Args[1:] {
		if arg == "-"+name || arg == "--"+name {
			if i+2 <= len(os.Args)-1 {
				v = os.Args[i+2]
			}
		}
	}
	return &v
}

func flagBool(name, _ string) *bool {
	v := false
	for _, arg := range os.Args[1:] {
		if arg == "-"+name || arg == "--"+name {
			v = true
		}
	}
	return &v
}

func fatal(format string, args ...any) {
	fmt.Printf("FATAL : "+format+"\n", args...)
	os.Exit(1)
}
