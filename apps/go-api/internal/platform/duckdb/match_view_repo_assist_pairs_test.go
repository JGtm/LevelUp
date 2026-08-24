// Package duckdb — match_view_repo_assist_pairs_test.go : Q21d contre une VRAIE base
// DuckDB en mémoire, avec le SCHÉMA DE PRODUCTION (migration.EnsureMatchKillEvents).
//
// Pourquoi la vraie DDL et pas une table de test : la lecture passe par la vue
// `match_kill_events_latest`, qui porte un QUALIFY sur la passe de décodage et un résidu
// calculé. Un CREATE TABLE de circonstance testerait une requête contre un schéma qui
// n'existe nulle part — exactement le genre de test vert qui ne protège rien.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// killEventRow : une ligne de `match_kill_events` telle que le test la pose. Les champs
// nullables sont des pointeurs pour distinguer « non mesuré » de zéro — c'est le sujet
// même de la table.
type killEventRow struct {
	matchID     string
	publishable bool
	timeMS      int
	victim      string
	killerXUID  *string
	assistKnown bool
	assistGT    *string
	assistXUID  *string
	killerPct   *int
	assistPct   *int
}

// strPtr / intPtr existent déjà dans le package (home_repo_cache_challenges_roundtrip_test.go,
// season_pass_repo_helpers.go) : on les réutilise plutôt que d'en redéclarer.

// newAssistPairsDB ouvre une base DuckDB en mémoire au schéma de production et y insère
// les lignes fournies.
func newAssistPairsDB(t *testing.T, rows []killEventRow) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("sql.Open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.EnsureMatchKillEvents(db); err != nil {
		t.Fatalf("EnsureMatchKillEvents: %v", err)
	}
	const ins = `INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms, victim_gamertag,
		 feed_killer_xuid, feed_present, assist_known, assist_gamertag, assist_xuid,
		 killer_damage_pct, assist_damage_pct, read_path, read_origin)
		VALUES (?, 'pass-1', 'rev-1', ?, ?, ?, ?, TRUE, ?, ?, ?, ?, ?, 'marche', 'credit-concordant')`
	for i, r := range rows {
		if _, err := db.Exec(ins, r.matchID, r.publishable, r.timeMS, r.victim,
			r.killerXUID, r.assistKnown, r.assistGT, r.assistXUID,
			r.killerPct, r.assistPct); err != nil {
			t.Fatalf("insert ligne %d: %v", i, err)
		}
	}
	return db
}

// queryAssistPairs joue Q21d + son scan, comme le fait le lecteur du repo.
func queryAssistPairs(t *testing.T, db *sql.DB, matchID string) ([]domain.MatchAssistPairRaw, domain.MatchAssistScopeRaw) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), Q21dAssistPairs, matchID, matchID)
	if err != nil {
		t.Fatalf("Q21d: %v", err)
	}
	defer rows.Close()
	pairs, scope, err := scanAssistPairs(rows)
	if err != nil {
		t.Fatalf("scanAssistPairs: %v", err)
	}
	return pairs, scope
}

// TestQ21dAssistPairs_Nominal : deux assistants, trois tueurs assistés, et le décompte des
// éliminations volées (part de l'assistant STRICTEMENT supérieure à celle du tueur).
func TestQ21dAssistPairs_Nominal(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		// A assiste K1 deux fois — dont une volée (60 > 39).
		{"m1", true, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(39), intPtr(60)},
		{"m1", true, 2000, "v2", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(70), intPtr(29)},
		// A assiste K2 une fois — pas volée (égalité stricte : 49 == 49 n'est PAS un vol).
		{"m1", true, 3000, "v3", strPtr("K2"), true, strPtr("Alpha"), strPtr("A"), intPtr(49), intPtr(49)},
		// B assiste K1 une fois — volée.
		{"m1", true, 4000, "v4", strPtr("K1"), true, strPtr("Bravo"), strPtr("B"), intPtr(20), intPtr(79)},
		// Mort mesurée SANS assistant : compte dans la portée, pas dans les paires.
		{"m1", true, 5000, "v5", strPtr("K3"), true, nil, nil, intPtr(100), nil},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")

	if scope.MatchDeaths != 5 || scope.MeasuredDeaths != 5 {
		t.Fatalf("portée = %+v, attendu {5 5}", scope)
	}
	if len(pairs) != 3 {
		t.Fatalf("paires = %d (%+v), attendu 3", len(pairs), pairs)
	}
	// ORDER BY assist_count DESC, assist_gamertag, feed_killer_xuid
	want := []domain.MatchAssistPairRaw{
		{AssistXUID: "A", AssistGamertag: "Alpha", KillerXUID: "K1", AssistCount: 2, StolenCount: 1},
		{AssistXUID: "A", AssistGamertag: "Alpha", KillerXUID: "K2", AssistCount: 1, StolenCount: 0},
		{AssistXUID: "B", AssistGamertag: "Bravo", KillerXUID: "K1", AssistCount: 1, StolenCount: 1},
	}
	for i := range want {
		if pairs[i] != want[i] {
			t.Errorf("paire %d = %+v, attendu %+v", i, pairs[i], want[i])
		}
	}
}

// TestQ21dAssistPairs_MesureSansAssistant : le match est mesuré (assist_known) mais aucune
// mort ne porte d'assistant nommé. La requête doit rendre la PORTÉE quand même — c'est
// l'état « mesuré, zéro assistance », qui ne se confond pas avec « non mesuré ».
func TestQ21dAssistPairs_MesureSansAssistant(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("K1"), true, nil, nil, intPtr(100), nil},
		{"m1", true, 2000, "v2", strPtr("K2"), true, nil, nil, intPtr(100), nil},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if len(pairs) != 0 {
		t.Fatalf("paires = %+v, attendu aucune", pairs)
	}
	if scope.MatchDeaths != 2 || scope.MeasuredDeaths != 2 {
		t.Fatalf("portée = %+v, attendu {2 2}", scope)
	}
}

// TestQ21dAssistPairs_NonMesure : le match a un film décodé mais AUCUNE ligne
// `assist_known`. MeasuredDeaths tombe à 0 alors que MatchDeaths reste positif — c'est
// exactement le couple qui permet à l'écran d'écrire « non mesuré » plutôt que « aucune ».
func TestQ21dAssistPairs_NonMesure(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("K1"), false, nil, nil, nil, nil},
		{"m1", true, 2000, "v2", strPtr("K2"), false, nil, nil, nil, nil},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if len(pairs) != 0 {
		t.Fatalf("paires = %+v, attendu aucune", pairs)
	}
	if scope.MatchDeaths != 2 || scope.MeasuredDeaths != 0 {
		t.Fatalf("portée = %+v, attendu {2 0}", scope)
	}
}

// TestQ21dAssistPairs_MatchAbsent : aucun film pour ce match (ou titre sans décodeur).
// La requête rend une ligne de portée à zéro — le service en déduit qu'il n'y a rien à
// publier, et l'écran ne rend rien.
func TestQ21dAssistPairs_MatchAbsent(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"autre", true, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69)},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if len(pairs) != 0 {
		t.Fatalf("paires = %+v, attendu aucune", pairs)
	}
	if scope.MatchDeaths != 0 || scope.MeasuredDeaths != 0 {
		t.Fatalf("portée = %+v, attendu {0 0}", scope)
	}
}

// TestQ21dAssistPairs_NonPubliableEcarte : une passe non publiable ligne à ligne (BTB)
// sort des paires ET de la portée mesurée. Une paire nomme deux joueurs : elle exige la
// même portée que le kill feed.
func TestQ21dAssistPairs_NonPubliableEcarte(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", false, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69)},
		{"m1", true, 2000, "v2", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69)},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if scope.MatchDeaths != 2 || scope.MeasuredDeaths != 1 {
		t.Fatalf("portée = %+v, attendu {2 1}", scope)
	}
	if len(pairs) != 1 || pairs[0].AssistCount != 1 || pairs[0].StolenCount != 1 {
		t.Fatalf("paires = %+v, attendu une paire 1/1", pairs)
	}
}

// TestQ21dAssistPairs_PartsNonMesureesEtNonBornees : deux réserves de la doctrine, dans un
// seul test parce qu'elles portent sur les mêmes deux colonnes.
//
//   - une part MANQUANTE n'est jamais un vol (NULL > x rend NULL, donc faux au FILTER) —
//     et la mort reste comptée dans assist_count : on n'a pas cessé de l'observer ;
//   - une part AU-DELÀ DE 100 (mesures réelles jusqu'à 228) n'est pas plafonnée : la
//     comparaison porte sur l'ordre des deux parts, pas sur leur échelle.
func TestQ21dAssistPairs_PartsNonMesureesEtNonBornees(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), nil, nil},
		{"m1", true, 2000, "v2", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(50), nil},
		{"m1", true, 3000, "v3", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(100), intPtr(228)},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if scope.MeasuredDeaths != 3 {
		t.Fatalf("portée = %+v, attendu 3 morts mesurées", scope)
	}
	if len(pairs) != 1 {
		t.Fatalf("paires = %+v, attendu une paire", pairs)
	}
	if pairs[0].AssistCount != 3 {
		t.Errorf("assist_count = %d, attendu 3 (aucune mort perdue par une part absente)", pairs[0].AssistCount)
	}
	if pairs[0].StolenCount != 1 {
		t.Errorf("stolen_count = %d, attendu 1 (la seule où 228 > 100)", pairs[0].StolenCount)
	}
}

// TestQ21dAssistPairs_TueurBotEcarte : un tueur BOT n'a pas de xuid (`feed_killer_xuid`
// NULL). La paire n'aurait pas de destinataire nommable — elle est écartée au SQL, jamais
// normalisée en chaîne vide (qui fusionnerait tous les bots en un acteur unique).
func TestQ21dAssistPairs_TueurBotEcarte(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", nil, true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69)},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if len(pairs) != 0 {
		t.Fatalf("paires = %+v, attendu aucune", pairs)
	}
	// La mort reste MESURÉE : la portée ne ment pas parce que la paire est inutilisable.
	if scope.MatchDeaths != 1 || scope.MeasuredDeaths != 1 {
		t.Fatalf("portée = %+v, attendu {1 1}", scope)
	}
}
