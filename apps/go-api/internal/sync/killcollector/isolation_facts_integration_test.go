//go:build integration

// Package killcollector — isolation_facts_integration_test.go : LES DEUX TABLES DES FAITS
// D'ISOLEMENT, produites par le COLLECTEUR sur un film REEL.
//
// ⚠ MEME FIXTURE ET MEME SKIP que positions_integration_test.go : les films ne sont pas
// versionnes (107 Mo). Sans KILLSOURCE_FIXTURES, ce test se SKIPPE. La variable pointe la RACINE
// du cache de films ; le test lit `<racine>/9b191a7f/chunk_*.bin` :
//
//	KILLSOURCE_FIXTURES=../../../../../data/cache/film_chunks //	  go test -count=1 -tags=integration -p 1 -run FaitsDIsolementFilmReel //	  ./internal/sync/killcollector/
//
// (chemin relatif au paquet ; en absolu, `<racine du depot>/data/cache/film_chunks`.) Ce que ce fichier ajoute a la couverture PURE (`replay/death_context_test.go`,
// les quatre etats) et a la couverture SCHEMA (`persist/lives_persister_integration_test.go`,
// l'idempotence de la passe), c'est le CHAINAGE : que la passe de positions remonte bien son
// materiau, que les equipes de la base y arrivent, et que les deux tables se remplissent
// ensemble.
package killcollector

import (
	"context"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/haloclient"
)

// fakeRosterAvecEquipes : le roster de test, MAIS AVEC LES EQUIPES.
//
// SANS ELLES, LA PROJECTION SE SAUTE, et c'est le comportement documente : « isole » se mesure
// entre coequipiers, et le film ne porte aucun camp. Le double par defaut (`fakeRoster`) n'en
// pose pas — il sert les tests du journal des morts, qui n'en ont pas besoin.
type fakeRosterAvecEquipes struct {
	base fakeRoster
}

func (r fakeRosterAvecEquipes) IdentitiesForMatch(ctx context.Context, matchID string) (MatchIdentities, error) {
	ids, err := r.base.IdentitiesForMatch(ctx, matchID)
	if err != nil {
		return ids, err
	}
	ids.Equipes = map[string]int{}
	ids.DepartMS = map[string]int64{}
	for i, x := range ids.XUIDs {
		// DEUX CAMPS, par parite de l'ordre stable des xuids. La composition exacte n'a pas
		// d'importance ici : ce que le test verifie est le CHAINAGE, pas la mesure — celle-ci
		// est couverte par les tests purs, ou les equipes sont posees a la main.
		ids.Equipes[x] = i % 2
	}
	return ids, nil
}

// TestKillSourceFaitsDIsolementFilmReel — les deux tables se remplissent, et une seconde passe
// ne double pas les vues.
func TestKillSourceFaitsDIsolementFilmReel(t *testing.T) {
	const film = "9b191a7f"
	chunks := chargerFilmDeFixture(t, film)
	cat := realMapQuantCatalog(t)

	db := openSharedTestDB(t)
	client := &fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{film: chunks}}
	caps := games.CapabilityMap{
		games.CapFilmKillSource:    games.CapSupported,
		games.CapFilmKillPositions: games.CapSupported,
	}
	col := NewKillSourceCollector(client, fakeRosterAvecEquipes{base: fakeRoster{}},
		sharedWriter(db), caps, 0).
		WithPositionCapture(staticMapNames{names: allCatalogNames(cat)}, cat)

	if _, _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("CollectMatch: %v", err)
	}

	var vies, contextes, positions int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_lives_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM match_death_context_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?)`,
		film, film, film).Scan(&vies, &contextes, &positions); err != nil {
		t.Fatalf("select: %v", err)
	}
	t.Logf("film %s : %d vies, %d contextes, %d positions", film, vies, contextes, positions)
	if positions == 0 {
		t.Skip("0 ligne de position — la carte de ce film n'est probablement pas au catalogue " +
			"de bornes : cas normal, pas une regression (cf. positions_integration_test.go)")
	}
	if vies == 0 {
		t.Fatal("0 vie ecrite alors que les positions le sont : la projection des faits " +
			"d'isolement ne recoit pas le materiau de la passe")
	}

	// LES DEUX TABLES SERVENT LA MEME PASSE. Deux `decode_pass` differents rendraient les vues
	// incoherentes entre elles — un contexte citerait des vies que la vue des vies ne sert plus.
	if contextes > 0 {
		var passes int
		if err := db.QueryRow(`SELECT COUNT(DISTINCT p) FROM (
			SELECT decode_pass AS p FROM match_lives_latest WHERE match_id = ?
			UNION SELECT decode_pass FROM match_death_context_latest WHERE match_id = ?)`,
			film, film).Scan(&passes); err != nil {
			t.Fatalf("select passes: %v", err)
		}
		if passes != 1 {
			t.Errorf("%d passes distinctes entre les deux vues, attendu 1", passes)
		}
	}

	// LA SOMME DES ETATS FAIT LE TOTAL, sur des donnees REELLES. Le persister le refuse deja,
	// mais le verifier ici prouve que le producteur ne se contente pas de passer la validation :
	// il compte bien quatre etats disjoints.
	var incoherents int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_death_context_latest
		WHERE match_id = ? AND teammates_visible + teammates_waiting
		    + teammates_out_of_sight + teammates_left <> teammates_total`, film).
		Scan(&incoherents); err != nil {
		t.Fatalf("select somme: %v", err)
	}
	if incoherents != 0 {
		t.Errorf("%d contexte(s) dont la somme des etats ne fait pas le total", incoherents)
	}

	// UNE DISTANCE N'EXISTE QUE S'IL Y A UN COEQUIPIER VISIBLE.
	var distancesOrphelines int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_death_context_latest
		WHERE match_id = ? AND nearest_teammate_m IS NOT NULL AND teammates_visible = 0`, film).
		Scan(&distancesOrphelines); err != nil {
		t.Fatalf("select distances: %v", err)
	}
	if distancesOrphelines != 0 {
		t.Errorf("%d distance(s) sans aucun coequipier visible — seule une position repliquee "+
			"autorise une distance", distancesOrphelines)
	}

	// IDEMPOTENCE DE LA PASSE (ADR 0026) : la vue sert la DERNIERE passe, la table garde tout.
	if _, _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	var vies2, brut2 int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_lives_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM match_lives WHERE match_id = ?)`, film, film).
		Scan(&vies2, &brut2); err != nil {
		t.Fatalf("select 2e passe: %v", err)
	}
	if vies2 != vies {
		t.Errorf("apres 2 passes, la vue sert %d vies au lieu de %d — elle melange deux passes",
			vies2, vies)
	}
	if brut2 <= vies {
		t.Errorf("table = %d lignes apres 2 passes, attendu > %d : append-only, la 1ere passe "+
			"reste physiquement presente", brut2, vies)
	}
}
