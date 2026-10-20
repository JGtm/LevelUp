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
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/migration"
)

// filmCreditOrigin : l'origine `read_origin` d'une ligne produite par le DÉCODEUR DE FILM
// quand le crédit qu'il lit concorde avec celui du titre. Son propriétaire typé est
// `games/halo_infinite/film/internal/facts/killsource.OriginCredit` — un paquet TITLE-SPECIFIC, que
// `platform/duckdb` n'a pas à importer. Elle n'est pas non plus dans `killscope` (qui ne
// porte que les valeurs partagées par les écrivains crédit) ni dans le ratchet J4R-3 :
// même traitement que les dix autres fixtures du dépôt qui l'écrivent en clair
// (migration/, persist/, killcollector/).
const filmCreditOrigin = "credit-concordant"

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
	// killerGT : gamertag du tueur écrit par le film (seule identité d'un tueur BOT).
	killerGT *string
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
		 feed_killer_xuid, feed_killer_gamertag, feed_present, assist_known, assist_gamertag, assist_xuid,
		 killer_damage_pct, assist_damage_pct, read_path, read_origin)
		VALUES (?, 'pass-1', 'rev-1', ?, ?, ?, ?, ?, TRUE, ?, ?, ?, ?, ?, ?, ?)`
	// La PORTÉE des lignes passe par les constantes de `domain/killscope` et jamais par un
	// littéral : c'est sur `read_path` que se décide la préséance entre producteurs, et une
	// copie qui dérive d'un caractère la rend aveugle sans erreur (ratchet J4R-3,
	// `internal/archlint/no_raw_kill_scope_literal_test.go`). Ici la MARCHE du décodeur de
	// film : c'est le seul producteur qui connaisse l'assistant — les producteurs crédit
	// écrivent `OriginCreditOnly`, « le crédit et rien que le crédit ».
	for i, r := range rows {
		if _, err := db.Exec(ins, r.matchID, r.publishable, r.timeMS, r.victim,
			r.killerXUID, r.killerGT, r.assistKnown, r.assistGT, r.assistXUID,
			r.killerPct, r.assistPct, killscope.ReadPathFilmWalk, filmCreditOrigin); err != nil {
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
		{"m1", true, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(39), intPtr(60), nil},
		{"m1", true, 2000, "v2", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(70), intPtr(29), nil},
		// A assiste K2 une fois — pas volée (égalité stricte : 49 == 49 n'est PAS un vol).
		{"m1", true, 3000, "v3", strPtr("K2"), true, strPtr("Alpha"), strPtr("A"), intPtr(49), intPtr(49), nil},
		// B assiste K1 une fois — volée.
		{"m1", true, 4000, "v4", strPtr("K1"), true, strPtr("Bravo"), strPtr("B"), intPtr(20), intPtr(79), nil},
		// Mort mesurée SANS assistant : compte dans la portée, pas dans les paires.
		{"m1", true, 5000, "v5", strPtr("K3"), true, nil, nil, intPtr(100), nil, nil},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")

	if scope.MatchDeaths != 5 || scope.MeasuredDeaths != 5 {
		t.Fatalf("portée = %+v, attendu {5 5}", scope)
	}
	if len(pairs) != 3 {
		t.Fatalf("paires = %d (%+v), attendu 3", len(pairs), pairs)
	}
	// ORDER BY assist_count DESC, assist_gamertag, feed_killer_xuid.
	// Les parts moyennes : A->K1 = ROUND(AVG(60, 29)) = 45 ; A->K2 = 49 ; B->K1 = 79.
	// Comparaison par DeepEqual : AvgAssistPct est un pointeur, `!=` comparerait les
	// ADRESSES et échouerait sur deux pointeurs vers la même valeur.
	want := []domain.MatchAssistPairRaw{
		{AssistXUID: "A", AssistGamertag: "Alpha", KillerXUID: "K1", AssistCount: 2, StolenCount: 1, AvgAssistPct: intPtr(45)},
		{AssistXUID: "A", AssistGamertag: "Alpha", KillerXUID: "K2", AssistCount: 1, StolenCount: 0, AvgAssistPct: intPtr(49)},
		{AssistXUID: "B", AssistGamertag: "Bravo", KillerXUID: "K1", AssistCount: 1, StolenCount: 1, AvgAssistPct: intPtr(79)},
	}
	for i := range want {
		if !reflect.DeepEqual(pairs[i], want[i]) {
			t.Errorf("paire %d = %+v, attendu %+v", i, pairs[i], want[i])
		}
	}
}

// TestQ21dAssistPairs_MesureSansAssistant : le match est mesuré (assist_known) mais aucune
// mort ne porte d'assistant nommé. La requête doit rendre la PORTÉE quand même — c'est
// l'état « mesuré, zéro assistance », qui ne se confond pas avec « non mesuré ».
func TestQ21dAssistPairs_MesureSansAssistant(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("K1"), true, nil, nil, intPtr(100), nil, nil},
		{"m1", true, 2000, "v2", strPtr("K2"), true, nil, nil, intPtr(100), nil, nil},
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
// exactement le couple qui fait retirer le bloc au service plutôt que d'écrire « aucune ».
func TestQ21dAssistPairs_NonMesure(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("K1"), false, nil, nil, nil, nil, nil},
		{"m1", true, 2000, "v2", strPtr("K2"), false, nil, nil, nil, nil, nil},
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
		{"autre", true, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69), nil},
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
		{"m1", false, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69), nil},
		{"m1", true, 2000, "v2", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69), nil},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if scope.MatchDeaths != 2 || scope.MeasuredDeaths != 1 {
		t.Fatalf("portée = %+v, attendu {2 1}", scope)
	}
	if len(pairs) != 1 || pairs[0].AssistCount != 1 || pairs[0].StolenCount != 1 {
		t.Fatalf("paires = %+v, attendu une paire 1/1", pairs)
	}
}

// TestQ21dAssistPairs_MortsPubliables : la portée compte aussi les morts PUBLIABLES, assistance
// connue ou non — le fait « journal des morts publiable » de la Vue match (frags pendant l'effet
// d'un bonus). Un match sans ligne : zéro.
func TestQ21dAssistPairs_MortsPubliables(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", strPtr("K1"), false, nil, nil, nil, nil, nil},
		{"m2", false, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69), nil},
		{"m2", false, 2000, "v2", strPtr("K2"), false, nil, nil, nil, nil, nil},
	})
	if _, scope := queryAssistPairs(t, db, "m1"); scope.MatchDeaths != 1 || scope.MeasuredDeaths != 0 || scope.PublishableDeaths != 1 {
		t.Errorf("m1 portée = %+v, attendu {1 0 1}", scope)
	}
	if _, scope := queryAssistPairs(t, db, "m2"); scope.MatchDeaths != 2 || scope.PublishableDeaths != 0 {
		t.Errorf("m2 portée = %+v, attendu 2 morts dont 0 publiable", scope)
	}
	if _, scope := queryAssistPairs(t, db, "absent"); scope.PublishableDeaths != 0 {
		t.Errorf("match absent : %+v", scope)
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
		{"m1", true, 1000, "v1", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), nil, nil, nil},
		{"m1", true, 2000, "v2", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(50), nil, nil},
		{"m1", true, 3000, "v3", strPtr("K1"), true, strPtr("Alpha"), strPtr("A"), intPtr(100), intPtr(228), nil},
		// B n'a AUCUNE part mesurée sur sa paire : la moyenne doit rester ABSENTE
		// (nil), jamais un « 0 % » fabriqué.
		{"m1", true, 4000, "v4", strPtr("K1"), true, strPtr("Bravo"), strPtr("B"), intPtr(80), nil, nil},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if scope.MeasuredDeaths != 4 {
		t.Fatalf("portée = %+v, attendu 4 morts mesurées", scope)
	}
	if len(pairs) != 2 {
		t.Fatalf("paires = %+v, attendu deux paires", pairs)
	}
	if pairs[0].AssistCount != 3 {
		t.Errorf("assist_count = %d, attendu 3 (aucune mort perdue par une part absente)", pairs[0].AssistCount)
	}
	if pairs[0].StolenCount != 1 {
		t.Errorf("stolen_count = %d, attendu 1 (la seule où 228 > 100)", pairs[0].StolenCount)
	}
	// AVG ignore les NULL : la seule part mesurée (228) EST la moyenne — non plafonnée.
	if pairs[0].AvgAssistPct == nil || *pairs[0].AvgAssistPct != 228 {
		t.Errorf("avg_assist_pct = %v, attendu 228 (AVG sur les seules parts mesurées, sans plafond)", pairs[0].AvgAssistPct)
	}
	if pairs[1].AvgAssistPct != nil {
		t.Errorf("avg_assist_pct de Bravo = %v, attendu nil (aucune part mesurée)", *pairs[1].AvgAssistPct)
	}
}

// TestQ21dAssistPairs_BotsComptes : les BOTS comptent comme les autres acteurs. Un tueur
// bot (xuid NULL, gamertag de film) reçoit sa paire ; un assistant bot (xuid NULL, gamertag
// de film) a la sienne. Deux bots distincts ne fusionnent pas : le groupement porte aussi
// sur les gamertags, et aucun xuid n'est normalisé en chaîne vide au SQL.
func TestQ21dAssistPairs_BotsComptes(t *testing.T) {
	db := newAssistPairsDB(t, []killEventRow{
		{"m1", true, 1000, "v1", nil, true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(69), strPtr("343 Ritzy [bot]")},
		{"m1", true, 2000, "v2", nil, true, strPtr("Alpha"), strPtr("A"), intPtr(30), intPtr(20), strPtr("343 Cosmo [bot]")},
		{"m1", true, 3000, "v3", strPtr("K1"), true, strPtr("343 Oscar [bot]"), nil, intPtr(60), intPtr(40), nil},
	})
	pairs, scope := queryAssistPairs(t, db, "m1")
	if scope.MatchDeaths != 3 || scope.MeasuredDeaths != 3 {
		t.Fatalf("portée = %+v, attendu {3 3}", scope)
	}
	want := []domain.MatchAssistPairRaw{
		{AssistGamertag: "343 Oscar [bot]", KillerXUID: "K1", AssistCount: 1, AvgAssistPct: intPtr(40)},
		{AssistXUID: "A", AssistGamertag: "Alpha", KillerGamertag: "343 Cosmo [bot]", AssistCount: 1, AvgAssistPct: intPtr(20)},
		{AssistXUID: "A", AssistGamertag: "Alpha", KillerGamertag: "343 Ritzy [bot]", AssistCount: 1, StolenCount: 1, AvgAssistPct: intPtr(69)},
	}
	if !reflect.DeepEqual(pairs, want) {
		t.Fatalf("paires = %+v\nattendu %+v", pairs, want)
	}
}
