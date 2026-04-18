//go:build integration

package notify

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openMediaNotifyDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE SEQUENCE media_id_seq START 1;
		CREATE TABLE media_files (
			id INTEGER DEFAULT nextval('media_id_seq'),
			file_path VARCHAR PRIMARY KEY,
			file_name VARCHAR,
			kind VARCHAR,
			indexed_at TIMESTAMPTZ DEFAULT now(),
			discord_notified_at TIMESTAMPTZ
		);
		CREATE TABLE media_match_associations (
			media_file_id INTEGER,
			match_id VARCHAR
		);
	`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestQueryUnnotifiedMedia_Empty(t *testing.T) {
	db := openMediaNotifyDB(t)
	rows, err := queryUnnotifiedMedia(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0, got %d", len(rows))
	}
}

func TestQueryUnnotifiedMedia_WithData(t *testing.T) {
	db := openMediaNotifyDB(t)
	db.Exec(`INSERT INTO media_files (file_path, file_name, kind) VALUES
		('/path/a.mp4', 'a.mp4', 'video'),
		('/path/b.png', 'b.png', 'image')`)

	rows, err := queryUnnotifiedMedia(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2, got %d", len(rows))
	}
}

func TestQueryUnnotifiedMedia_SkipsNotified(t *testing.T) {
	db := openMediaNotifyDB(t)
	db.Exec(`INSERT INTO media_files (file_path, file_name, kind, discord_notified_at) VALUES
		('/path/a.mp4', 'a.mp4', 'video', now())`)
	db.Exec(`INSERT INTO media_files (file_path, file_name, kind) VALUES
		('/path/b.png', 'b.png', 'image')`)

	rows, err := queryUnnotifiedMedia(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1, got %d", len(rows))
	}
}

func TestMarkMediaNotified_Empty(t *testing.T) {
	db := openMediaNotifyDB(t)
	err := markMediaNotified(db, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMarkMediaNotified_WithData(t *testing.T) {
	db := openMediaNotifyDB(t)
	db.Exec(`INSERT INTO media_files (file_path, file_name, kind) VALUES
		('/path/a.mp4', 'a.mp4', 'video'),
		('/path/b.png', 'b.png', 'image')`)

	err := markMediaNotified(db, []string{"/path/a.mp4"})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM media_files WHERE discord_notified_at IS NOT NULL").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 notified, got %d", count)
	}
}
