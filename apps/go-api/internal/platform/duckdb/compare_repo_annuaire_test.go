//go:build integration

// Package duckdb — compare_repo_annuaire_test.go : LA PARITÉ DES NOMS de la page Comparer (lot
// perf L7). GetLocalStats ne joint plus v_gamertag_lookup : le nom vient de l'annuaire de la
// lecture, sur l'historique du joueur comparé — tous ses matchs hors Campagne, ceux que la
// lecture agrège. La référence est l'ancienne expression sur la VRAIE vue (nomSelonLaVue).
package duckdb

import (
	"context"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"
)

// TestCompareRepo_Annuaire_MemeNomQueLaVue : un joueur comparé par niveau de la cascade, plus
// x_autre, qui n'a jamais croisé le joueur du repo : son nom se lit sur SON historique (mcx).
func TestCompareRepo_Annuaire_MemeNomQueLaVue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedAnnuaire(t, pdb)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry (match_id) VALUES ('mcx')`)
	execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants (match_id, xuid, gamertag, outcome, team_id)
		VALUES ('mcx', 'x_autre', 'NomAutre', 2, 0)`)
	repo := NewCompareRepo(pdb)
	for _, x := range []string{"x_alias", "x_alias_vide", "x_part", "x_triple", "x_kfl", "x_ennemi", "x_kvp", "x_rien", "x_autre"} {
		s, err := repo.GetLocalStats(ctx, x, titlepkg.DefaultSlug)
		if err != nil {
			t.Fatalf("GetLocalStats(%s) : %v", x, err)
		}
		verifierNom(t, pdb, "Comparer", s.XUID, s.Gamertag)
		if x == "x_autre" && s.Gamertag != "NomAutre" {
			t.Errorf("x_autre nommé %q, attendu NomAutre (son historique, pas celui du joueur du repo)", s.Gamertag)
		}
	}
}
