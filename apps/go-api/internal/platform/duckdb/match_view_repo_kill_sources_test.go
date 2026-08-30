// Package duckdb — match_view_repo_kill_sources_test.go : Q21b (arme + headshot du kill
// feed) contre une VRAIE base DuckDB en mémoire, au SCHÉMA DE PRODUCTION
// (migration.EnsureMatchKillEvents). Même raison que match_view_repo_assist_pairs_test.go :
// la lecture passe par `match_kill_events_latest` (QUALIFY sur la passe de décodage), un
// CREATE TABLE de circonstance ne protégerait rien.
//
// G.1 (2026-08-30) : Q21b lit désormais AUSSI `source_category`, avec la MÊME garde
// d'unanimité que `source_tag` (HAVING count(DISTINCT ...) = 1), appliquée INDÉPENDAMMENT
// aux deux colonnes. Ces tests verrouillent ce comportement — en particulier le cas qui
// n'existait pas avant ce lot : arme unanime, catégorie AMBIGUË.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/migration"
)

// killSourceRow : une ligne de `match_kill_events` telle que le test la pose, avec les deux
// colonnes de la VÉRITÉ SOURCE (Q21b) en plus des colonnes minimales de crédit.
type killSourceRow struct {
	matchID    string
	timeMS     int
	killerXUID string
	sourceTag  *int
	category   *string
}

func newKillSourcesDB(t *testing.T, rows []killSourceRow) *sql.DB {
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
		 feed_killer_xuid, feed_present, assist_known, source_tag, source_category,
		 read_path, read_origin)
		VALUES (?, 'pass-1', 'rev-1', TRUE, ?, 'victime', ?, TRUE, FALSE, ?, ?, ?, ?)`
	for i, r := range rows {
		if _, err := db.Exec(ins, r.matchID, r.timeMS, r.killerXUID, r.sourceTag, r.category,
			killscope.ReadPathFilmWalk, filmCreditOrigin); err != nil {
			t.Fatalf("insert ligne %d: %v", i, err)
		}
	}
	return db
}

// scannedKillSource : ce que le test lit de Q21b, avant le calcul Headshot (fait par le test
// lui-même via killscope.IsHeadshotCategory — le MÊME chemin que le repo, pour vérifier le
// couple requête+prédicat bout en bout sans dupliquer le scan du repo).
type scannedKillSource struct {
	xuid      string
	timeMS    int64
	sourceTag uint32
	headshot  bool
}

func queryKillSources(t *testing.T, db *sql.DB, matchID string) []scannedKillSource {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), Q21bKillSources, matchID)
	if err != nil {
		t.Fatalf("Q21b: %v", err)
	}
	defer rows.Close()
	var out []scannedKillSource
	for rows.Next() {
		var (
			xuid     string
			timeMS   int64
			tag      uint32
			category sql.NullString
		)
		if err := rows.Scan(&xuid, &timeMS, &tag, &category); err != nil {
			t.Fatalf("scan: %v", err)
		}
		s := scannedKillSource{xuid: xuid, timeMS: timeMS, sourceTag: tag}
		if category.Valid {
			s.headshot = killscope.IsHeadshotCategory(category.String)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	return out
}

// strPtr / intPtr existent déjà dans le package (home_repo_cache_challenges_roundtrip_test.go,
// season_pass_repo_helpers.go) : on les réutilise plutôt que d'en redéclarer.

// TestQ21bKillSources_Headshot : le cas nominal — un kill dont la catégorie vaut EXACTEMENT
// "Headshot" ressort avec Headshot=true ; un kill mesuré avec une autre catégorie ressort
// avec Headshot=false (mesuré, jamais absent).
func TestQ21bKillSources_Headshot(t *testing.T) {
	db := newKillSourcesDB(t, []killSourceRow{
		{"m1", 1000, "K1", intPtr(0x11), strPtr("Headshot")},
		{"m1", 2000, "K1", intPtr(0x22), strPtr("SilentMelee")},
	})
	got := queryKillSources(t, db, "m1")
	if len(got) != 2 {
		t.Fatalf("lignes = %+v, attendu 2", got)
	}
	// Pas d'ORDER BY dans Q21b (agrégat groupé) : indexer par time_ms plutôt que supposer
	// l'ordre de retour.
	byTime := map[int64]scannedKillSource{}
	for _, s := range got {
		byTime[s.timeMS] = s
	}
	if !byTime[1000].headshot {
		t.Errorf("kill 1000 : headshot = false, attendu true (catégorie Headshot)")
	}
	if byTime[2000].headshot {
		t.Errorf("kill 2000 : headshot = true, attendu false (catégorie SilentMelee, mesurée mais pas headshot)")
	}
}

// TestQ21bKillSources_HeadshotMultiplierExclu : LE cas que le rapport G.0 interdit
// d'inclure — HeadshotMultiplier n'est PAS un headshot au sens produit (oracle : 84,4 %
// d'accord si inclus, contre 99,3 % avec le filtre strict). Verrouille le filtre STRICT
// jusqu'à la requête SQL, pas seulement le prédicat Go.
func TestQ21bKillSources_HeadshotMultiplierExclu(t *testing.T) {
	// Dérivé de l'énumération du décodeur, jamais un littéral brut ici (le ratchet
	// no_raw_headshot_category_literal_test.go interdit "HeadshotMultiplier" hors de son
	// foyer — killsource EST ce foyer) : la valeur voyage depuis la SEULE source de vérité.
	db := newKillSourcesDB(t, []killSourceRow{
		{"m1", 1000, "K1", intPtr(0x11), strPtr(killsource.CategoryHeadshotMultiplier.Name())},
	})
	got := queryKillSources(t, db, "m1")
	if len(got) != 1 {
		t.Fatalf("lignes = %+v, attendu 1", got)
	}
	if got[0].headshot {
		t.Errorf("HeadshotMultiplier compté comme headshot — filtre STRICT violé (G.0 : 84,4%% " +
			"d'accord oracle si inclus, contre 99,3%% sans)")
	}
}

// TestQ21bKillSources_CategorieAmbigueEcartee : arme UNANIME (même source_tag) mais
// CATÉGORIE différente entre les deux morts du même (tueur, instant) — un double kill où
// l'un est un headshot et l'autre non. AVANT G.1 la ligne serait sortie avec une catégorie
// arbitraire (MIN alphabétique) ; désormais la garde d'unanimité l'écarte ENTIÈREMENT, comme
// le fait déjà `source_tag` pour une arme ambiguë — jamais un headshot qui pourrait être faux.
func TestQ21bKillSources_CategorieAmbigueEcartee(t *testing.T) {
	db := newKillSourcesDB(t, []killSourceRow{
		{"m1", 1000, "K1", intPtr(0x11), strPtr("Headshot")},
		{"m1", 1000, "K1", intPtr(0x11), strPtr("None")}, // même (tueur, instant), même arme, catégorie différente
	})
	got := queryKillSources(t, db, "m1")
	if len(got) != 0 {
		t.Fatalf("lignes = %+v, attendu aucune (catégorie ambiguë doit écarter, pas arbitrer)", got)
	}
}

// TestQ21bKillSources_ArmeAmbigueRestEcartee : non-régression — la garde PRÉ-EXISTANTE sur
// l'arme (source_tag ambigu) doit continuer d'écarter la ligne, catégorie identique ou non.
func TestQ21bKillSources_ArmeAmbigueRestEcartee(t *testing.T) {
	db := newKillSourcesDB(t, []killSourceRow{
		{"m1", 1000, "K1", intPtr(0x11), strPtr("Headshot")},
		{"m1", 1000, "K1", intPtr(0x22), strPtr("Headshot")}, // même clé, arme différente
	})
	got := queryKillSources(t, db, "m1")
	if len(got) != 0 {
		t.Fatalf("lignes = %+v, attendu aucune (arme ambiguë)", got)
	}
}

// TestQ21bKillSources_SourceTagAbsentEcarte : `source_tag` NULL écarte la ligne (comme
// avant G.1) — la garde `source_tag IS NOT NULL` n'a pas bougé.
func TestQ21bKillSources_SourceTagAbsentEcarte(t *testing.T) {
	db := newKillSourcesDB(t, []killSourceRow{
		{"m1", 1000, "K1", nil, nil},
	})
	got := queryKillSources(t, db, "m1")
	if len(got) != 0 {
		t.Fatalf("lignes = %+v, attendu aucune (source_tag NULL)", got)
	}
}
