//go:build integration

// Package killcollector — shots_capability_test.go : LA PORTE `CapFilmWeaponShots`, SANS FILM.
//
// POURQUOI CE FICHIER EXISTE (constat J4R-4). La porte des tirs n'était couverte que par
// `TestTirsParArmeSuiventLeurPropreCapability`, qui décode un FILM RÉEL — donc qui se SKIPPE sans
// `KILLSOURCE_FIXTURES`. Or les films ne sont pas versionnés (107 Mo) : sur toute machine sans
// fixtures, et en CI, la porte n'était vérifiée par RIEN, et le skip donnait l'apparence
// contraire. Sixième occurrence du motif « skip = faux vert » dans le chantier.
//
// Ce test tourne PARTOUT où DuckDB tourne. Il n'a pas besoin d'un film parce que la porte ne
// porte pas sur le décodage : elle porte sur l'ÉCRITURE de `match_weapon_shots`. L'instrument est
// celui de `shots_test.go` — des fire-events écrits au bit près, dans un chunk de réplication
// synthétique. Ce que le film réel apporte en plus (la fidélité du format), l'autre test le
// couvre quand les fixtures sont là ; ce que celui-ci apporte, c'est que la porte soit tenue
// TOUS LES JOURS.
package killcollector

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/haloclient"
)

// chunkDeTirsSynthetique : un chunk de RÉPLICATION portant des fire-events lisibles.
//
// Le type compte : `ReplicationChunks` ne retient que `FilmChunkTypeReplicationData`. Un chunk du
// mauvais type produirait zéro tir, et le test passerait alors au vert pour la mauvaise raison —
// « aucune ligne écrite » est justement ce qu'il doit distinguer d'« aucune ligne autorisée ».
func chunkDeTirsSynthetique(t *testing.T) []haloclient.FilmChunk {
	t.Helper()
	a1, a2 := deuxArmes(t)
	var b bitBuf
	ecrireFireEvent(&b, 3, a1)
	ecrireFireEvent(&b, 3, a1)
	ecrireFireEvent(&b, 7, a2)
	return []haloclient.FilmChunk{{
		Index:     0,
		ChunkType: haloclient.FilmChunkTypeReplicationData,
		Data:      b.bytes(),
	}}
}

func compterTirsEnBase(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_weapon_shots`).Scan(&n); err != nil {
		t.Fatalf("count match_weapon_shots: %v", err)
	}
	return n
}

// TestPorteDesTirsSansFixture — la porte, dans LES DEUX SENS.
//
// Un test qui ne vérifierait que le sens fermé passerait au vert sur une porte fermée en
// permanence (rien n'est jamais écrit) ; un test qui ne vérifierait que le sens ouvert passerait
// au vert sur une porte inexistante. Les deux assertions ensemble ne laissent aucune des deux
// pannes s'installer — et inverser la condition dans `collectShots` fait rougir l'une ou l'autre.
func TestPorteDesTirsSansFixture(t *testing.T) {
	chunks := chunkDeTirsSynthetique(t)
	ctx := context.Background()

	ventiler := func(t *testing.T, caps games.CapabilityMap) int {
		t.Helper()
		db := openSharedTestDB(t)
		col := NewKillSourceCollector(&fakeFilmClient{}, fakeRoster{}, sharedWriter(db), caps, 0)
		// `collectShots` est appelée DIRECTEMENT : c'est elle qui porte la garde, et l'atteindre
		// par `CollectMatch` exigerait un kill-feed décodable, donc un film.
		ids, err := fakeRoster{}.IdentitiesForMatch(ctx, "m-tirs")
		if err != nil {
			t.Fatalf("identités: %v", err)
		}
		col.collectShots(ctx, "m-tirs", chunks, ids)
		return compterTirsEnBase(t, db)
	}

	// (1) SANS la capability : rien n'est écrit, et ce n'est pas une erreur — dégradation
	//     gracieuse, la passe des morts reste acquise.
	sansCap := ventiler(t, games.CapabilityMap{games.CapFilmKillSource: games.CapSupported})
	if sansCap != 0 {
		t.Errorf("%d ligne(s) de tirs écrites SANS `film.weapon_shots` — la porte ne garde rien. "+
			"Un titre exposant le kill enrichi se verrait ventiler ses tirs, alors que la "+
			"ventilation a ses propres réserves (Fiesta et BTB non livrables)", sansCap)
	}

	// (2) AVEC la capability : les lignes arrivent. Sans cette moitié, une porte fermée dans les
	//     deux cas passerait pour une porte qui fonctionne.
	avecCap := ventiler(t, games.CapabilityMap{
		games.CapFilmKillSource:  games.CapSupported,
		games.CapFilmWeaponShots: games.CapSupported,
	})
	if avecCap == 0 {
		t.Error("aucune ligne de tirs AVEC `film.weapon_shots` — la porte est fermée dans les " +
			"deux cas, donc la seule façon de ne pas ventiler resterait de ne pas décoder du tout")
	}
}
