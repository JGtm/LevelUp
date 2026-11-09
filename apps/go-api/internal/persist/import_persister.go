package persist

// import_persister.go — l'écriture d'un import de tables Parquet étrangères (lot L22.b : données d'un joueur
// exportées par une instance hébergée), sous les règles de ce paquet : INSERT seulement, une transaction,
// aucun UPSERT ni UPDATE ni DELETE (ADR 0019, 0026).
//
// POURQUOI UN PERSISTER À PART : un lot typé (SharedBatch) décrit un match par ses lignes, table par
// table, avec des types Go ; un import vient d'un autre déploiement, dont le schéma peut avoir un peu
// divergé. Le persister lit donc les colonnes du Parquet et celles de la table locale et n'écrit que
// leur intersection ; une colonne locale obligatoire absente du fichier (NOT NULL sans défaut) refuse la
// table entière plutôt que d'inventer une valeur.
//
// IDEMPOTENCE : une ligne dont la clé (ImportTable.Key) existe déjà dans la table locale n'est pas
// réécrite. Rejouer le même import n'insère rien ; un match déjà synchronisé n'est jamais dupliqué.
//
// IDENTIFIANT LOCAL : une colonne `id` que la table locale remplit elle-même (défaut, séquence) n'est PAS
// reprise du fichier : la séquence de l'instance d'origine n'est pas celle d'ici, et les vues `_latest`
// départagent deux passes par (written_at, id) ; l'ordre des lignes du fichier est conservé (ORDER BY id).
//
// ÉTENDUE : ImportTable.Where et ImportTable.OwnerCol restreignent les lignes lues du fichier, pour qu'une
// archive ne puisse écrire que ce que l'appelant a décidé d'accepter (les matchs du joueur, ses lignes).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// colID : la colonne d'identifiant que la table locale remplit elle-même.
const colID = "id"

// ImportTable décrit une table à importer : son nom local, le fichier Parquet source et les colonnes de la
// clé d'unicité (une ligne dont la clé existe déjà localement est ignorée).
type ImportTable struct {
	Name string
	File string
	Key  []string
	// Where : condition SQL sur l'alias `s` (les lignes du fichier) construite par l'APPELANT, jamais depuis
	// une entrée libre ; une ligne qui ne la vérifie pas n'entre pas. Vide : aucune restriction.
	Where string
	// OwnerCol et OwnerValue : si le fichier porte la colonne OwnerCol, seules les lignes dont elle vaut
	// OwnerValue entrent (une table propre à un joueur ne reçoit jamais la ligne d'un autre).
	OwnerCol, OwnerValue string
}

// ImportPersister écrit des tables Parquet dans une base ouverte en écriture. Le bail d'écriture de la base
// est tenu par l'appelant pendant tout l'appel.
type ImportPersister struct {
	db *sql.DB
}

// NewImportPersister crée un persister sur un handle d'écriture déjà acquis.
func NewImportPersister(db *sql.DB) *ImportPersister { return &ImportPersister{db: db} }

// Persist insère les lignes nouvelles de chaque table, dans UNE transaction : ou tout est écrit, ou rien.
// Rend, par table, le nombre de lignes insérées. Une table absente de la base locale ou sans colonne
// commune est une erreur (le schéma local est celui de l'application, il doit connaître la table).
func (p *ImportPersister) Persist(ctx context.Context, tables []ImportTable) (map[string]int64, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("import: ouverture de la transaction: %w", err)
	}
	inserted := make(map[string]int64, len(tables))
	for _, t := range tables {
		n, err := insertNew(ctx, tx, t)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("import de la table %s: %w", t.Name, err)
		}
		inserted[t.Name] = n
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("import: validation de la transaction: %w", err)
	}
	return inserted, nil
}

// insertNew insère les lignes de t dont la clé n'existe pas encore.
func insertNew(ctx context.Context, tx *sql.Tx, t ImportTable) (int64, error) {
	lit := SQLLiteral(t.File)
	plan, err := planColumns(ctx, tx, t.Name, lit)
	if err != nil {
		return 0, err
	}
	for _, k := range t.Key {
		if !contains(plan.insert, k) {
			return 0, fmt.Errorf("colonne de clé %q absente du fichier ou de la table", k)
		}
	}
	var on []string
	for _, k := range t.Key {
		on = append(on, fmt.Sprintf("x.%s = s.%s", quoteIdent(k), quoteIdent(k)))
	}
	q := fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM read_parquet('%s') s WHERE NOT EXISTS (SELECT 1 FROM %s x WHERE %s)`,
		quoteIdent(t.Name), quoteIdents(plan.insert), prefixed("s", plan.insert), lit, quoteIdent(t.Name), strings.Join(on, " AND "))
	if t.Where != "" {
		q += " AND (" + t.Where + ")"
	}
	if t.OwnerCol != "" && plan.file[t.OwnerCol] {
		q += fmt.Sprintf(" AND CAST(s.%s AS VARCHAR) = '%s'", quoteIdent(t.OwnerCol), SQLLiteral(t.OwnerValue))
	}
	if plan.orderByID {
		q += " ORDER BY s." + quoteIdent(colID)
	}
	res, err := tx.ExecContext(ctx, q)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// columnPlan : ce qu'on écrit d'une table. insert : les colonnes communes au fichier et à la table (ordre de la
// table) ; file : toutes les colonnes du fichier ; orderByID : l'ordre du fichier se lit dans sa colonne id.
type columnPlan struct {
	insert    []string
	file      map[string]bool
	orderByID bool
}

// planColumns rend les colonnes à écrire, et refuse la table dont une colonne obligatoire sans défaut manque
// au fichier.
func planColumns(ctx context.Context, tx *sql.Tx, table, lit string) (columnPlan, error) {
	plan := columnPlan{}
	file, err := fileColumns(ctx, tx, lit)
	if err != nil {
		return plan, err
	}
	plan.file = file
	local, err := tx.QueryContext(ctx, `SELECT column_name, is_nullable, column_default FROM information_schema.columns
		WHERE table_schema = 'main' AND table_name = ? ORDER BY ordinal_position`, table)
	if err != nil {
		return plan, err
	}
	defer func() { _ = local.Close() }()
	found := false
	for local.Next() {
		var name, nullable string
		var def sql.NullString
		if err := local.Scan(&name, &nullable, &def); err != nil {
			return plan, err
		}
		found = true
		switch {
		case name == colID && def.Valid:
			plan.orderByID = file[colID]
		case file[name]:
			plan.insert = append(plan.insert, name)
		case nullable == "NO" && !def.Valid:
			return plan, fmt.Errorf("colonne obligatoire %q absente du fichier", name)
		}
	}
	if err := local.Err(); err != nil {
		return plan, err
	}
	if !found {
		return plan, fmt.Errorf("table inconnue de la base locale")
	}
	if len(plan.insert) == 0 {
		return plan, fmt.Errorf("aucune colonne commune")
	}
	return plan, nil
}

// fileColumns rend l'ensemble des colonnes du fichier Parquet.
func fileColumns(ctx context.Context, tx *sql.Tx, lit string) (map[string]bool, error) {
	cols := map[string]bool{}
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT column_name FROM (DESCRIBE SELECT * FROM read_parquet('%s'))`, lit))
	if err != nil {
		return nil, fmt.Errorf("lecture du fichier: %w", err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			_ = rows.Close()
			return nil, err
		}
		cols[c] = true
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return cols, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

func quoteIdents(cols []string) string {
	q := make([]string, len(cols))
	for i, c := range cols {
		q[i] = quoteIdent(c)
	}
	return strings.Join(q, ", ")
}

func prefixed(alias string, cols []string) string {
	q := make([]string, len(cols))
	for i, c := range cols {
		q[i] = alias + "." + quoteIdent(c)
	}
	return strings.Join(q, ", ")
}

// SQLLiteral protège une valeur placée dans un littéral SQL entre apostrophes (apostrophe doublée) : chemin
// d'un fichier, xuid.
func SQLLiteral(s string) string { return strings.ReplaceAll(s, "'", "''") }
