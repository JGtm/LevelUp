package migration

// compaction_rewrite.go — `--rewrite-file` de la compaction (DC.5) : rendre au disque la place
// que la compaction a libérée.
//
// DuckDB réutilise les blocs libérés par le DROP des anciennes tables, mais ne RÉTRÉCIT jamais le
// fichier. La seule façon de le réduire est d'en écrire un neuf : `COPY FROM DATABASE` recopie le
// catalogue (tables, vues, index, séquences, macros, types) puis les données.
//
// Ce paquet fournit les deux briques pures ; l'ouverture des fichiers et l'échange sur disque
// appartiennent à la commande (`cmd/levelup/cmd_compact_passes_rewrite.go`) :
//   - CopierBaseVers : la copie, dans un fichier neuf ;
//   - LireInventaire / Ecarts : ce qui doit être identique entre l'ancien et le neuf, lu sur
//     CHAQUE fichier ouvert SEUL (une vue lue au travers d'un ATTACH pourrait se lier aux tables de
//     l'autre catalogue et « prouver » une égalité fausse).
//
// Mesuré sur DuckDB 1.5 (2026-09-26) : une séquence recopiée continue là où l'ancienne en était
// (`nextval` rend la même valeur des deux côtés) ; son `sql` porte `START <prochaine valeur>` et
// sert de preuve. Sa colonne `last_value` diffère (elle affiche la prochaine valeur, pas la
// dernière tirée) : aucun code ne la lit, elle n'entre pas dans l'inventaire.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

// Inventaire : le catalogue d'une base et le contenu qui doit survivre à sa réécriture.
type Inventaire struct {
	// Objets : une ligne par objet du catalogue — genre, nom, DDL tel que DuckDB le tient.
	Objets []string
	// Comptes : lignes de chaque table.
	Comptes map[string]int64
	// Vues : empreinte des vues `_latest` du registre de compaction présentes dans la base.
	Vues map[string]EmpreinteVue
}

// requetesCatalogue : chaque famille d'objets, sa clé d'ordre et son DDL.
var requetesCatalogue = []string{
	`SELECT 'table ' || table_name || ' ' || sql FROM duckdb_tables()
		WHERE database_name = current_database() ORDER BY table_name`,
	`SELECT 'vue ' || view_name || ' ' || sql FROM duckdb_views()
		WHERE database_name = current_database() AND NOT internal ORDER BY view_name`,
	`SELECT 'index ' || index_name || ' ' || sql FROM duckdb_indexes()
		WHERE database_name = current_database() ORDER BY index_name`,
	`SELECT 'sequence ' || sequence_name || ' ' || sql FROM duckdb_sequences()
		WHERE database_name = current_database() ORDER BY sequence_name`,
	`SELECT 'macro ' || function_name || ' ' || COALESCE(macro_definition, '') FROM duckdb_functions()
		WHERE database_name = current_database() AND NOT internal
		  AND function_type IN ('macro', 'table_macro') ORDER BY function_name`,
	`SELECT 'type ' || type_name || ' ' || logical_type FROM duckdb_types()
		WHERE database_name = current_database() AND NOT internal ORDER BY type_name`,
}

// LireInventaire lit l'inventaire de la base ouverte.
func LireInventaire(ctx context.Context, db *sql.DB) (Inventaire, error) {
	inv := Inventaire{Comptes: map[string]int64{}, Vues: map[string]EmpreinteVue{}}
	for _, q := range requetesCatalogue {
		lignes, err := lireColonne(ctx, db, q)
		if err != nil {
			return inv, fmt.Errorf("inventaire: catalogue: %w", err)
		}
		inv.Objets = append(inv.Objets, lignes...)
	}
	tables, err := lireColonne(ctx, db, `SELECT table_name FROM duckdb_tables()
		WHERE database_name = current_database() ORDER BY table_name`)
	if err != nil {
		return inv, fmt.Errorf("inventaire: tables: %w", err)
	}
	for _, t := range tables {
		var n int64
		if err := compter(ctx, db, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, t), &n); err != nil {
			return inv, fmt.Errorf("inventaire: compte de %s: %w", t, err)
		}
		inv.Comptes[t] = n
	}
	for _, c := range tablesCompactables {
		_, present, err := lireTableEtVue(ctx, db, c)
		if err != nil {
			return inv, err
		}
		if !present {
			continue
		}
		e, err := empreinteDeVue(ctx, db, c.View)
		if err != nil {
			return inv, err
		}
		inv.Vues[c.View] = e
	}
	return inv, nil
}

func lireColonne(ctx context.Context, db *sql.DB, q string) ([]string, error) {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // lecture seule, l'erreur d'itération est lue par rows.Err
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Ecarts rend ce qui diffère entre deux inventaires ; vide = identiques.
func (a Inventaire) Ecarts(b Inventaire) []string {
	var out []string
	if strings.Join(a.Objets, "\n") != strings.Join(b.Objets, "\n") {
		out = append(out, fmt.Sprintf("catalogue différent (%d objets contre %d) : %s",
			len(a.Objets), len(b.Objets), premierEcart(a.Objets, b.Objets)))
	}
	for _, t := range clesTriees(a.Comptes, b.Comptes) {
		if a.Comptes[t] != b.Comptes[t] {
			out = append(out, fmt.Sprintf("table %s : %d lignes contre %d", t, a.Comptes[t], b.Comptes[t]))
		}
	}
	for _, v := range clesTriees(a.Vues, b.Vues) {
		if a.Vues[v] != b.Vues[v] {
			out = append(out, fmt.Sprintf("vue %s : empreinte %+v contre %+v", v, a.Vues[v], b.Vues[v]))
		}
	}
	return out
}

func premierEcart(a, b []string) string {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return fmt.Sprintf("%q contre %q", a[i], b[i])
		}
	}
	return "listes de longueurs différentes"
}

func clesTriees[V any](a, b map[string]V) []string {
	vu := map[string]bool{}
	for k := range a {
		vu[k] = true
	}
	for k := range b {
		vu[k] = true
	}
	out := make([]string, 0, len(vu))
	for k := range vu {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// aliasReecriture : le nom sous lequel le fichier neuf est attaché le temps de la copie.
const aliasReecriture = "levelup_reecriture"

// CopierBaseVers recopie la base ouverte (catalogue puis données) dans le fichier `cible`, qui
// ne doit pas exister, puis le détache. Aucune écriture dans la base source.
func CopierBaseVers(ctx context.Context, db *sql.DB, cible string) error {
	if strings.ContainsAny(cible, "'\x00") {
		return fmt.Errorf("réécriture: chemin cible refusé (%q)", cible)
	}
	var source string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&source); err != nil {
		return fmt.Errorf("réécriture: base courante: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`ATTACH '%s' AS %s`, cible, aliasReecriture)); err != nil {
		return fmt.Errorf("réécriture: attache de %s: %w", cible, err)
	}
	_, errCopie := db.ExecContext(ctx, fmt.Sprintf(`COPY FROM DATABASE "%s" TO %s`, source, aliasReecriture))
	if _, err := db.ExecContext(ctx, `DETACH `+aliasReecriture); err != nil {
		if errCopie == nil {
			return fmt.Errorf("réécriture: détache de %s: %w", cible, err)
		}
		slog.ErrorContext(ctx, "réécriture: détache après une copie en échec", "cible", cible, "err", err)
	}
	if errCopie != nil {
		return fmt.Errorf("réécriture: copie vers %s: %w", cible, errCopie)
	}
	return nil
}
