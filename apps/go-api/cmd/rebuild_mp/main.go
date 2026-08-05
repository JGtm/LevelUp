//go:build cgo

// rebuild_mp — reconstruit `match_participants` (shared) pour défaire la corruption
// d'index ART filter-pushdown (bug DuckDB #23046), en DÉLÉGUANT à la recette testée.
//
// ─── CE QUE CETTE RÉÉCRITURE CORRIGE (dette H4, ouverte par J4R-7) ────────────────
//
// La version précédente refaisait le swap à la main : elle capturait la DDL d'une
// LISTE ÉCRITE EN DUR de vues dépendantes, puis `DROP TABLE ... CASCADE` suivi d'un
// `CREATE VIEW` de recréation. Deux défauts, mesurés :
//
//  1. `DROP TABLE t CASCADE` de DuckDB réussit mais LAISSE la vue dépendante au
//     catalogue (sonde jouée le 2026-08-02). Le `CREATE VIEW` de recréation échouait
//     alors en « Catalog Error: View with name "v" already exists! » → rollback →
//     `os.Exit(1)`. L'outil de réparation ART était donc INUTILISABLE dès qu'une vue
//     dépendait de la table, c'est-à-dire toujours.
//  2. La liste de vues en dur divergeait du schéma à chaque bascule (elle a dû être
//     réécrite le 2026-08-02), et `//go:build ignore` faisait que RIEN — ni compilation,
//     ni vet, ni test — ne le disait.
//
// La réparation vit désormais dans `migration.RebuildMatchParticipantsART`, qui est le
// chemin de PRODUCTION (migration enregistrée + rebuild runtime) et qui est TESTÉ :
// swap CTAS transactionnel, garde anti-perte `rebuilt == before` AVANT le DROP,
// récupération d'un `__rebuilt` orphelin laissé par un crash, puis recréation des vues
// par `CREATE OR REPLACE` (qui, elle, écrase une vue restée au catalogue) et des index.
// Aucune liste de vues n'est recopiée ici : le schéma est sa propre source.
//
// Le diagnostic, lui, reste la valeur propre de cet outil — et il utilise la sonde
// existante `duckdb.ProbeARTDivergences` (index vs table-scan sur un échantillon de
// clés) au lieu du match codé en dur de l'incident de mai 2026.
//
// Usage :
//
//	go run -tags cgo ./cmd/rebuild_mp <shared_matches_v2.duckdb> [-match <match_id>] [-samples N]
//
// PRÉ-REQUIS : backup préalable, et AUCUN writer (serveur, backfill) ne doit tenir la
// DB — le modèle mono-process interdit RO+RW sur le même fichier (ADR 0013/0016).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

func main() {
	matchID := flag.String("match", "", "match_id à sonder finement avant/après (optionnel)")
	samples := flag.Int("samples", 20, "nombre de clés échantillonnées par la sonde ART")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: rebuild_mp <shared_matches_v2.duckdb> [-match <id>] [-samples N]")
		os.Exit(2)
	}

	if err := run(context.Background(), flag.Arg(0), *matchID, *samples); err != nil {
		fmt.Fprintln(os.Stderr, "ERREUR:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dbPath, matchID string, samples int) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", dbPath, err)
	}
	defer db.Close()

	before, err := snapshot(ctx, db, matchID, samples)
	if err != nil {
		return fmt.Errorf("état avant: %w", err)
	}
	fmt.Printf("AVANT  : %s\n", before)

	// Le rebuild lui-même : transactionnel, gardé anti-perte, idempotent.
	if err := migration.RebuildMatchParticipantsART(ctx, db); err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}

	after, err := snapshot(ctx, db, matchID, samples)
	if err != nil {
		return fmt.Errorf("état après: %w", err)
	}
	fmt.Printf("APRÈS  : %s\n", after)

	switch {
	case after.rows != before.rows:
		return fmt.Errorf("PERTE DE LIGNES : %d -> %d (la garde anti-perte aurait dû l'empêcher — "+
			"restaurer le backup et ouvrir un incident)", before.rows, after.rows)
	case after.divergences > 0:
		return fmt.Errorf("divergence ART TOUJOURS PRÉSENTE après rebuild (%d clé(s) sur %d) — "+
			"la corruption n'est pas celle que cet outil défait", after.divergences, samples)
	case before.divergences > 0:
		fmt.Printf("OK : divergence ART résolue (%d -> 0), aucune perte de ligne (%d).\n",
			before.divergences, after.rows)
	default:
		fmt.Printf("OK : aucune divergence détectée avant NI après, aucune perte de ligne (%d). "+
			"Le rebuild a tourné à blanc — c'est le résultat attendu sur une base saine.\n", after.rows)
	}
	return nil
}

// artState — l'état mesuré d'une table vis-à-vis du bug ART.
type artState struct {
	rows        int
	divergences int
	// matchIndexed / matchScan : la sonde fine d'un match_id nommé. Renseignées
	// seulement si -match est fourni. Leur ÉCART est la signature du bug : la
	// lecture par index rend moins de lignes que le table-scan sur la même clé.
	matchIndexed int
	matchScan    int
	matchProbed  bool
}

func (s artState) String() string {
	out := fmt.Sprintf("lignes=%d  clés_divergentes=%d", s.rows, s.divergences)
	if s.matchProbed {
		out += fmt.Sprintf("  match[index=%d scan=%d]", s.matchIndexed, s.matchScan)
	}
	return out
}

func snapshot(ctx context.Context, db *sql.DB, matchID string, samples int) (artState, error) {
	var st artState
	if err := db.QueryRowContext(ctx,
		// `|| ''` force le table-scan : c'est la lecture qui ne consulte PAS l'index ART,
		// donc la seule qui donne le nombre de lignes réellement présentes.
		`SELECT COUNT(*) FROM match_participants WHERE match_id || '' IS NOT NULL`).Scan(&st.rows); err != nil {
		return st, fmt.Errorf("compte des lignes: %w", err)
	}

	report, err := duckdb.ProbeARTDivergences(ctx, db, samples)
	if err != nil {
		return st, fmt.Errorf("sonde ART: %w", err)
	}
	for _, d := range report.Divergences {
		if d.Table == "match_participants" {
			st.divergences++
		}
	}

	if matchID == "" {
		return st, nil
	}
	st.matchProbed = true
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, matchID).Scan(&st.matchIndexed); err != nil {
		return st, fmt.Errorf("sonde match (index): %w", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_participants WHERE match_id || '' = ?`, matchID).Scan(&st.matchScan); err != nil {
		return st, fmt.Errorf("sonde match (scan): %w", err)
	}
	return st, nil
}
