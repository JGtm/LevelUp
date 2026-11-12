package sqlfrag

import (
	"strings"
	"testing"
)

// TestAnnuaireSQL_PorteeBase : la portée base lit les participants sur TOUTE la base — aucun
// `match_id` —, la portée de la lecture les borne à ses matchs (un seul gabarit par niveau) ; la
// localisation du repli (DA.10) ne projette que des match_id.
func TestAnnuaireSQL_PorteeBase(t *testing.T) {
	base := AnnuaireNomsBaseSQL()
	if strings.Contains(base, "match_id") {
		t.Errorf("AnnuaireNomsBaseSQL borne par match :\n%s", base)
	}
	if lecture := AnnuaireNomsSQL(); strings.Replace(lecture, " AND "+SQLDansListeParJointure("match_id"), "", 1) != base {
		t.Errorf("les deux portées divergent hors de la borne de match :\n%s\n---\n%s", lecture, base)
	}
	// DA.10 : la localisation ne rend QUE des match_id (lecture brute admise pour localiser,
	// jamais pour lire une valeur) — aucune colonne de nom dans ce qu'elle projette.
	loc := AnnuaireKillFeedLocaliserSQL()
	if n := strings.Count(loc, "SELECT match_id FROM"); n != 4 {
		t.Errorf("AnnuaireKillFeedLocaliserSQL : %d projections « SELECT match_id », attendu 4 :\n%s", n, loc)
	}
	for _, interdit := range []string{"SELECT xuid, gamertag", "MAX(", "_gamertag AS", "SELECT *"} {
		if strings.Contains(loc, interdit) {
			t.Errorf("AnnuaireKillFeedLocaliserSQL lit une valeur (%q) :\n%s", interdit, loc)
		}
	}
}
