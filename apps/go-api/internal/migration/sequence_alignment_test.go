//go:build cgo

package migration

// sequence_alignment_test.go — contrat de AlignSequencesToColumns sur une vraie base fichier :
// une séquence en retard est avancée et l'insertion suivante ne collisionne plus (y compris
// après fermeture/réouverture), une base saine ou déjà alignée n'est pas touchée, le compteur
// réel est lu même quand les colonnes start_value/last_value sont périmées (rejeu du WAL).

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func openSeqTestDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func execSeq(t *testing.T, db *sql.DB, stmts ...string) {
	t.Helper()
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
}

func nextOf(t *testing.T, db *sql.DB, seq string) int64 {
	t.Helper()
	n, err := sequenceNextValue(context.Background(), db, seq)
	if err != nil {
		t.Fatalf("sequenceNextValue(%s): %v", seq, err)
	}
	return n
}

// lagSchema : séquence à START 1, ids 1..10 déjà posés EXPLICITEMENT (forme d'une base legacy
// dont la séquence a été recréée après coup).
var lagSchema = []string{
	`CREATE SEQUENCE s START 1`,
	`CREATE TABLE t (id BIGINT DEFAULT nextval('s') PRIMARY KEY, v INTEGER)`,
	`INSERT INTO t (id, v) SELECT range + 1, range FROM range(10)`,
}

func TestAlignSequences_AvanceUneSequenceEnRetard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lag.duckdb")
	db := openSeqTestDB(t, path)
	execSeq(t, db, lagSchema...)

	if _, err := db.Exec(`INSERT INTO t (v) VALUES (99)`); err == nil ||
		!strings.Contains(err.Error(), "Duplicate key") {
		t.Fatalf("prémisse : l'insertion par défaut doit collisionner avant alignement, err=%v", err)
	}

	got, err := AlignSequencesToColumns(context.Background(), db)
	if err != nil {
		t.Fatalf("AlignSequencesToColumns: %v", err)
	}
	if len(got) != 1 || got[0].Sequence != "s" || got[0].ColumnMax != 10 || got[0].NextAfter <= 10 {
		t.Fatalf("alignement inattendu: %+v", got)
	}
	var id int64
	if err := db.QueryRow(`INSERT INTO t (v) VALUES (100) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("insertion après alignement: %v", err)
	}
	if id <= 10 {
		t.Fatalf("id rendu %d, attendu > 10", id)
	}
	_ = db.Close()

	// La consommation est durable : relue SEULE après réouverture.
	db2 := openSeqTestDB(t, path)
	if next := nextOf(t, db2, "s"); next <= id {
		t.Fatalf("après réouverture : prochaine valeur %d <= dernier id %d", next, id)
	}
	if _, err := db2.Exec(`INSERT INTO t (v) VALUES (101)`); err != nil {
		t.Fatalf("insertion après réouverture: %v", err)
	}
}

// La consommation SEULE (aucune insertion derrière) survit à la fermeture, y compris quand la
// base se rouvre par rejeu du WAL (fermeture sans point de reprise).
func TestAlignSequences_ConsommationDurableSansInsertion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "durable.duckdb")
	db := openSeqTestDB(t, path)
	execSeq(t, db, lagSchema...)
	execSeq(t, db, `CHECKPOINT`, `PRAGMA disable_checkpoint_on_shutdown`)
	if got, err := AlignSequencesToColumns(context.Background(), db); err != nil || len(got) != 1 {
		t.Fatalf("alignement: got=%+v err=%v", got, err)
	}
	_ = db.Close()

	db2 := openSeqTestDB(t, path)
	if next := nextOf(t, db2, "s"); next <= 10 {
		t.Fatalf("consommation perdue à la réouverture : prochaine valeur %d <= max 10", next)
	}
	if _, err := db2.Exec(`INSERT INTO t (v) VALUES (1)`); err != nil {
		t.Fatalf("insertion après réouverture: %v", err)
	}
}

// Cas limite : prochaine valeur ÉGALE au max (le premier id rendu serait déjà pris).
func TestAlignSequences_ProchaineValeurEgaleAuMax(t *testing.T) {
	db := openSeqTestDB(t, filepath.Join(t.TempDir(), "eq.duckdb"))
	execSeq(t, db,
		`CREATE SEQUENCE s START 1`,
		`CREATE TABLE t (id BIGINT DEFAULT nextval('s') PRIMARY KEY, v INTEGER)`,
		`INSERT INTO t (v) SELECT range FROM range(4)`, // ids 1..4, prochaine = 5
		`INSERT INTO t (id, v) VALUES (5, 0)`,          // 5 pris à la main
	)
	if next := nextOf(t, db, "s"); next != 5 {
		t.Fatalf("prémisse : prochaine valeur %d, attendu 5", next)
	}
	got, err := AlignSequencesToColumns(context.Background(), db)
	if err != nil {
		t.Fatalf("AlignSequencesToColumns: %v", err)
	}
	if len(got) != 1 || got[0].NextBefore != 5 || got[0].NextAfter != 6 || got[0].Consumed != 1 {
		t.Fatalf("alignement inattendu: %+v", got)
	}
	if _, err := db.Exec(`INSERT INTO t (v) VALUES (9)`); err != nil {
		t.Fatalf("insertion après alignement: %v", err)
	}
}

func TestAlignSequences_Idempotent(t *testing.T) {
	db := openSeqTestDB(t, filepath.Join(t.TempDir(), "idem.duckdb"))
	execSeq(t, db, lagSchema...)
	if got, err := AlignSequencesToColumns(context.Background(), db); err != nil || len(got) != 1 {
		t.Fatalf("1er passage: got=%+v err=%v", got, err)
	}
	before := nextOf(t, db, "s")
	got, err := AlignSequencesToColumns(context.Background(), db)
	if err != nil || len(got) != 0 {
		t.Fatalf("2e passage : aucune séquence ne doit bouger, got=%+v err=%v", got, err)
	}
	if after := nextOf(t, db, "s"); after != before {
		t.Fatalf("2e passage a consommé : %d -> %d", before, after)
	}
}

func TestAlignSequences_BaseSaineInchangee(t *testing.T) {
	db := openSeqTestDB(t, filepath.Join(t.TempDir(), "saine.duckdb"))
	execSeq(t, db,
		`CREATE SEQUENCE s START 1`,
		`CREATE SEQUENCE vide START 1`,
		`CREATE TABLE t (id BIGINT DEFAULT nextval('s') PRIMARY KEY, v INTEGER)`,
		`CREATE TABLE u (id BIGINT DEFAULT nextval('vide'), v INTEGER)`,
		`INSERT INTO t (v) SELECT range FROM range(7)`,
		`INSERT INTO u (id, v) VALUES (NULL, 1)`, // colonne entièrement NULL : rien à aligner
	)
	before := nextOf(t, db, "s")
	got, err := AlignSequencesToColumns(context.Background(), db)
	if err != nil || len(got) != 0 {
		t.Fatalf("base saine : got=%+v err=%v", got, err)
	}
	if after := nextOf(t, db, "s"); after != before {
		t.Fatalf("base saine modifiée : %d -> %d", before, after)
	}
	if after := nextOf(t, db, "vide"); after != 1 {
		t.Fatalf("séquence d'une colonne NULL consommée : %d", after)
	}
}

// Après rejeu du WAL, start_value/last_value restent ceux du dernier point de reprise ; le
// compteur réel (colonne sql) est déjà au-delà du max : rien à faire.
func TestAlignSequences_LitLeCompteurReelApresRejeuDuWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal.duckdb")
	db := openSeqTestDB(t, path)
	execSeq(t, db,
		`PRAGMA disable_checkpoint_on_shutdown`,
		`CREATE SEQUENCE s START 1`,
		`CREATE TABLE t (id BIGINT DEFAULT nextval('s') PRIMARY KEY, v INTEGER)`,
		`INSERT INTO t (v) SELECT range FROM range(5)`,
	)
	_ = db.Close()

	db2 := openSeqTestDB(t, path)
	before := nextOf(t, db2, "s")
	if before != 6 {
		t.Fatalf("prémisse : compteur réel %d, attendu 6", before)
	}
	got, err := AlignSequencesToColumns(context.Background(), db2)
	if err != nil || len(got) != 0 {
		t.Fatalf("base saine après rejeu du WAL : got=%+v err=%v", got, err)
	}
	if after := nextOf(t, db2, "s"); after != before {
		t.Fatalf("consommation à tort après rejeu du WAL : %d -> %d", before, after)
	}
}

// Une séquence partagée par deux colonnes s'aligne sur le plus grand des deux max ; les
// lignes (ids NULL compris) ne sont jamais modifiées.
func TestAlignSequences_SequencePartageeEtIdsNullIntacts(t *testing.T) {
	db := openSeqTestDB(t, filepath.Join(t.TempDir(), "partage.duckdb"))
	execSeq(t, db,
		`CREATE SEQUENCE s START 1`,
		`CREATE TABLE a (id BIGINT DEFAULT nextval('s'), v INTEGER)`,
		`CREATE TABLE b (id BIGINT DEFAULT nextval('main.s'), v INTEGER)`,
		`INSERT INTO a (id, v) VALUES (3, 0), (NULL, 1), (NULL, 2)`,
		`INSERT INTO b (id, v) VALUES (40, 0)`,
	)
	got, err := AlignSequencesToColumns(context.Background(), db)
	if err != nil {
		t.Fatalf("AlignSequencesToColumns: %v", err)
	}
	if len(got) != 1 || got[0].ColumnMax != 40 || got[0].NextAfter != 41 {
		t.Fatalf("alignement inattendu: %+v", got)
	}
	var nulls, total int
	if err := db.QueryRow(`SELECT count(*) FILTER (WHERE id IS NULL), count(*) FROM a`).Scan(&nulls, &total); err != nil {
		t.Fatalf("recompte: %v", err)
	}
	if nulls != 2 || total != 3 {
		t.Fatalf("lignes modifiées : %d NULL / %d lignes, attendu 2 / 3", nulls, total)
	}
}
