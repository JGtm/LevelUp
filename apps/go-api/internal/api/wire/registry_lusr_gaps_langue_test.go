package wire

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// Le libellé de playlist d'un trou LUSR suit la langue de la requête ; en anglais, ou sans traduction, le
// nom du registre reste.
func TestNameLUSRGapPlaylists_SuitLaLangueDeLaRequete(t *testing.T) {
	meta, err := duckdb.OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR, lang VARCHAR, name VARCHAR)`,
		`INSERT INTO mode_name_tr VALUES ('Slayer','fr','Assassin')`,
	} {
		if _, err := meta.Exec(ctx, q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}
	r := &ServiceRegistry{resolveByGT: func(_ context.Context, _, _ string) (*duckdb.PlayerDB, error) {
		return &duckdb.PlayerDB{Metadata: meta}, nil
	}}
	gaps := func() []domain.LUSRGapItem {
		return []domain.LUSRGapItem{{Playlist: "Arena:Slayer on Bazaar"}, {Playlist: "Arena:Mystery on Bazaar"}}
	}

	fr := gaps()
	r.nameLUSRGapPlaylists(ctxkeys.WithLocale(ctx, "fr"), "halo_infinite", "gt", fr)
	if fr[0].Playlist != "Assassin" || fr[1].Playlist != "Arena:Mystery on Bazaar" {
		t.Errorf("fr : %+v", fr)
	}
	en := gaps()
	r.nameLUSRGapPlaylists(ctxkeys.WithLocale(ctx, "en"), "halo_infinite", "gt", en)
	if en[0].Playlist != "Arena:Slayer on Bazaar" {
		t.Errorf("en : %+v", en)
	}
}
