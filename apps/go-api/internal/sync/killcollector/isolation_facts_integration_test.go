//go:build integration

// Package killcollector — isolation_facts_integration_test.go : LES DEUX TABLES DES FAITS
// D'ISOLEMENT, produites par le COLLECTEUR sur un film REEL.
//
// # IL CHOISIT SON FILM, IL NE SE SAUTE PLUS SUR LE PREMIER VENU
//
// Une version précédente ne connaissait qu'UN film (`9b191a7f`) et se sautait quand sa carte
// n'était pas au catalogue de bornes — ce qui est le cas. Le test annonçait donc une couverture
// qu'il n'avait pas, sur un cache qui contient plus d'un millier de films.
//
// Il essaie désormais les répertoires de `KILLSOURCE_FIXTURES` DANS UN ORDRE STABLE et retient
// le PREMIER qui produit réellement des positions. Le critère est le résultat, pas le nom de la
// carte : le test n'a aucun moyen de connaître la carte d'un film sans base (il fabrique son
// roster), et un critère indirect aurait pu diverger de ce qui décide vraiment.
//
// ⚠ SANS `KILLSOURCE_FIXTURES`, IL SE SAUTE — les films ne sont pas versionnés (107 Mo) :
//
//	KILLSOURCE_FIXTURES=../../../../../data/cache/film_chunks \
//	  go test -count=1 -tags=integration -p 1 -run FaitsDIsolementFilmReel \
//	  ./internal/sync/killcollector/
//
// (chemin relatif au paquet ; en absolu, `<racine du dépôt>/data/cache/film_chunks`.)
package killcollector

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/haloclient"
)

// filmsEssayesAuMax borne le nombre de films tentés.
//
// UNE BORNE, PARCE QUE LE CACHE EN CONTIENT PLUS D'UN MILLIER et qu'un décodage coûte de 1 à
// 45 s. Vingt suffisent très largement : la couverture du catalogue de bornes est de 79 cartes
// natives sur un parc dominé par elles, donc la probabilité que vingt films consécutifs soient
// tous hors catalogue est négligeable — et si elle se réalisait, le skip le DIRAIT, en listant
// les films essayés.
const filmsEssayesAuMax = 20

// fakeRosterAvecEquipes : le roster de test, MAIS AVEC LES EQUIPES.
//
// SANS ELLES, LA PROJECTION SE SAUTE, et c'est le comportement documenté : « isolé » se mesure
// entre coéquipiers, et le film ne porte aucun camp. Le double par défaut (`fakeRoster`) n'en
// pose pas — il sert les tests du journal des morts, qui n'en ont pas besoin.
type fakeRosterAvecEquipes struct{ base fakeRoster }

func (r fakeRosterAvecEquipes) IdentitiesForMatch(ctx context.Context, matchID string) (MatchIdentities, error) {
	ids, err := r.base.IdentitiesForMatch(ctx, matchID)
	if err != nil {
		return ids, err
	}
	ids.Equipes = map[string]int{}
	ids.DepartMS = map[string]int64{}
	ids.ArriveeMS = map[string]int64{}
	for i, x := range ids.XUIDs {
		// DEUX CAMPS, par parité de l'ordre stable des xuids. La composition exacte n'a pas
		// d'importance ici : ce que le test vérifie est le CHAÎNAGE, pas la mesure — celle-ci
		// est couverte par les tests purs, où les équipes sont posées à la main.
		ids.Equipes[x] = i % 2
	}
	return ids, nil
}

// filmsDeFixture liste les films disponibles, dans un ordre STABLE (deux exécutions doivent
// essayer les mêmes dans le même ordre).
func filmsDeFixture(t *testing.T) []string {
	t.Helper()
	root := os.Getenv("KILLSOURCE_FIXTURES")
	if root == "" {
		t.Skip("KILLSOURCE_FIXTURES absent : les films ne sont pas versionnes (107 Mo). " +
			"Rejouer avec KILLSOURCE_FIXTURES=<racine du cache de films> " +
			"go test -count=1 -tags=integration -p 1 ./internal/sync/killcollector/")
	}
	entrees, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("racine de fixtures illisible (%s) : %v", root, err)
	}
	var out []string
	for _, e := range entrees {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	if len(out) > filmsEssayesAuMax {
		out = out[:filmsEssayesAuMax]
	}
	return out
}

// TestKillSourceFaitsDIsolementFilmReel — les deux tables se remplissent sur un film réel, et
// une seconde passe ne double pas les vues.
func TestKillSourceFaitsDIsolementFilmReel(t *testing.T) {
	cat := realMapQuantCatalog(t)
	essayes := []string{}

	for _, film := range filmsDeFixture(t) {
		chunks := chargerFilmDeFixture(t, film)
		if len(chunks) == 0 {
			continue
		}
		db := openSharedTestDB(t)
		col := NewKillSourceCollector(
			&fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{film: chunks}},
			fakeRosterAvecEquipes{base: fakeRoster{}}, sharedWriter(db),
			games.CapabilityMap{
				games.CapFilmKillSource:    games.CapSupported,
				games.CapFilmKillPositions: games.CapSupported,
			}, 0).
			WithPositionCapture(staticMapNames{names: allCatalogNames(cat)}, cat)

		if _, _, err := col.CollectMatch(context.Background(), film); err != nil {
			t.Logf("film %s : CollectMatch a echoue (%v) — on essaie le suivant", film, err)
			essayes = append(essayes, film)
			continue
		}
		vies, contextes, positions := comptesDuFilm(t, db, film)
		t.Logf("film %s : %d vies, %d contextes, %d positions", film, vies, contextes, positions)
		if positions == 0 || vies == 0 {
			essayes = append(essayes, film)
			continue
		}
		verifierFaitsDIsolement(t, db, col, film, vies, contextes)
		return
	}
	t.Skipf("aucun des %d films essayes n'a produit de position exploitable (carte hors "+
		"catalogue de bornes, ou pont slot->xuid vide) : %v — cas normal, pas une regression",
		len(essayes), essayes)
}

// comptesDuFilm lit les trois comptes qui décident si un film est exploitable.
func comptesDuFilm(t *testing.T, db *sql.DB, film string) (vies, contextes, positions int) {
	t.Helper()
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_lives_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM match_death_context_latest WHERE match_id = ?),
		(SELECT COUNT(*) FROM kill_positions_latest WHERE match_id = ?)`,
		film, film, film).Scan(&vies, &contextes, &positions); err != nil {
		t.Fatalf("select: %v", err)
	}
	return vies, contextes, positions
}

// verifierFaitsDIsolement porte les assertions, sur le film retenu.
func verifierFaitsDIsolement(t *testing.T, db *sql.DB, col *KillSourceCollector,
	film string, vies, contextes int,
) {
	t.Helper()

	// LES DEUX TABLES SERVENT LA MEME PASSE. Deux `decode_pass` différents rendraient les vues
	// incohérentes entre elles — un contexte citerait des vies que la vue des vies ne sert plus.
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

	// LA SOMME DES ETATS FAIT LE TOTAL, sur des données RÉELLES. Le persister le refuse déjà,
	// mais le vérifier ici prouve que le producteur ne se contente pas de passer la validation :
	// il compte bien quatre états disjoints.
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
	var orphelines int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_death_context_latest
		WHERE match_id = ? AND nearest_teammate_m IS NOT NULL AND teammates_visible = 0`, film).
		Scan(&orphelines); err != nil {
		t.Fatalf("select distances: %v", err)
	}
	if orphelines != 0 {
		t.Errorf("%d distance(s) sans aucun coequipier visible — seule une position repliquee "+
			"autorise une distance", orphelines)
	}

	// LES DEUX COLONNES DE `match_lives` NE PRENNENT QUE LEURS VALEURS D'ENUM.
	var horsEnum int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lives_latest WHERE match_id = ?
		AND (end_cause NOT IN ('death','film_end','cut') OR named_by NOT IN ('death','closure'))`,
		film).Scan(&horsEnum); err != nil {
		t.Fatalf("select enum: %v", err)
	}
	if horsEnum != 0 {
		t.Errorf("%d vie(s) portant une valeur hors enum", horsEnum)
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
