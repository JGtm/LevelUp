// Package indexcheck — détection d'un index ART DÉSYNCHRONISÉ de sa table.
//
// LA RÈGLE, EN UNE PHRASE. Pour un axe indexé, deux comptages qui DOIVENT être
// égaux sont comparés clé par clé :
//   - la RÉFÉRENCE, par SCAN FORCÉ : la clé de regroupement est une EXPRESSION
//     (`col || ”`, `CAST(col AS VARCHAR)`) qu'aucun index ART ne peut servir ;
//   - la MESURE, par LOOKUP INDEXÉ (`WHERE col = ?`), colonnes NUES, que le
//     planner sert par l'index.
//
// Tout écart = index désynchronisé de la table (bug DuckDB #23645, TOUJOURS
// ouvert en 1.5.5). Les clés NULL sont EXCLUES du verdict et comptées à part :
// `col = NULL` ne matche jamais et produirait un faux écart.
//
// POURQUOI UN PAQUET. La règle vivait en double : dans la sonde data-health de
// `personal_score_awards` et dans le diagnostic de `cmd/repair_psa_index`. La
// sonde data-health de `match_skill_rank` (G.3, 2026-09-13) en aurait fait une
// troisième et une quatrième copie — seuil de la règle CLAUDE.md n°6. Le paquet
// est la source unique pour `match_skill_rank` ; le garde-rail qui interdit d'en
// recopier la carte d'axes est `internal/archlint/no_local_msr_axes_test.go`.
// (Les deux consommateurs côté personal_score_awards ont été SUPPRIMÉS le
// 2026-09-20 avec les index qu'ils surveillaient : ce paquet ne sert plus que
// `match_skill_rank` — sonde data-health + `cmd/repair_msr_index`.)
//
// CE PAQUET NE RÉPARE RIEN. Il lit, il compare, il rend un rapport. La DDL de
// réparation est capturée dans la base par l'appelant (`cmd/repair_msr_index`),
// jamais recopiée.
package indexcheck

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// Axis décrit UN axe indexé à vérifier.
type Axis struct {
	// Name : libellé lisible de l'axe (rapport, logs).
	Name string
	// KeyExprs : expressions de regroupement, SCAN FORCÉ (jamais servi par un ART).
	KeyExprs []string
	// LookupWhere : prédicat de lookup, colonnes NUES (l'index peut servir).
	LookupWhere string
	// Indexes : index susceptibles de servir ce lookup → à reconstruire si écart.
	Indexes []string
}

// Options paramètre UNE passe de vérification.
type Options struct {
	// Table : la table vérifiée. Interpolée dans le SQL : elle vient TOUJOURS
	// d'une carte d'axes littérale du dépôt, jamais d'une entrée externe.
	Table string
	// MaxKeys borne le nombre de clés comparées. 0 = pas de borne.
	MaxKeys int
	// Sample : tirer MaxKeys clés au hasard (réservoir) au lieu de prendre les
	// MaxKeys premières. Le tirage change à chaque passe, donc la couverture
	// s'accumule sur les cycles successifs — c'est le mode des sondes
	// périodiques, dont le coût doit rester borné. Sans effet si MaxKeys == 0.
	Sample bool
}

// KeyCount — une clé distincte et son compte de RÉFÉRENCE (par scan forcé).
type KeyCount struct {
	Key   []string
	Count int
}

// Divergence — une clé dont le lookup indexé ne rend pas le compte du scan.
type Divergence struct {
	Key     []string
	Scanned int
	Indexed int
}

// Report — résultat de la vérification d'UN axe.
type Report struct {
	Axis        string
	Table       string
	Keys        int  // clés effectivement comparées (hors NULL, après bornage)
	NullKeys    int  // clés écartées parce que NULL (non interrogeables par égalité)
	Truncated   bool // le nombre de clés distinctes dépassait MaxKeys
	ScannedRows int
	IndexedRows int
	Divergences []Divergence
}

// OK — aucun écart constaté sur cet axe.
func (r Report) OK() bool { return len(r.Divergences) == 0 }

// RowsMissing — total des lignes INVISIBLES au lookup indexé (scan - indexé).
// Seul un DÉFICIT compte : un excédent est une autre pathologie (doublons
// d'index) et ne doit pas être masqué par une soustraction négative.
func (r Report) RowsMissing() int {
	total := 0
	for _, d := range r.Divergences {
		if d.Scanned > d.Indexed {
			total += d.Scanned - d.Indexed
		}
	}
	return total
}

// Compare est LA RÈGLE, isolée de tout SQL : pour chaque clé de référence, le
// comptage indexé doit égaler le comptage du scan. C'est le seul endroit où un
// index réellement désynchronisé peut être simulé en test — on ne sait pas
// corrompre un ART DuckDB à la demande.
func Compare(refs []KeyCount, lookup func(key []string) (int, error)) (indexedRows int, divergences []Divergence, err error) {
	for _, ref := range refs {
		indexed, lookupErr := lookup(ref.Key)
		if lookupErr != nil {
			return indexedRows, divergences, lookupErr
		}
		indexedRows += indexed
		if indexed != ref.Count {
			divergences = append(divergences, Divergence{
				Key: ref.Key, Scanned: ref.Count, Indexed: indexed,
			})
		}
	}
	return indexedRows, divergences, nil
}

// TableExists — présence de la table dans la base ouverte.
func TableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, table).Scan(&n); err != nil {
		return false, fmt.Errorf("présence de la table %s: %w", table, err)
	}
	return n > 0, nil
}

// referenceSQL bâtit la requête de RÉFÉRENCE (scan forcé). En mode échantillon,
// le regroupement est enveloppé dans un `USING SAMPLE … (reservoir)` : le tirage
// porte sur les CLÉS DISTINCTES, pas sur les lignes — une clé tirée est donc
// toujours comparée sur la TOTALITÉ de ses lignes.
func referenceSQL(a Axis, opts Options) string {
	exprs := strings.Join(a.KeyExprs, ", ")
	group := fmt.Sprintf(`SELECT %s, COUNT(*) AS n FROM %s GROUP BY %s`, exprs, opts.Table, exprs)
	if opts.Sample && opts.MaxKeys > 0 {
		return fmt.Sprintf("SELECT * FROM (%s) USING SAMPLE %d ROWS (reservoir)", group, opts.MaxKeys)
	}
	return group + " ORDER BY " + exprs
}

// scanReference établit, par SCAN FORCÉ, le compte de référence par clé.
func scanReference(ctx context.Context, db *sql.DB, a Axis, opts Options) ([]KeyCount, *Report, error) {
	meta := &Report{Axis: a.Name, Table: opts.Table}
	rows, err := db.QueryContext(ctx, referenceSQL(a, opts))
	if err != nil {
		return nil, meta, fmt.Errorf("scan de référence (%s): %w", a.Name, err)
	}
	defer rows.Close() //nolint:errcheck // lecture

	var refs []KeyCount
	for rows.Next() {
		key, n, scanErr := scanOneKey(rows, len(a.KeyExprs))
		if scanErr != nil {
			return nil, meta, fmt.Errorf("scan de référence (%s): %w", a.Name, scanErr)
		}
		if key == nil {
			meta.NullKeys++
			continue
		}
		if opts.MaxKeys > 0 && len(refs) >= opts.MaxKeys {
			meta.Truncated = true
			continue
		}
		refs = append(refs, KeyCount{Key: key, Count: n})
		meta.ScannedRows += n
	}
	if err := rows.Err(); err != nil {
		return nil, meta, fmt.Errorf("itération du scan (%s): %w", a.Name, err)
	}
	return refs, meta, nil
}

// scanOneKey lit une ligne du scan de référence. Rend `nil` comme clé quand une
// composante est NULL (clé non interrogeable par égalité).
func scanOneKey(rows *sql.Rows, arity int) ([]string, int, error) {
	vals := make([]sql.NullString, arity)
	dest := make([]any, 0, arity+1)
	for i := range vals {
		dest = append(dest, &vals[i])
	}
	var n int
	dest = append(dest, &n)
	if err := rows.Scan(dest...); err != nil {
		return nil, 0, err
	}
	key := make([]string, 0, arity)
	for _, v := range vals {
		if !v.Valid {
			return nil, n, nil
		}
		key = append(key, v.String)
	}
	return key, n, nil
}

// Run vérifie UN axe : scan de référence, puis lookup indexé clé par clé.
func Run(ctx context.Context, db *sql.DB, a Axis, opts Options) (Report, error) {
	refs, rep, err := scanReference(ctx, db, a, opts)
	if err != nil {
		return *rep, err
	}
	rep.Keys = len(refs)
	if len(refs) == 0 {
		return *rep, nil
	}

	stmt, err := db.PrepareContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, opts.Table, a.LookupWhere))
	if err != nil {
		return *rep, fmt.Errorf("préparation du lookup (%s): %w", a.Name, err)
	}
	defer stmt.Close() //nolint:errcheck // statement de lecture

	indexedRows, divergences, err := Compare(refs, func(key []string) (int, error) {
		args := make([]any, 0, len(key))
		for _, k := range key {
			args = append(args, k)
		}
		var indexed int
		if scanErr := stmt.QueryRowContext(ctx, args...).Scan(&indexed); scanErr != nil {
			return 0, fmt.Errorf("lookup indexé (%s, clé %v): %w", a.Name, key, scanErr)
		}
		return indexed, nil
	})
	rep.IndexedRows = indexedRows
	rep.Divergences = divergences
	return *rep, err
}

// RunAll passe tous les axes et rend les rapports DANS L'ORDRE de la carte.
func RunAll(ctx context.Context, db *sql.DB, axes []Axis, opts Options) ([]Report, error) {
	reports := make([]Report, 0, len(axes))
	for _, a := range axes {
		rep, err := Run(ctx, db, a, opts)
		if err != nil {
			return reports, err
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

// IndexesToRebuild — union ORDONNÉE des index des axes en écart.
func IndexesToRebuild(reports []Report, axes []Axis) []string {
	seen := map[string]bool{}
	for i, rep := range reports {
		if rep.OK() || i >= len(axes) {
			continue
		}
		for _, name := range axes[i].Indexes {
			seen[name] = true
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
