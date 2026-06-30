package migrations

// order_audit_test.go — garde-fou de complétude d'ordre vu de l'union
// (registre global legacy + steps title-owned). Vit ici car ce package peut
// importer migration ET ses propres Steps() ; le test côté package migration ne
// voit que le registre global (Phase 1.5.1 B).
//
// Bout-en-bout : prouve aussi que les steps title-owned passent réellement par
// le runner (provider → combineSteps → RunSteps) et appliquent leur DDL.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// TestCanonicalCoversGlobalAndTitle : chaque step (global + title) est dans
// migration.canonicalOrder, et canonicalOrder n'a pas d'entrée morte vis-à-vis
// de l'union. Déplacer un step package migration → title ne doit jamais sortir
// son Name de canonicalOrder ni en oublier un nouveau.
func TestCanonicalCoversGlobalAndTitle(t *testing.T) {
	order := migration.CanonicalOrder()
	inOrder := make(map[string]bool, len(order))
	for _, n := range order {
		inOrder[n] = true
	}

	present := make(map[string]bool)
	check := func(name string) {
		present[name] = true
		if !inOrder[name] {
			t.Errorf("step %q absent de migration.canonicalOrder (order.go) — l'ajouter à la bonne position", name)
		}
	}
	for _, m := range migration.All() { // legacy (registre global)
		check(m.Name)
	}
	for _, m := range Steps() { // title-owned
		check(m.Name)
	}
	for _, n := range order { // reverse : pas d'entrée morte
		if !present[n] {
			t.Errorf("canonicalOrder référence %q absent de (global + title) — entrée morte", n)
		}
	}
}

// TestTitleStepsRunEndToEnd : le step title-owned add_pve_schema, fourni via le
// provider, est bien exécuté par RunForDB et crée sa table (preuve bout-en-bout
// de la voie B).
func TestTitleStepsRunEndToEnd(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetSharedPvE); err != nil {
		t.Fatalf("RunForDB(SharedPvE): %v", err)
	}

	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'pve_match_stats'",
	).Scan(&n); err != nil {
		t.Fatalf("query table: %v", err)
	}
	if n != 1 {
		t.Errorf("table pve_match_stats absente après migration title-owned (voie B cassée)")
	}
}

// TestTitleStepsRunEndToEnd_Metadata : la famille prestige metadata title-owned
// (create_prestige_metadata_schema + ALTER challenge_template), fournie via le
// provider, est exécutée par RunForDB(TargetMetadata) et crée ses tables.
// Restaure la couverture du test skip-guardé côté package migration (b10).
func TestTitleStepsRunEndToEnd_Metadata(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}

	for _, table := range []string{
		"challenge_template", "preset_arc", "preset_arc_step",
		"asset_translations", "medal_translations", "milestone_catalog", "playlists_catalog",
	} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table,
		).Scan(&n); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table %s absente après migration metadata title-owned (voie B cassée)", table)
		}
	}

	// Sanity : colonnes ajoutées par les ALTER title-owned (source + tagging).
	for _, col := range []string{"source", "lusr_components", "radar_axes", "is_long_term"} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'challenge_template' AND column_name = ?", col,
		).Scan(&n); err != nil {
			t.Fatalf("query column %s: %v", col, err)
		}
		if n != 1 {
			t.Errorf("challenge_template.%s absente — ALTER title-owned non appliqué", col)
		}
	}

	// Médaille custom Vengeur (9000000001) seedée pour Infinite (native H5) —
	// step seed_custom_vengeur_medal. Sans elle, elle n'apparaît pas dans le
	// catalogue médailles du tab Asset Drawer d'Infinite.
	var nameFR, nameEN string
	var isCustom bool
	if err := db.QueryRow(
		`SELECT name_fr, name_en, is_custom FROM medal_definitions WHERE medal_name_id = 9000000001`,
	).Scan(&nameFR, &nameEN, &isCustom); err != nil {
		t.Fatalf("Vengeur medal absent de medal_definitions: %v", err)
	}
	if nameFR != "Vengeur" || nameEN != "Avenger" || !isCustom {
		t.Errorf("Vengeur medal = {fr:%q en:%q custom:%v}, want {Vengeur Avenger true}", nameFR, nameEN, isCustom)
	}
}

// TestTitleStepsRunEndToEnd_Player : les CONSOMMATEURS player title-owned (b15 :
// perf_chain, psa_checked, fix_career_xp) s'appliquent sur le schéma de base player
// GLOBAL via RunForDB(TargetPlayer) (provider câblé combine global+title). Prouve que
// les ALTER title-owned tournent sur les tables créées par le schéma de base resté
// global, dans le bon ordre (canonicalOrder).
func TestTitleStepsRunEndToEnd_Player(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(Player): %v", err)
	}

	// Tables de base + prestige/progression (title-owned depuis b25) + colonnes additives.
	for _, table := range []string{
		"player_match_enrichment", "career_progression", "match_skill_rank", "sessions",
		"match_citations", "arc", "arc_titles", "challenge", "moment_card", "prestige_telemetry",
		"baseline_state", "improvement_campaign", "streak", "record_history", "milestone_earned",
	} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table,
		).Scan(&n); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table player %s absente", table)
		}
	}
	// Ordre load-bearing : challenge.campaign_id (ALTER create_improvement_campaign_schema)
	// après create_prestige_player_schema (crée challenge).
	var nCamp int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'challenge' AND column_name = 'campaign_id'",
	).Scan(&nCamp); err != nil {
		t.Fatalf("query challenge.campaign_id: %v", err)
	}
	if nCamp != 1 {
		t.Errorf("challenge.campaign_id absente — ordre prestige→campaign cassé")
	}
	for _, col := range []string{"performance_chain", "psa_checked_at"} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'player_match_enrichment' AND column_name = ?", col,
		).Scan(&n); err != nil {
			t.Fatalf("query pme.%s: %v", col, err)
		}
		if n != 1 {
			t.Errorf("player_match_enrichment.%s absente — consommateur title-owned non appliqué", col)
		}
	}
}

// TestTitleStepsRunEndToEnd_Shared : le god-file shared title-owned (34 steps, b23),
// fourni via le provider, est exécuté par RunForDB(TargetShared) et crée les tables de
// base + les vues v6. Restaure la couverture des 4 tests skip-guardés (migration_test.go).
func TestTitleStepsRunEndToEnd_Shared(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	for _, table := range []string{"match_registry", "match_participants", "medals_earned", "xuid_aliases", "weapon_kills"} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table,
		).Scan(&n); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table shared %s absente après migration title-owned", table)
		}
	}
	for _, view := range []string{"v_gamertag_lookup", "v_weapon_kills", "v_match_full", "mv_player_matches"} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_type = 'VIEW'", view,
		).Scan(&n); err != nil {
			t.Fatalf("query view %s: %v", view, err)
		}
		if n != 1 {
			t.Errorf("vue shared %s absente après migration title-owned", view)
		}
	}

	// v_gamertag_lookup résout les bots (BotSQLCase) — régression repair_v_gamertag_lookup.
	if _, err := db.Exec(`
		INSERT INTO match_registry (match_id, start_time) VALUES ('t_bot', '2026-05-16');
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES ('t_bot', 'bid(3.0)', NULL, 0, 2);
	`); err != nil {
		t.Fatalf("seed bot: %v", err)
	}
	var got string
	if err := db.QueryRow(`SELECT gamertag FROM v_gamertag_lookup WHERE xuid = 'bid(3.0)'`).Scan(&got); err != nil {
		t.Fatalf("query v_gamertag_lookup bot: %v", err)
	}
	if got != "343 Ellis" {
		t.Errorf("bot bid(3.0) → %q, want 343 Ellis (BotSQLCase non appliqué)", got)
	}
}

// TestTitleStepsRunEndToEnd_SharedSocial : les racines shared_social title-owned (b24)
// créent les tables média/notifications/prestige + drop_idx_pn_xuid_unread retire l'index
// ART/NULL. Restaure la couverture des tests skip-guardés (migration_test/prestige/notifications).
func TestTitleStepsRunEndToEnd_SharedSocial(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetSharedSocial); err != nil {
		t.Fatalf("RunForDB(SharedSocial): %v", err)
	}

	for _, table := range []string{
		"media_files", "media_match_associations", "match_favorites",
		"player_notifications", "notification_preferences", "player_records",
		"prestige_events", "user_prestige", "squad", "squad_member",
		"squad_challenge", "squad_challenge_participant",
	} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table,
		).Scan(&n); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table shared_social %s absente après migration title-owned", table)
		}
	}

	// drop_idx_pn_xuid_unread : l'index ART/NULL doit être absent ; les 2 autres présents.
	var nUnread int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = 'idx_pn_xuid_unread'`).Scan(&nUnread); err != nil {
		t.Fatalf("query idx_pn_xuid_unread: %v", err)
	}
	if nUnread != 0 {
		t.Errorf("idx_pn_xuid_unread présent — drop_idx_pn_xuid_unread non appliqué")
	}
}
