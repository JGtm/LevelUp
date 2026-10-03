//go:build integration

// Package killcollector — positions_integration_test.go : LE GATE DE LA CAPTURE, sur un film REEL.
//
// Meme fixture, meme skip, meme raison que collector_test.go (TestKillSourceCollecteFilmReelEt
// RelitParLaVue) : ⚠ LES FILMS NE SONT PAS VERSIONNES (107 Mo). Sans KILLSOURCE_FIXTURES, ce
// test se SKIPPE — la commande exacte est dans collector_test.go.
//
// LA CARTE DU FILM EST SA VRAIE CARTE (2026-09-27) : 9b191a7f a ete joue sur Bazaar
// (`cartes_des_films_integration_test.go`). Le test offrait jusque-la TOUS les noms du catalogue
// comme candidats, et la premiere entree de l iteration d une map Go gagnait — une carte tiree au
// hasard. « 0 ligne de position » reste un SKIP documente plutot qu un echec.
package killcollector

import (
	"context"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/haloclient"
)

// realMapQuantCatalog charge le catalogue de bornes REEL du depot — DONNEE DE REFERENCE
// VERSIONNEE (data/titles/halo_infinite/reference/map_quant_bounds.json, ~22 Ko, commitee), pas
// une sortie de sync/backfill : elle est disponible meme dans un worktree sans data/ de travail.
//
// DELEGUE DEPUIS LE LOT 1.9.2 : `catalogueDeBornesVersionne`
// (positions_decoupage_catalogue_test.go, SANS tag de build) fait exactement cela, et deux
// chargeurs du meme fichier auraient ete deux chemins a tenir d accord.
func realMapQuantCatalog(t *testing.T) *decfilm.MapQuantCatalog {
	t.Helper()
	return catalogueDeBornesVersionne(t)
}

// staticMapNames : port.ReplayMapNameRepo qui rend TOUJOURS la meme liste, sans base — le test
// n a pas besoin de savoir QUEL match on lui demande, il n a qu UN film de fixture.
type staticMapNames struct{ names []string }

func (s staticMapNames) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{Names: s.names}, nil
}

func (s staticMapNames) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{Names: s.names}, nil
}

// TestKillSourcePositionsFilmReelEtRelitParLaVue — meme film et meme gate que
// TestKillSourceCollecteFilmReelEtRelitParLaVue (collector_test.go), positions en plus : le pont
// disque, les quatre lectures hors ligne et le pont slot->xuid s emboitent sur du binaire REEL,
// pas seulement sur des chunks synthetiques.
func TestKillSourcePositionsFilmReelEtRelitParLaVue(t *testing.T) {
	const film = "9b191a7f"
	chunks := chargerFilmDeFixture(t, film)
	cat := realMapQuantCatalog(t)

	db := openSharedTestDB(t)
	client := &fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{film: chunks}}
	caps := games.CapabilityMap{
		games.CapFilmKillSource:    games.CapSupported,
		games.CapFilmKillPositions: games.CapSupported,
	}
	col := NewKillSourceCollector(client, fakeRoster{}, sharedWriter(db), caps, 0).
		WithPositionCapture(cartesDesFixtures(), cat)

	if _, _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("CollectMatch: %v", err)
	}

	var morts, positions int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?)`,
		film, film).Scan(&morts, &positions); err != nil {
		t.Fatalf("select: %v", err)
	}
	t.Logf("film %s : %d morts, %d lignes de position", film, morts, positions)
	if positions == 0 {
		t.Skip("0 ligne de position — la carte de ce film n'est probablement pas au catalogue " +
			"de bornes (carte Forge, ou hors des 79 cartes natives) : cas normal, pas une regression")
	}
	if positions > morts {
		t.Errorf("%d lignes de position pour %d morts : ne peut jamais depasser (au plus une "+
			"ligne par mort resolue)", positions, morts)
	}

	var sansKiller int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest
		WHERE match_id = ? AND (killer_xuid IS NULL OR killer_xuid = '')`, film).Scan(&sansKiller); err != nil {
		t.Fatalf("select killer_xuid: %v", err)
	}
	if sansKiller != 0 {
		t.Errorf("%d ligne(s) sans killer_xuid — la cle fonctionnelle doit toujours etre renseignee", sansKiller)
	}

	var auMoinsUnePosition int
	if err := db.QueryRow(`SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?
		AND (killer_x IS NOT NULL OR victim_x IS NOT NULL)`, film).Scan(&auMoinsUnePosition); err != nil {
		t.Fatalf("select coordonnees: %v", err)
	}
	if auMoinsUnePosition != positions {
		t.Errorf("%d ligne(s) sans AUCUNE coordonnee (ni tueur ni victime) — "+
			"BuildKillPositions ne devait en ecrire aucune", positions-auMoinsUnePosition)
	}

	// Idempotence append-only (ADR 0026) : une 2e passe ne double pas la vue, la table garde les
	// deux (meme propriete que collector_test.go pour match_kill_events).
	if _, _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	var positions2, table2 int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM kill_positions WHERE match_id = ?)`, film, film).
		Scan(&positions2, &table2); err != nil {
		t.Fatalf("select 2e passe: %v", err)
	}
	if positions2 != positions {
		t.Errorf("apres 2 passes, la vue sert %d lignes au lieu de %d — le dedoublonnage par cle "+
			"(match_id, killer_xuid, time_ms) ne tient plus", positions2, positions)
	}
	if table2 <= positions {
		t.Errorf("table = %d lignes apres 2 passes, attendu > %d (append-only : la 1ere passe "+
			"reste physiquement presente)", table2, positions)
	}
}
