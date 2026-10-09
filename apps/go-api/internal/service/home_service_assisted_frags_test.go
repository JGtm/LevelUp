package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestEnrichMatchesWithAssistedFrags_PartialMap : seuls les matchs présents dans la map
// du repo portent l'objet ; un match hors de la mesure reste nil, et « zéro assisté »
// porte un objet à Total 0. La base est le compte officiel de la tuile.
func TestEnrichMatchesWithAssistedFrags_PartialMap(t *testing.T) {
	repo := &mockHomeRepo{
		assistedFrags: map[string]domain.MatchAssistedFrags{
			"m1": {FragsFilm: 12, Received: domain.AssistTiers{Total: 7, Low: 2, Mid: 3, High: 1}},
			"m3": {FragsFilm: 4},
		},
	}
	kills, killsM3 := 20, 4
	items := []domain.RecentMatchItem{{MatchID: "m1", Kills: &kills}, {MatchID: "m2"}, {MatchID: "m3", Kills: &killsM3}}

	enrichMatchesWithAssistedFrags(context.Background(), repo, items)

	if items[0].AssistedFrags == nil {
		t.Fatal("m1 : objet attendu")
	}
	// Base = frags officiels de la tuile (20), même quand le film n'en lit que 12.
	if got := *items[0].AssistedFrags; got.Received.Total != 7 || got.Received.Mid != 3 || got.FragsOfficial != 20 {
		t.Fatalf("m1 = %+v", got)
	}
	if items[1].AssistedFrags != nil {
		t.Fatalf("m2 absent de la map : nil attendu, got %+v", *items[1].AssistedFrags)
	}
	if items[2].AssistedFrags == nil || items[2].AssistedFrags.Received.Total != 0 || items[2].AssistedFrags.FragsOfficial != 4 {
		t.Fatalf("m3 (zéro assisté) = %+v", items[2].AssistedFrags)
	}
}

// TestEnrichMatchesWithAssistedFrags_RepoError : erreur repo → aucun item enrichi, pas de
// panic (la journalisation WARN est le contrat, la dégradation est totale).
func TestEnrichMatchesWithAssistedFrags_RepoError(t *testing.T) {
	repo := &mockHomeRepo{assistedFragsErr: errors.New("boom")}
	items := []domain.RecentMatchItem{{MatchID: "m1"}, {MatchID: "m2"}}

	enrichMatchesWithAssistedFrags(context.Background(), repo, items)

	for _, it := range items {
		if it.AssistedFrags != nil {
			t.Fatalf("%s : nil attendu après erreur repo, got %+v", it.MatchID, *it.AssistedFrags)
		}
	}
}

// countingAssistedFragsRepo compte les appels au repo (liste vide = aucun appel).
type countingAssistedFragsRepo struct {
	mockHomeRepo
	calls int
}

func (c *countingAssistedFragsRepo) LoadMatchAssistedFrags(ctx context.Context, ids []string) (map[string]domain.MatchAssistedFrags, error) {
	c.calls++
	return c.mockHomeRepo.LoadMatchAssistedFrags(ctx, ids)
}

// TestEnrichMatchesWithAssistedFrags_Empty : liste vide → aucun appel au repo.
func TestEnrichMatchesWithAssistedFrags_Empty(t *testing.T) {
	repo := &countingAssistedFragsRepo{}
	enrichMatchesWithAssistedFrags(context.Background(), repo, nil)
	if repo.calls != 0 {
		t.Fatalf("liste vide : 0 appel attendu, got %d", repo.calls)
	}
}
