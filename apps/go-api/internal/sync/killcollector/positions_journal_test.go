package killcollector

import (
	"context"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
)

// TestCollectPositions_LeJournalFusionneOuvreLaPasse — E8.1 (a), decision D-16 : les couples
// a placer viennent du JOURNAL FUSIONNE. Un joueur renomme (le fil du film le nomme « Fluxnic »,
// la base « DI0NIC ») n a de xuid que par le credit : sa ligne de film porte des xuids VIDES, la
// ligne fusionnee les porte. Avant le correctif, la passe lisait la liste pre-fusion et s arretait
// a « rien a placer » ; elle lit maintenant le journal, et va jusqu a la resolution de la carte
// (ici hors catalogue, compte par `metricPositionsNoMap`).
func TestCollectPositions_LeJournalFusionneOuvreLaPasse(t *testing.T) {
	c := &KillSourceCollector{
		caps:          games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported},
		mapNames:      fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Carte Forge Inconnue"}}},
		mapBounds:     testMapQuantCatalog(),
		acquireShared: panicWriter,
	}
	avant := observability.LoadCounter(metricPositionsNoMap)
	fusionne := persist.KillSourceBatch{Deaths: []persist.KillEventInsert{
		{TimeMS: 1000, FeedKillerGamertag: "Fluxnic", FeedKillerXUID: "111", VictimXUID: "222"},
	}}
	c.collectPositions(context.Background(), "m1", nil, nil, MatchIdentities{}, fusionne)
	if got := observability.LoadCounter(metricPositionsNoMap) - avant; got != 1 {
		t.Errorf("%s a bouge de %d, attendu 1 : le journal fusionne porte un couple resolu, la passe "+
			"devait aller jusqu a la carte", metricPositionsNoMap, got)
	}
}
