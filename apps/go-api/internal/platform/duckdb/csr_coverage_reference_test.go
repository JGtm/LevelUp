package duckdb

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// Le contrôle croisé applique la référence des playlists classées à des comptes par playlist :
// il ne décide d'aucun classement, il alerte.
func TestControlCSRAgainstReference(t *testing.T) {
	ref := rankedplaylists.All()[0].AssetID
	got := controlCSRAgainstReference([]playlistCSRCount{
		{playlistID: ref, matches: 10, withCSR: 7},               // 3 matchs de playlist classée sans CSR
		{playlistID: "hors-reference-a", matches: 4, withCSR: 0}, // social normal : rien
		{playlistID: "hors-reference-b", matches: 2, withCSR: 2}, // CSR hors référence : 2
	})
	if got.ReferenceMatchesWithoutCSR != 3 || got.CSRMatchesOffReference != 2 || !got.Alert {
		t.Errorf("contrôle = %+v, want without_csr=3 off_reference=2 alert=true", got)
	}

	clean := controlCSRAgainstReference([]playlistCSRCount{
		{playlistID: ref, matches: 5, withCSR: 5},
		{playlistID: "hors-reference-a", matches: 9, withCSR: 0},
	})
	if clean.Alert || clean.ReferenceMatchesWithoutCSR != 0 || clean.CSRMatchesOffReference != 0 {
		t.Errorf("couverture complète : pas d'alerte attendue, got %+v", clean)
	}
	if controlCSRAgainstReference(nil).Alert {
		t.Error("aucun match : pas d'alerte")
	}
}
