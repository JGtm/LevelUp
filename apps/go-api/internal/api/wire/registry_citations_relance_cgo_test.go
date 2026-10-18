//go:build cgo

package wire

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestCitationsRecomputer_ParticipantsParLeProviderDuTitre : la relance lit les participants du
// match dans la base partagée du titre, par son provider (jamais un handle ouvert à côté).
func TestCitationsRecomputer_ParticipantsParLeProviderDuTitre(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants VALUES ('m1','x1'),('m1','x2'),('m1','x2'),('m2','x3'),('m1',NULL)`); err != nil {
		t.Fatal(err)
	}
	reg := &ServiceRegistry{cfg: &config.AppConfig{
		RepoRoot: t.TempDir(), SharedProvider: sharedprovider.FromInMemoryDB(db, "shared.duckdb"),
	}}
	got, err := citationsRecomputer{r: reg}.Participants(context.Background(), "halo_infinite", "m1")
	if err != nil {
		t.Fatalf("Participants: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("participants = %v, attendu x1 et x2 une fois chacun", got)
	}

	sansProvider := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: t.TempDir()}}
	if _, err := (citationsRecomputer{r: sansProvider}).Participants(context.Background(), "halo_infinite", "m1"); err == nil {
		t.Fatal("sans base partagée câblée, une erreur est attendue")
	}
}
