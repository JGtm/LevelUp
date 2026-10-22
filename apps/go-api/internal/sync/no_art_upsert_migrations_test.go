package sync

// no_art_upsert_migrations_test.go — les TABLES PROTÉGÉES (tablesProtegees) restent interdites
// d'UPSERT jusque dans les migrations.
//
// Le périmètre commun des scans anti-ART (dansLePerimetreART) exclut les migrations (une étape
// tourne une fois, au démarrage, mono-processus) ; ce scan l'étend aux fichiers de migration pour
// les tables protégées SEULEMENT : une migration garde le droit d'écrire en UPSERT une table qui
// n'est pas protégée.
//
// Détection STATEMENT-level (et non file-level comme TestNoARTPatternsOnProtectedTables) : un
// fichier de migration nomme souvent plusieurs tables, et seul compte l'ordre SQL dont la cible
// est protégée. Chaque littéral Go est découpé en ordres au `;` ; un ordre est en violation s'il
// insère dans une table protégée (`INSERT [OR …] INTO <table>`) et porte `ON CONFLICT … DO
// UPDATE`, ou s'il est un `INSERT OR REPLACE INTO <table>`.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// allowlistUpsertMigrations : fichier de migration -> justification DATÉE d'un UPSERT conservé
// sur une table protégée. Vide : aucun n'est toléré.
var allowlistUpsertMigrations = map[string]string{}

// dansLesMigrations : fichier de production d'un paquet de migrations (registre global ou jeu
// title-owned).
func dansLesMigrations(path string) bool {
	if strings.HasSuffix(path, "_test.go") {
		return false
	}
	p := filepath.ToSlash(path)
	return strings.Contains(p, "/migration/") || strings.Contains(p, "/migrations/")
}

var (
	reUpsertDoUpdate  = regexp.MustCompile(`(?is)\bON\s+CONFLICT\b.*\bDO\s+UPDATE\b`)
	reInsertOrReplace = regexp.MustCompile(`(?is)^\s*INSERT\s+OR\s+REPLACE\b`)
)

// upsertProtegeViolations rend les ordres SQL d'un source déjà déscommenté qui écrivent une
// table protégée en UPSERT. Fonction pure : c'est elle que le témoin exerce.
func upsertProtegeViolations(text string, tables []string) []string {
	cibles := make(map[string]*regexp.Regexp, len(tables))
	for _, table := range tables {
		cibles[table] = regexp.MustCompile(`(?is)^\s*INSERT\s+(?:OR\s+\w+\s+)?INTO\s+"?` +
			regexp.QuoteMeta(table) + `"?(?:\s|\(|$)`)
	}
	var out []string
	for _, lit := range reGoStringLiteral.FindAllString(text, -1) {
		for _, ordre := range strings.Split(strings.Trim(lit, "`\""), ";") {
			for table, re := range cibles {
				if !re.MatchString(ordre) {
					continue
				}
				if reUpsertDoUpdate.MatchString(ordre) || reInsertOrReplace.MatchString(ordre) {
					out = append(out, "table="+table)
				}
			}
		}
	}
	return out
}

// TestNoUpsertOnProtectedTablesInMigrations — le scan des migrations du module.
func TestNoUpsertOnProtectedTablesInMigrations(t *testing.T) {
	repoRoot := findRepoRoot(t)
	var violations []string
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" ||
				name == "data" || name == "logs" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || !dansLesMigrations(path) {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(repoRoot, path)
		rel = filepath.ToSlash(rel)
		if _, allowed := allowlistUpsertMigrations[rel]; allowed {
			return nil
		}
		for _, v := range upsertProtegeViolations(stripGoComments(string(content)), tablesProtegees) {
			violations = append(violations, v+" file="+rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("UPSERT (ON CONFLICT DO UPDATE / INSERT OR REPLACE) sur une table protégée dans une "+
			"migration : %d :\n  - %s\nRemède : SELECT-then-write (duckdbkeys.Upsert) ou "+
			"INSERT pur sur une table append-only.", len(violations), strings.Join(violations, "\n  - "))
	}
	for fichier := range allowlistUpsertMigrations {
		if _, statErr := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(fichier))); statErr != nil {
			t.Errorf("allowlistUpsertMigrations : %q introuvable — entrée à retirer", fichier)
		}
	}
}

// TestUpsertProtegeDetection_Sanity — LE TÉMOIN : sans lui, un scan qui ne détecte plus rien
// passerait pour vert.
func TestUpsertProtegeDetection_Sanity(t *testing.T) {
	tables := []string{"playlists_catalog", "streak"}
	cas := []struct {
		nom    string
		source string
		rouge  bool
	}{
		{
			// VERBATIM (à l'indentation près) : la forme que portait ranked_playlists.go avant
			// le correctif du constat C4.
			nom: "UPSERT DO UPDATE sur playlists_catalog (forme réelle du constat C4)",
			source: "db.ExecContext(ctx, `\n\t\t\tINSERT INTO playlists_catalog\n\t\t\t  (title_slug, playlist_asset_id, current_version_id, name_canonical,\n" +
				"\t\t\t   experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)\n" +
				"\t\t\tVALUES (?, ?, '', ?, 'ranked', TRUE, ?, ?, ?, ?)\n" +
				"\t\t\tON CONFLICT (title_slug, playlist_asset_id) DO UPDATE SET\n\t\t\t  name_canonical = excluded.name_canonical`)",
			rouge: true,
		},
		{
			nom:    "INSERT OR REPLACE sur une table protégée",
			source: "q := `INSERT OR REPLACE INTO streak (id) VALUES (?)`",
			rouge:  true,
		},
		{
			nom:    "UPSERT sur une table NON protégée (toléré en migration)",
			source: "q := `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, 'playlist', ?, ?) ON CONFLICT (asset_id, asset_type, lang) DO UPDATE SET name = excluded.name`",
			rouge:  false,
		},
		{
			nom:    "ON CONFLICT DO NOTHING sur une table protégée (pas un UPDATE)",
			source: "q := `INSERT INTO playlists_catalog (title_slug) VALUES (?) ON CONFLICT DO NOTHING`",
			rouge:  false,
		},
		{
			nom:    "table protégée et UPSERT dans deux ordres distincts du même littéral",
			source: "s := `INSERT INTO playlists_catalog (a) VALUES (1); INSERT INTO other_t (a) VALUES (1) ON CONFLICT (a) DO UPDATE SET a = 2`",
			rouge:  false,
		},
		{
			nom:    "nom de table protégée en préfixe d'une autre table",
			source: "q := `INSERT INTO streak_history_x (id) VALUES (1) ON CONFLICT (id) DO UPDATE SET id = 2`",
			rouge:  false,
		},
	}
	for _, c := range cas {
		got := upsertProtegeViolations(c.source, tables)
		if c.rouge && len(got) == 0 {
			t.Errorf("%s : le garde-rail NE MORD PAS", c.nom)
		}
		if !c.rouge && len(got) > 0 {
			t.Errorf("%s : faux positif — %v", c.nom, got)
		}
	}
}
