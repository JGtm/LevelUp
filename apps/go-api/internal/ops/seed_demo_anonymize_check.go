// Package ops — seed_demo_anonymize_check.go : CONTRÔLE DES VALEURS de la démo générée.
//
// La liste `identity` (seed_demo.go) dit quelles colonnes l'anonymisation remappe ; elle ne
// peut pas dire ce qu'elle oublie. Ce contrôle, joué à la fin de chaque seed, regarde les
// VALEURS : toute colonne texte (et toute colonne nommée `%xuid%`) des bases générées du titre
// est relue, et la présence d'une identité RÉELLE fait échouer le seed.
//
// IDENTITÉS RÉELLES : les xuid du roster (tous les participants du corpus) et des profils
// suivis, au format d'un xuid Xbox (chiffres, 15 au moins, jamais le préfixe « 0000 » des
// identités démo) ; les gamertags que la source associe à ces xuid et ceux des profils.
// Un xuid est cherché en SOUS-CHAÎNE (listes, JSON) ; un gamertag en valeur EXACTE ou en
// chaîne JSON (`"<gamertag>"`), pour ne pas confondre un nom court avec un mot d'un libellé.
//
// HORS CONTRÔLE : metadata.duckdb (référentiels du titre copiés tels quels, sans colonne
// d'identité) et les rejeux figés (artefacts intacts par décision D-1, masqués au service).
package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// demoIdentityLeak : une valeur réelle trouvée dans une base démo générée.
type demoIdentityLeak struct {
	DB, Table, Column, Value string
}

// where situe la fuite sans répéter la valeur réelle (le message finit dans les journaux).
func (l demoIdentityLeak) where() string {
	return fmt.Sprintf("%s:%s.%s", l.DB, l.Table, l.Column)
}

// realIdentities : les identités réelles à ne jamais retrouver dans la démo.
type realIdentities struct {
	xuids     []string
	gamertags map[string]bool
}

// isRealXUID : la forme d'un xuid Xbox réel (une identité démo commence par « 0000 »).
func isRealXUID(s string) bool {
	if len(s) < 15 || strings.HasPrefix(s, "0000") {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// add enregistre une identité réelle (xuid au bon format, gamertag non vide).
func (r *realIdentities) add(xuid, gamertag string) {
	if isRealXUID(xuid) {
		r.xuids = append(r.xuids, xuid)
	}
	if gamertag = strings.TrimSpace(gamertag); gamertag != "" {
		r.gamertags[gamertag] = true
	}
}

// leakIn rend la première identité réelle que porte `value`, "" sinon.
func (r realIdentities) leakIn(value string) string {
	if r.gamertags[value] {
		return value
	}
	for _, x := range r.xuids {
		if strings.Contains(value, x) {
			return x
		}
	}
	if strings.Contains(value, `"`) {
		for gt := range r.gamertags {
			if strings.Contains(value, `"`+gt+`"`) {
				return gt
			}
		}
	}
	return ""
}

// loadRealIdentities rassemble les identités réelles du seed : roster, gamertags que la
// source associe à ses xuid, profils suivis.
func loadRealIdentities(ctx context.Context, opts SeedDemoOptions, roster []demoRosterEntry) (realIdentities, error) {
	real := realIdentities{gamertags: map[string]bool{}}
	xuids := make([]any, 0, len(roster))
	for _, e := range roster {
		real.add(e.SourceXUID, "")
		xuids = append(xuids, e.SourceXUID)
	}
	if err := addProfileIdentities(opts.ProfilesPath, &real); err != nil {
		return real, err
	}
	if len(xuids) == 0 {
		return real, nil
	}
	shared, err := duckdb.OpenReadOnly(opts.SourceSharedDB)
	if err != nil {
		return real, fmt.Errorf("base partagée source: %w", err)
	}
	defer func() { _ = shared.Close() }()
	in := strings.TrimSuffix(strings.Repeat("?,", len(xuids)), ",")
	for _, table := range []string{tableXUIDAliases, tableMatchParticipants} {
		rows, err := shared.SQLDb().QueryContext(ctx,
			`SELECT DISTINCT gamertag FROM `+table+` WHERE gamertag IS NOT NULL AND xuid IN (`+in+`)`, xuids...)
		if err != nil {
			if errIsMissingTable(err) {
				continue
			}
			return real, fmt.Errorf("gamertags réels (%s): %w", table, err)
		}
		for rows.Next() {
			var gt string
			if err := rows.Scan(&gt); err != nil {
				_ = rows.Close()
				return real, fmt.Errorf("gamertags réels (%s, scan): %w", table, err)
			}
			real.add("", gt)
		}
		if err := rows.Close(); err != nil {
			return real, err
		}
	}
	return real, nil
}

// addProfileIdentities ajoute les joueurs suivis de db_profiles.json (v3 par titre, ou v2.1).
func addProfileIdentities(profilesPath string, real *realIdentities) error {
	if profilesPath == "" {
		return nil
	}
	raw, err := os.ReadFile(profilesPath)
	if err != nil {
		return fmt.Errorf("profils suivis: %w", err)
	}
	var v3 struct {
		Profiles map[string]map[string]profileEntry `json:"profiles"`
	}
	if json.Unmarshal(raw, &v3) == nil {
		for _, byGT := range v3.Profiles {
			for gt, p := range byGT {
				real.add(p.XUID, gt)
			}
		}
		return nil
	}
	var v2 struct {
		Profiles map[string]profileEntry `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &v2); err != nil {
		return fmt.Errorf("profils suivis illisibles: %w", err)
	}
	for gt, p := range v2.Profiles {
		real.add(p.XUID, gt)
	}
	return nil
}

// scanDemoIdentityLeaks relit les colonnes texte (et `%xuid%`) des tables d'une base démo et
// rend les valeurs qui portent une identité réelle.
func scanDemoIdentityLeaks(ctx context.Context, dbPath string, real realIdentities) ([]demoIdentityLeak, error) {
	db, err := duckdb.OpenReadOnly(dbPath)
	if err != nil {
		return nil, fmt.Errorf("base démo %s: %w", dbPath, err)
	}
	defer func() { _ = db.Close() }()
	rows, err := db.SQLDb().QueryContext(ctx, `SELECT c.table_name, c.column_name FROM duckdb_columns() c
		JOIN duckdb_tables() t ON t.database_name = c.database_name AND t.schema_name = c.schema_name
			AND t.table_name = c.table_name
		WHERE c.data_type LIKE '%VARCHAR%' OR c.data_type = 'JSON' OR c.column_name ILIKE '%xuid%'
		ORDER BY 1, 2`)
	if err != nil {
		return nil, fmt.Errorf("colonnes de %s: %w", dbPath, err)
	}
	var cols [][2]string
	for rows.Next() {
		var tc [2]string
		if err := rows.Scan(&tc[0], &tc[1]); err != nil {
			_ = rows.Close()
			return nil, err
		}
		cols = append(cols, tc)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var leaks []demoIdentityLeak
	for _, tc := range cols {
		found, err := scanColumn(ctx, db, tc[0], tc[1], real)
		if err != nil {
			return nil, err
		}
		for _, v := range found {
			leaks = append(leaks, demoIdentityLeak{DB: dbPath, Table: tc[0], Column: tc[1], Value: v})
		}
	}
	return leaks, nil
}

// scanColumn rend les valeurs distinctes d'une colonne qui portent une identité réelle.
func scanColumn(ctx context.Context, db *duckdb.DB, table, column string, real realIdentities) ([]string, error) {
	q := fmt.Sprintf(`SELECT DISTINCT CAST("%s" AS VARCHAR) FROM "%s" WHERE "%s" IS NOT NULL`, column, table, column)
	rows, err := db.SQLDb().QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s.%s: %w", table, column, err)
	}
	defer func() { _ = rows.Close() }()
	var found []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if real.leakIn(v) != "" {
			found = append(found, v)
		}
	}
	return found, rows.Err()
}

// verifyDemoAnonymization contrôle les bases générées du titre (partagée, sociale, joueurs
// seedés) et échoue sur la première fuite, en les nommant toutes.
func verifyDemoAnonymization(ctx context.Context, opts SeedDemoOptions, layout titlePkg.DemoLayout,
	seeded []seededDemoPlayer, roster []demoRosterEntry) error {
	real, err := loadRealIdentities(ctx, opts, roster)
	if err != nil {
		return fmt.Errorf("contrôle d'anonymisation: %w", err)
	}
	slug := opts.TitleSlug
	paths := []string{layout.SharedDBPath(slug), layout.SharedSocialDBPath(slug), layout.SharedPVEDBPath(slug)}
	for _, p := range seeded {
		paths = append(paths, layout.PlayerDBPath(slug, p.Dir))
	}
	parColonne := map[string]int{}
	for _, p := range paths {
		if !fileExists(p) {
			continue
		}
		found, err := scanDemoIdentityLeaks(ctx, p, real)
		if err != nil {
			return fmt.Errorf("contrôle d'anonymisation: %w", err)
		}
		for _, l := range found {
			parColonne[l.where()]++
		}
	}
	if len(parColonne) == 0 {
		return nil
	}
	lignes := make([]string, 0, len(parColonne))
	for w, n := range parColonne {
		lignes = append(lignes, fmt.Sprintf("%s (%d valeur(s))", w, n))
	}
	sort.Strings(lignes)
	return fmt.Errorf("contrôle d'anonymisation : identités réelles dans %d colonne(s) de la démo :\n  %s",
		len(lignes), strings.Join(lignes, "\n  "))
}
