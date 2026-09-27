//go:build integration

package killcollector

// nom_de_remplissage_integration_test.go — LE COLLECTEUR N ECRIT JAMAIS UN NOM DE REMPLISSAGE (lot
// J7.1 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-2).
//
// Le decodeur fabrique des noms `?N` pour rendre la bijection carree ; ils ne designent personne.
// Sur `b1ad85eb` (match a remplacement), `?10` sortait comme assistant NOMME : ecrit en base par
// ce collecteur, accepte par le persister, affiche sur la page du match. Ce test decode le film
// REEL, ecrit par le persister et relit PAR LA VUE `_latest` : aucune colonne de nom — victime,
// tueur, assistant — ne porte un nom qui commence par `?`.
//
// ⚠ LE FILM N EST PAS VERSIONNE : saute sans `KILLSOURCE_FIXTURES` (cf. l en-tete de
// `collector_test.go`). Rejouer :
//
//	KILLSOURCE_FIXTURES=../../../../../data/cache/film_chunks \
//	  go test -tags=integration -p 1 ./internal/sync/killcollector/ -run TestCollecteur_NEcritJamaisUnNomDeRemplissage

import (
	"context"
	"testing"

	"levelup/go-api/internal/sync/haloclient"
)

// TestCollecteur_NEcritJamaisUnNomDeRemplissage — LA CHAINE ENTIERE, sur le film qui publiait `?10`.
func TestCollecteur_NEcritJamaisUnNomDeRemplissage(t *testing.T) {
	const film = "b1ad85eb"  // match a remplacement : le temoin de FK-1 et de FK-2
	const carte = "Domicile" // la carte du match (PLAN_DECODEUR_FILM, table du lot 5.2b.1)
	chunks := chargerFilmDeFixture(t, film)

	db := openSharedTestDB(t)
	client := &fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{film: chunks}}
	col := sousLaCarte(t, NewKillSourceCollector(client, fakeRoster{}, sharedWriter(db), capsAvecFilm(), 0), carte)
	outcome, _, err := col.CollectMatch(context.Background(), film)
	if err != nil {
		t.Fatalf("CollectMatch: %v", err)
	}
	if outcome != OutcomeWritten {
		t.Fatalf("outcome = %q, attendu %q", outcome, OutcomeWritten)
	}

	var morts, remplissage int
	if err := db.QueryRow(`SELECT COUNT(*),
		COUNT(*) FILTER (WHERE starts_with(COALESCE(victim_gamertag, ''), '?')
			OR starts_with(COALESCE(feed_killer_gamertag, ''), '?')
			OR starts_with(COALESCE(assist_gamertag, ''), '?'))
		FROM match_kill_events_latest WHERE match_id = ?`, film).Scan(&morts, &remplissage); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if morts == 0 {
		t.Fatal("la vue ne sert aucune mort — le temoin ne prouve rien")
	}
	if remplissage != 0 {
		t.Errorf("%d ligne(s) sur %d portent un nom de remplissage `?` — un nom qui ne designe "+
			"personne est ecrit en base et affiche sur la page du match", remplissage, morts)
	}
}
