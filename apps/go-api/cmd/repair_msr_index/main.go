//go:build cgo

// cmd/repair_msr_index — diagnostic et réparation des index ART de
// `match_skill_rank` sur les DB joueur (item C.8 du plan
// .ai/PLAN_FINITIONS_2026-09-13.md, P0 du 2026-09-13).
//
// **Pourquoi** : mesuré sur pièces le 2026-09-13, serveur arrêté, sur la player DB
// de JGtm — `COUNT(*) FILTER (WHERE playlist_group='h5_arena')` (scan complet) rend
// 1 826 lignes, tandis que `WHERE playlist_group = 'h5_arena' GROUP BY rating_type`
// (lookup servi par idx_msr_playlist) n'en rend que 11 + 11. L'index ART est
// désynchronisé de la table : la donnée est INTACTE, mais tout lecteur qui
// interroge par prédicat indexé sert des lignes AMPUTÉES, silencieusement.
// C'est la signature exacte du bug DuckDB #23645, déjà constatée le 2026-08-27 sur
// personal_score_awards — la famille touche donc aussi match_skill_rank.
//
// **Ce que l'outil fait** (calqué sur cmd/repair_psa_index) :
//  1. DIAGNOSTIC — pour chaque axe indexé, compare le comptage par scan forcé au
//     comptage par lookup indexé, clé à clé ;
//  2. RÉPARATION (option `-repair`) — DROP INDEX + CREATE INDEX avec la DDL
//     CAPTURÉE dans la base, sur les seuls index des axes en écart ;
//  3. RE-VÉRIFICATION — rejoue le diagnostic ; sortie en erreur si un écart
//     subsiste, ou si le nombre de lignes a bougé.
//
// Aucune ligne de DONNÉES n'est modifiée : DDL d'index uniquement. Jamais de
// DELETE ni d'UPDATE — ce serait le vecteur ART lui-même.
//
// **Mode par défaut : dry-run** (DB ouverte en `access_mode=read_only`). La DB
// n'est ouverte en écriture QUE si `-repair` est passé.
//
// **Pré-requis** : serveur ARRÊTÉ — DuckDB refuse l'ouverture RW d'une base déjà
// tenue par un autre process (ADR 0013/0016, un seul writer). Sauvegarde préalable
// à la charge de l'opérateur.
//
// Usage (depuis apps/go-api) :
//
//	go run ./cmd/repair_msr_index -data ../../data            # dry-run, 4 joueurs
//	go run ./cmd/repair_msr_index -data ../../data -repair    # réparation
//	go run ./cmd/repair_msr_index -db <chemin stats.duckdb>   # une seule DB
package main

import (
	"context"
	"database/sql"
	"flag"
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

// defaultPlayers — les 4 joueurs suivis (mêmes gamertags que cmd/repair_psa_index).
var defaultPlayers = []string{"JGtm", "Chocoboflor", "Madina97294", "XxDaemonGamerxX"}

func main() {
	dataRoot := flag.String("data", defaultDataRoot, "racine du dossier data/")
	dbPath := flag.String("db", "", "chemin explicite d'une seule stats.duckdb (prioritaire sur -data)")
	players := flag.String("players", strings.Join(defaultPlayers, ","), "gamertags séparés par des virgules")
	repair := flag.Bool("repair", false, "réparer les index en écart (sans ce drapeau : dry-run en lecture seule)")
	flag.Parse()

	targets, err := resolveTargets(*dbPath, *dataRoot, *players)
	if err != nil {
		fmt.Printf("FATAL : %v\n", err)
		os.Exit(1)
	}

	mode := "DRY-RUN (lecture seule)"
	if *repair {
		mode = "RÉPARATION (écriture DDL)"
	}
	fmt.Printf("== repair_msr_index — %s ==\n", mode)
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
// à une DB. En dry-run, un écart est SIGNALÉ sans erreur : c'est le résultat
// attendu d'un diagnostic.
func processTarget(ctx context.Context, t target, repair bool) error {
	fmt.Printf("── %s ──\n%s\n", t.label, t.path)

	db, err := openDB(t.path, repair)
	if err != nil {
		return err
	}
	defer db.Close()

	total, err := countRows(ctx, db)
	if err != nil {
		return err
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
		fmt.Printf("VERDICT : %d axe(s) en écart — réparation requise (relancer avec -repair).\n\n",
			countDivergentAxes(before))
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

	totalAfter, err := countRows(ctx, db)
	if err != nil {
		return fmt.Errorf("recomptage après réparation: %w", err)
	}
	if totalAfter != total {
		return fmt.Errorf("ALERTE : lignes avant=%d après=%d — la réparation ne doit toucher AUCUNE donnée",
			total, totalAfter)
	}
	if n := countDivergentAxes(after); n > 0 {
		return fmt.Errorf("%d axe(s) TOUJOURS en écart après réparation — corruption au-delà de l'index, STOP", n)
	}
	fmt.Printf("VERDICT : réparé, 0 écart, %d lignes intactes.\n\n", totalAfter)
	return nil
}

// countRows compte les lignes de match_skill_rank (scan complet, jamais un index).
func countRows(ctx context.Context, db *sql.DB) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_skill_rank`).Scan(&n); err != nil {
		return 0, fmt.Errorf("table match_skill_rank illisible: %w", err)
	}
	return n, nil
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
		fmt.Printf("   %-54s clés=%-6d scan=%-7d indexé=%-7d %s\n",
			r.axis, r.keys, r.scannedRows, r.indexedRows, status)
		if r.nullKeys > 0 {
			fmt.Printf("     (%d clé(s) NULL ignorée(s) — non interrogeables par égalité)\n", r.nullKeys)
		}
		if r.truncated {
			fmt.Printf("     (échantillon tronqué à %d clés — les totaux ci-dessus ne portent que sur elles)\n",
				maxSampledKeys)
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
