package sync

import (
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/ops"
)

// TestClassifySnapshotCutErr : chaque sentinelle ops.ErrSnapshot* est réduite à la
// bonne raison de l'enum fermé ; une erreur inconnue tombe dans "other".
func TestClassifySnapshotCutErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"copy", fmt.Errorf("%w: COPY x", ops.ErrSnapshotCopy), snapCutFailCopy},
		{"manifest", fmt.Errorf("%w: flip", ops.ErrSnapshotManifest), snapCutFailManifest},
		{"read", fmt.Errorf("%w: gather", ops.ErrSnapshotRead), snapCutFailRead},
		{"other", errors.New("inconnu"), snapCutFailOther},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifySnapshotCutErr(c.err); got != c.want {
				t.Errorf("classifySnapshotCutErr(%v) = %q, attendu %q", c.err, got, c.want)
			}
		})
	}
}
