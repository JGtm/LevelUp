package migration

// written_at_utc_guard_mutation_test.go — PREUVE que written_at_utc_guard_test.go MORD.
//
// Un garde-rail non testé sur les régressions qu'il prétend interdire n'est qu'un
// commentaire exécutable (même raisonnement que TestStartTimeGuardsAreDiscriminant,
// internal/archlint). Ce fichier soumet au détecteur, pour CHAQUE famille, la forme
// fautive ET la forme canonique : la première doit être vue, la seconde ignorée.
//
// Le détecteur est exercé par sa VRAIE porte d'entrée — inspecterFichier sur un
// fichier Go de synthèse — et non par ses sous-fonctions. Une règle correcte que
// l'inspection AST n'atteindrait pas serait un faux vert.
//
// Les formes fautives sont assemblées par CONCATENATION : écrites d'un seul tenant
// elles feraient échouer le garde-rail contre ce fichier même. Le procédé est celui
// de la définition canonique de SQLStartTimeCanonical, coupée de la même manière.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// horlogeNue / horlogeNueAlt — les deux expressions sensibles au fuseau, coupées.
const (
	horlogeNue    = "now" + "()"
	horlogeNueAlt = "CURRENT_" + "TIMESTAMP"
)

// ecrireSourceDeSynthese pose un fichier Go compilable HORS du dépôt (t.TempDir),
// donc invisible du parcours de TestWrittenAtEcrituresEnUTC, et rend son chemin.
func ecrireSourceDeSynthese(t *testing.T, corps string) string {
	t.Helper()
	chemin := filepath.Join(t.TempDir(), "synthese.go")
	source := "package synthese\n\nimport \"time\"\n\nvar _ = time.Now\n\n" + corps
	if err := os.WriteFile(chemin, []byte(source), 0o600); err != nil {
		t.Fatalf("écriture de la source de synthèse: %v", err)
	}
	return chemin
}

// TestGardeRailWrittenAtMordSurChaqueFamille — cas fautifs : chacun DOIT être vu.
func TestGardeRailWrittenAtMordSurChaqueFamille(t *testing.T) {
	cas := []struct {
		famille string
		corps   string
	}{
		{
			// Famille 1 (S2) — non-régression du périmètre initial.
			famille: "DEFAULT de colonne",
			corps:   "const ddl = `CREATE TABLE t (written_at TIMESTAMP DEFAULT " + horlogeNue + ")`",
		},
		{
			famille: "ALTER ... SET DEFAULT",
			corps:   "const alt = `ALTER TABLE t ALTER COLUMN written_at SET DEFAULT " + horlogeNue + "`",
		},
		{
			famille: "colonne synthétique de rebuild",
			corps:   "const reb = `SELECT id, " + horlogeNueAlt + " AS written_at FROM t`",
		},
		{
			// Famille 2 (S5) — les trois formes d'écriture relevées dans le dépôt.
			famille: "INSERT ... VALUES",
			corps:   "const ins = `INSERT INTO t (xuid, written_at) VALUES (?, " + horlogeNueAlt + ")`",
		},
		{
			famille: "INSERT ... SELECT",
			corps:   "const sel = `INSERT INTO t (xuid, written_at) SELECT xuid, " + horlogeNue + " FROM s`",
		},
		{
			famille: "UPDATE ... SET",
			corps:   "const upd = `UPDATE t SET written_at = " + horlogeNue + " WHERE xuid = ?`",
		},
		{
			// Famille 2, versant LECTURE — le défaut de leaderboard_world_repo.
			famille: "âge calculé contre une horloge sensible au fuseau",
			corps:   "const age = `SELECT " + horlogeNueAlt + " - max(written_at) FROM t`",
		},
		{
			// Famille 3 (S5) — invisible des chaînes SQL : la valeur est liée.
			famille: "champ Go WrittenAt sur l'horloge locale",
			corps:   "var r = struct{ WrittenAt interface{} }{WrittenAt: time.Now()}",
		},
		{
			famille: "variable Go writtenAt sur l'horloge locale",
			corps:   "func f() interface{} { writtenAt := time.Now(); return writtenAt }",
		},
		{
			// Angle mort du relevé initial des 37 sites : l'ordre SQL n'existe
			// qu'après interpolation, aucune chaîne ne porte les deux moitiés
			// (steps_shared_social_media_assoc_append_only.go).
			famille: "horloge portée par une variable interpolée",
			corps: "func h(f string, a ...interface{}) {}\n" +
				"func i() { atExpr := \"" + horlogeNueAlt + "\"; " +
				"h(`INSERT INTO t (written_at) SELECT %s FROM s`, atExpr) }",
		},
		{
			famille: "horloge interpolée depuis une déclaration var",
			corps: "func h(f string, a ...interface{}) {}\n" +
				"var atExpr = \"COALESCE(created_at, " + horlogeNueAlt + ")\"\n" +
				"func i() { h(`INSERT INTO t (written_at) SELECT %s FROM s`, atExpr) }",
		},
		{
			// L'appel qui rapproche nom de colonne et type : aucune chaîne ne porte les deux.
			famille: "ajout de colonne written_at",
			corps: "func addColumnIfMissing(a, b, c, d string) {}\n" +
				"func g() { addColumnIfMissing(\"db\", \"t\", \"written_at\", \"TIMESTAMP DEFAULT " + horlogeNue + "\") }",
		},
	}

	for _, c := range cas {
		t.Run(c.famille, func(t *testing.T) {
			chemin := ecrireSourceDeSynthese(t, c.corps)
			if infractions := inspecterFichier(t, chemin); len(infractions) == 0 {
				t.Errorf("famille %q NON détectée — le garde-rail ne mord pas sur :\n  %s",
					c.famille, c.corps)
			}
		})
	}
}

// TestGardeRailWrittenAtEpargneLesFormesLegitimes — cas sains : aucun ne doit être vu.
// Sans ce volet, un détecteur qui hurle sur tout passerait pour un détecteur qui mord.
func TestGardeRailWrittenAtEpargneLesFormesLegitimes(t *testing.T) {
	canonique := "CAST(" + horlogeNue + " AT TIME ZONE 'UTC' AS TIMESTAMP)"

	cas := []struct {
		nom   string
		corps string
	}{
		{
			nom:   "DEFAULT canonique",
			corps: "const ddl = `CREATE TABLE t (written_at TIMESTAMP DEFAULT " + canonique + ")`",
		},
		{
			nom:   "INSERT canonique",
			corps: "const ins = `INSERT INTO t (xuid, written_at) VALUES (?, " + canonique + ")`",
		},
		{
			nom:   "lecture canonique",
			corps: "const age = `SELECT " + canonique + " - max(written_at) FROM t`",
		},
		{
			// steps_player_schema_authority.go : `created_at` n'arbitre aucune vue
			// `_latest`, son DEFAULT local ne relève pas de S5.
			nom: "DEFAULT local sur une AUTRE colonne, à côté d'un written_at canonique",
			corps: "const ddl = `CREATE TABLE t (created_at TIMESTAMP DEFAULT " + horlogeNueAlt +
				", written_at TIMESTAMP DEFAULT " + canonique + ")`",
		},
		{
			// steps_written_at_utc_default.go : le prédicat qui DÉTECTE le défaut
			// doit nommer l'horloge fautive sans jamais l'appeler.
			nom: "horloge citée dans un littéral SQL",
			corps: "const q = `SELECT table_name FROM duckdb_columns() WHERE column_name = 'written_at' " +
				"AND lower(column_default) LIKE '%" + horlogeNue + "%'`",
		},
		{
			// media_delete_test.go : deux instructions, aucune ne mélange deux horloges.
			nom: "instruction voisine, dans un script multi-instructions",
			corps: "const s = `CREATE VIEW v AS SELECT written_at FROM h ORDER BY written_at DESC; " +
				"INSERT INTO l (liked_at) VALUES (" + horlogeNueAlt + ");`",
		},
		{
			nom:   "champ Go WrittenAt en UTC explicite",
			corps: "var r = struct{ WrittenAt interface{} }{WrittenAt: time.Now().UTC()}",
		},
		{
			nom: "fragment interpolé en forme canonique",
			corps: "func h(f string, a ...interface{}) {}\n" +
				"func i() { atExpr := \"" + canonique + "\"; " +
				"h(`INSERT INTO t (written_at) SELECT %s FROM s`, atExpr) }",
		},
		{
			// Une variable d'horloge sans rapport ne doit pas contaminer un gabarit
			// qu'elle n'alimente pas.
			nom: "variable d'horloge non interpolée dans le gabarit written_at",
			corps: "func h(f string, a ...interface{}) {}\n" +
				"func i() { autre := \"" + horlogeNueAlt + "\"; _ = autre; " +
				"h(`INSERT INTO t (written_at) VALUES (?)`) }",
		},
		{
			// Le motif ne doit pas se déclencher sur une horloge Go sans rapport.
			nom:   "horloge Go sur une variable étrangère à written_at",
			corps: "func f() interface{} { debut := time.Now(); return debut }",
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			chemin := ecrireSourceDeSynthese(t, c.corps)
			if infractions := inspecterFichier(t, chemin); len(infractions) > 0 {
				t.Errorf("faux positif sur %q :\n  %s\n  source : %s",
					c.nom, strings.Join(infractions, "\n  "), c.corps)
			}
		})
	}
}
