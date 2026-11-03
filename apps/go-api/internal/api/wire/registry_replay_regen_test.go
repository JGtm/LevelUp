package wire

import (
	"errors"
	"slices"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replayregen"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/LevelUp/libs/film/filmproc"
)

// TestTitlesWithReplay_LesTitresQuiDeclarentLaCapacite : sur la configuration réelle du dépôt, la file de
// régénération ne retient que les titres qui produisent des rejeux (Halo Infinite), jamais Halo 5.
func TestTitlesWithReplay_LesTitresQuiDeclarentLaCapacite(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{Slug: "halo_5", Name: "Halo 5", Provider: "halo_5", Status: title.StatusActive})
	got := titlesWithReplay(root, reg)
	if !slices.Equal(got, []string{title.DefaultSlug}) {
		t.Fatalf("titres = %v, attendu seulement %q", got, title.DefaultSlug)
	}
}

// TestRegenerateReplay_UnDecodageEnCoursSeRendEnErrBusy : l'action admin déjà en vol et le verrou solo tenu par un
// autre décodage sont des attentes, pas des échecs : la file recommence le même film.
func TestRegenerateReplay_UnDecodageEnCoursSeRendEnErrBusy(t *testing.T) {
	for _, cause := range []error{ErrActionBusy, filmproc.ErrDecodeBusy} {
		err := busyOr(cause)
		if !errors.Is(err, replayregen.ErrBusy) || !errors.Is(err, cause) {
			t.Fatalf("%v : %v ne s'enveloppe pas en ErrBusy", cause, err)
		}
	}
	other := errors.New("film absent")
	if err := busyOr(other); errors.Is(err, replayregen.ErrBusy) || !errors.Is(err, other) {
		t.Fatalf("une autre erreur reste elle-même : %v", err)
	}
}
