// Package duckdb_test — is_file_lock_error_test.go : classification des erreurs de
// contention fichier (mono-writer DuckDB). Pas de DB ouverte : test pur sur la chaîne
// d'erreur. CGO requis uniquement parce que le package duckdb l'exige.
package duckdb_test

import (
	"errors"
	"testing"

	ddb "levelup/go-api/internal/platform/duckdb"
)

func TestIsFileLockError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{
			// Libellé réel observé au boot Windows (cf. logs/duckdb.log 2026-06-27) :
			// un tmp/server.exe résiduel d'Air tient encore le fichier.
			name: "windows_file_already_open",
			err: errors.New(`duckdb.OpenReadWrite(C:\...\stats.duckdb): database/sql/driver: ` +
				`could not connect to database: IO Error: Cannot open file "...stats.duckdb": ` +
				`Le processus ne peut pas accéder au fichier car ce fichier est utilisé par un autre processus.` +
				"\r\n\nFile is already open in \nC:\\...\\tmp\\server.exe (PID 34756)"),
			want: true,
		},
		{"linux_could_not_set_lock", errors.New("IO Error: Could not set lock on file"), true},
		{"conflicting_lock", errors.New("Conflicting lock is held in /proc/..."), true},
		{"different_configuration", errors.New("database is already opened with a different configuration than existing connections"), true},
		{"not_found_is_not_a_lock", errors.New(`IO Error: Cannot open file "x.duckdb": No such file or directory`), false},
		{"unrelated", errors.New("Catalog Error: Table with name foo does not exist"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ddb.IsFileLockError(tc.err); got != tc.want {
				t.Fatalf("IsFileLockError(%q) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
