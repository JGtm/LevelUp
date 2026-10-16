package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// writerSpy compte les prises du writer partagé ; elle le refuse pour ne rien écrire.
type writerSpy struct {
	sharedprovider.Provider
	acquired atomic.Int32
}

var errSpyWriter = errors.New("writer refuse par le test")

func (s *writerSpy) AcquireWriter(context.Context) (*sharedprovider.WriterHandle, error) {
	s.acquired.Add(1)
	return nil, errSpyWriter
}

// Le writer partagé n'est pris qu'APRÈS la validation du fichier : un fichier invalide,
// ou d'un autre propriétaire, ne bascule jamais la base partagée en écriture.
func TestOpenSpartanImport_SharedWriterOnlyAfterFileValidation(t *testing.T) {
	dir := t.TempDir()

	garbage := filepath.Join(dir, "garbage.db")
	if err := os.WriteFile(garbage, []byte("ceci n'est pas une base SQLite"), 0o600); err != nil {
		t.Fatal(err)
	}
	foreign := buildMinimalFixtureDB(t, dir, "foreign.db", "2533274811111111")
	valid := buildMinimalFixtureDB(t, dir, "owner.db", testOwnerXUID)

	cases := []struct {
		name        string
		path        string
		wantWriters int32
	}{
		{"fichier qui n'est pas une base OpenSpartan", garbage, 0},
		{"base d'un autre propriétaire", foreign, 0},
		{"base valide du propriétaire", valid, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spy := &writerSpy{}
			svc := NewOpenSpartanImportService(spy, "")
			_, err := svc.Import(context.Background(), testOwnerXUID, tc.path, ImportOptions{StashDir: dir})
			if err == nil {
				t.Fatal("l'import devait échouer (writer refusé ou fichier rejeté)")
			}
			if got := spy.acquired.Load(); got != tc.wantWriters {
				t.Fatalf("writer partagé pris %d fois, attendu %d (err = %v)", got, tc.wantWriters, err)
			}
			if tc.wantWriters == 1 && !errors.Is(err, errSpyWriter) {
				t.Fatalf("err = %v, attendu l'échec du writer", err)
			}
		})
	}
}
