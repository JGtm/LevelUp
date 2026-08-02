//go:build integration

// Package migration — migration_test.go : tests d'idempotence des migrations DuckDB.
//
// Sprint 21 — tâche 5 : appliquer les migrations sur DB vierge puis sur DB existante.
// Sprint 47 T18-T20 : vérifier les tables et vues créées par TargetShared/Player/PvE.
//
// Ces tests requièrent CGO (driver DuckDB) et sont marqués "integration".
// Lancer avec : go test -tags=integration ./internal/migration/ -v

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemDB ouvre une DuckDB in-memory pour les tests.
func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openMemDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// countMigrations retourne le nb de lignes dans schema_migrations.
func countMigrations(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : RunForDB(TargetMetadata) idempotent
//
// Les migrations metadata n'ont aucune dépendance externe (CREATE TABLE IF NOT
// EXISTS seulement) → elles tournent sur DB vierge sans erreur.
// ─────────────────────────────────────────────────────────────────────────────

func TestRunForDB_Metadata_IdempotentOnEmptyDB(t *testing.T) {
	db := openMemDB(t)

	metaMigs := ForTarget(TargetMetadata)
	if len(metaMigs) == 0 {
		t.Skip("aucune migration metadata enregistrée")
	}

	// Première passe — DB vierge
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("1ère passe RunForDB(Metadata): %v", err)
	}
	countAfterFirst := countMigrations(t, db)
	if countAfterFirst != len(metaMigs) {
		t.Errorf("après passe 1 : %d migrations appliquées, attendu %d",
			countAfterFirst, len(metaMigs))
	}

	// Deuxième passe — DB déjà migrée (test idempotence)
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("2ème passe RunForDB(Metadata): %v", err)
	}
	countAfterSecond := countMigrations(t, db)
	if countAfterSecond != countAfterFirst {
		t.Errorf("idempotence violée : passe 2 = %d lignes, passe 1 = %d",
			countAfterSecond, countAfterFirst)
	}

	// Troisième passe — confirmation
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("3ème passe RunForDB(Metadata): %v", err)
	}
	countAfterThird := countMigrations(t, db)
	if countAfterThird != countAfterFirst {
		t.Errorf("idempotence violée passe 3 : %d vs %d", countAfterThird, countAfterFirst)
	}

	t.Logf("✅ %d migrations metadata idempotentes (3 passes)", countAfterFirst)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : schema_migrations — pas de doublon possible
// ─────────────────────────────────────────────────────────────────────────────

func TestRunForDB_Metadata_NoDuplicateRows(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}
	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB 2ème: %v", err)
	}

	// Vérifier pas de doublon sur name (PK) — toutes migrations confondues
	var cnt int
	if err := db.QueryRow(
		"SELECT COUNT(DISTINCT name) FROM schema_migrations",
	).Scan(&cnt); err != nil {
		t.Fatalf("count distinct: %v", err)
	}
	total := countMigrations(t, db)
	if total != cnt {
		t.Errorf("doublons détectés : total=%d distinct=%d", total, cnt)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : toutes les migrations metadata marquées schema_done=TRUE après passe
// ─────────────────────────────────────────────────────────────────────────────

func TestRunForDB_Metadata_AllSchemaDone(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetMetadata); err != nil {
		t.Fatalf("RunForDB: %v", err)
	}

	var notDone int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE schema_done = FALSE",
	).Scan(&notDone); err != nil {
		t.Fatalf("query: %v", err)
	}
	if notDone > 0 {
		t.Errorf("%d migration(s) avec schema_done=FALSE après RunForDB", notDone)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : ForTarget filtre correctement par TargetDB
// ─────────────────────────────────────────────────────────────────────────────

func TestForTarget_ReturnsOnlyTargetMigrations(t *testing.T) {
	targets := []TargetDB{TargetMetadata, TargetPlayer, TargetShared, TargetSharedPvE}
	for _, target := range targets {
		migs := ForTarget(target)
		for _, m := range migs {
			if m.TargetDB != target {
				t.Errorf("ForTarget(%s) contient la migration %q de target %s",
					target, m.Name, m.TargetDB)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test : le nombre total de migrations couvertes
// ─────────────────────────────────────────────────────────────────────────────

func TestMigrationCount_MinimumExpected(t *testing.T) {
	// Phase 1.5 (voie B) : les migrations se répartissent désormais entre le registre
	// global All() et les steps title-owned. On mesure le total sur canonicalOrder (liste
	// stable de TOUS les noms, global + title — order_audit garantit qu'elle les couvre).
	all := CanonicalOrder()
	if len(all) < 36 {
		t.Errorf("seulement %d migrations dans canonicalOrder, minimum attendu: 36", len(all))
	}
	t.Logf("total migrations (canonicalOrder, global+title): %d ; registre global All(): %d",
		len(all), len(All()))

	metaCount := len(ForTarget(TargetMetadata))
	playerCount := len(ForTarget(TargetPlayer))
	sharedCount := len(ForTarget(TargetShared))
	pveCount := len(ForTarget(TargetSharedPvE))
	t.Logf("  global par target — metadata: %d, player: %d, shared: %d, pve: %d",
		metaCount, playerCount, sharedCount, pveCount)
}

// ─────────────────────────────────────────────────────────────────────────────
// Sprint 47 T18 — Shared : vérifier tables + vues v6 après migration
// ─────────────────────────────────────────────────────────────────────────────

// viewExists retourne true si la vue est dans information_schema.views.
func assertViewExists(t *testing.T, db *sql.DB, viewName string) bool {
	t.Helper()
	var cnt int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.views WHERE table_name = ?", viewName,
	).Scan(&cnt)
	if err != nil {
		t.Errorf("assertViewExists(%q): %v", viewName, err)
		return false
	}
	return cnt > 0
}

// tableExists retourne true si la table est dans information_schema.tables.
func assertTableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()
	var cnt int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", tableName,
	).Scan(&cnt)
	if err != nil {
		t.Errorf("assertTableExists(%q): %v", tableName, err)
		return false
	}
	return cnt > 0
}

// sharedBaseSchemaIsGlobal indique si create_base_shared_schema est encore enregistré
// dans le registre global (TargetShared). Depuis Phase 1.5 b23, le god-file shared est
// title-owned → ces tests RunForDB(TargetShared) global-only sont skippés (le provider
// title n'est pas câblable ici, cycle). Couverture réelle : TestTitleStepsRunEndToEnd_Shared.
func sharedBaseSchemaIsGlobal() bool {
	for _, m := range ForTarget(TargetShared) {
		if m.Name == "create_base_shared_schema" {
			return true
		}
	}
	return false
}

// sharedSocialBaseIsGlobal : create_base_shared_social_schema encore global ? Depuis b24
// les racines shared_social sont title-owned → tests RunForDB(TargetSharedSocial) global-only
// skippés. Couverture : TestTitleStepsRunEndToEnd_SharedSocial.
func sharedSocialBaseIsGlobal() bool {
	for _, m := range ForTarget(TargetSharedSocial) {
		if m.Name == "create_base_shared_social_schema" {
			return true
		}
	}
	return false
}

// playerBaseSchemaIsGlobal : create_base_player_schema encore global ? Depuis b25 la racine
// player est title-owned → tests RunForDB(TargetPlayer) global-only skippés. Couverture :
// TestTitleStepsRunEndToEnd_Player.
func playerBaseSchemaIsGlobal() bool {
	for _, m := range ForTarget(TargetPlayer) {
		if m.Name == "create_base_player_schema" {
			return true
		}
	}
	return false
}

// TestRunForDB_Shared_CoreTablesExist vérifie les tables centrales de shared.
func TestRunForDB_Shared_CoreTablesExist(t *testing.T) {
	db := openMemDB(t)

	if !sharedBaseSchemaIsGlobal() {
		t.Skip("create_base_shared_schema title-owned (Phase 1.5 b23) — couverture : TestTitleStepsRunEndToEnd_Shared")
	}

	if err := RunForDB(db, TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	tables := []string{
		"match_registry",
		"match_participants",
		"medals_earned",
		"xuid_aliases",
		"weapon_kills",
	}
	for _, tbl := range tables {
		if !assertTableExists(t, db, tbl) {
			t.Errorf("table %q absente après migration shared", tbl)
		}
	}
	t.Logf("✅ %d tables shared vérifiées", len(tables))
}

// TestRunForDB_Shared_V6ViewsExist vérifie les vues garanties en v6.
//
// Sprint 47 T18 en listait QUATRE. Il n'en reste TROIS : `v_killer_victim_full` a été supprimée
// le 2026-08-02 (deux LEFT JOIN morts re-produisant des colonnes déjà présentes, cf.
// steps_shared.go). Le test l'exigeait encore — doublement mort, puisqu'il se skippe aussi sur
// ce chemin depuis Phase 1.5 b23 : il aurait donc échoué le jour où le skip aurait été levé,
// sur une vue que le dépôt a délibérément retirée (constat J4R-6).
func TestRunForDB_Shared_V6ViewsExist(t *testing.T) {
	db := openMemDB(t)

	if !sharedBaseSchemaIsGlobal() {
		t.Skip("create_base_shared_schema title-owned (Phase 1.5 b23) — couverture : TestTitleStepsRunEndToEnd_Shared")
	}

	if err := RunForDB(db, TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	// Vues garanties présentes en v6 (cf. CLAUDE.md + copilot-instructions.md)
	views := []string{
		"v_gamertag_lookup",
		"v_weapon_kills",
	}
	for _, v := range views {
		if !assertViewExists(t, db, v) {
			t.Errorf("vue v6 %q absente après migration shared", v)
		}
	}

	// v_match_full nécessite un ATTACH 'meta' à query-time mais la définition
	// (CREATE VIEW) doit être présente.
	if !assertViewExists(t, db, "v_match_full") {
		t.Error("vue v6 \"v_match_full\" absente après migration shared")
	}

	// Et la vue SUPPRIMÉE ne doit pas revenir : la recréer réintroduirait deux LEFT JOIN
	// exécutés à chaque chargement de vue match, pour des colonnes déjà portées par la table.
	if assertViewExists(t, db, "v_killer_victim_full") {
		t.Error("vue \"v_killer_victim_full\" présente : elle a été supprimée le 2026-08-02, " +
			"son unique lecteur (Q20) lit la table directement")
	}

	t.Logf("✅ 3 vues v6 (v_gamertag_lookup, v_match_full, v_weapon_kills) présentes, " +
		"v_killer_victim_full bien absente")
}

// TestRunForDB_Shared_VGamertagLookup_ResolvesBotNames vérifie que la vue
// v_gamertag_lookup mappe bien les xuid bots 'bid(N.0)' vers leur nom 343 Halo.
//
// Régression : la migration `upgrade_v_gamertag_lookup_bots_and_raw_fallback`
// a été appliquée sur certaines DBs ~6h AVANT que le code de
// applyResolutionViews ne contienne BotSQLCase (commit 4440449d). Résultat :
// schema_migrations marquait done mais la vue restait l'ancienne définition,
// les bots s'affichaient en xuid brut. La migration `repair_v_gamertag_lookup_bots_2026_05_16`
// force le re-déploiement ; ce test prévient toute future divergence définition/code.
func TestRunForDB_Shared_VGamertagLookup_ResolvesBotNames(t *testing.T) {
	db := openMemDB(t)

	if !sharedBaseSchemaIsGlobal() {
		t.Skip("create_base_shared_schema title-owned (Phase 1.5 b23) — couverture : TestTitleStepsRunEndToEnd_Shared")
	}

	if err := RunForDB(db, TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	// Insère un bot avec gamertag NULL en match_participants (cas réel du sync :
	// l'API SPNKr renvoie xuid mais pas de gamertag pour les bots).
	// match_id seed minimal pour respecter le NOT NULL.
	if _, err := db.Exec(`
		INSERT INTO match_registry (match_id, start_time) VALUES ('test_match_bot', '2026-05-16');
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES
			('test_match_bot', 'bid(3.0)',  NULL, 0, 2),
			('test_match_bot', 'bid(15.0)', NULL, 1, 3),
			('test_match_bot', 'bid(999.0)', NULL, 0, 2);
	`); err != nil {
		t.Fatalf("seed match_participants: %v", err)
	}

	cases := []struct {
		xuid     string
		expected string
	}{
		{"bid(3.0)", "343 Ellis"},    // bot connu
		{"bid(15.0)", "343 Mak"},     // bot connu
		{"bid(999.0)", "bid(999.0)"}, // bot inconnu → fallback xuid brut (acceptable)
	}
	for _, c := range cases {
		var got string
		if err := db.QueryRow(`SELECT gamertag FROM v_gamertag_lookup WHERE xuid = ?`, c.xuid).Scan(&got); err != nil {
			t.Errorf("query v_gamertag_lookup pour %s: %v", c.xuid, err)
			continue
		}
		if got != c.expected {
			t.Errorf("xuid %s : got %q, want %q", c.xuid, got, c.expected)
		}
	}
}

// TestRunForDB_Shared_IdempotentOnExistingDB vérifie l'idempotence de TargetShared.
// Sprint 47 T18 — 3 passes, pas de doublons, schema_done = TRUE.
func TestRunForDB_Shared_IdempotentOnExistingDB(t *testing.T) {
	db := openMemDB(t)

	if !sharedBaseSchemaIsGlobal() {
		t.Skip("create_base_shared_schema title-owned (Phase 1.5 b23) — couverture : TestTitleStepsRunEndToEnd_Shared")
	}

	for pass := 1; pass <= 3; pass++ {
		if err := RunForDB(db, TargetShared); err != nil {
			t.Fatalf("RunForDB(Shared) passe %d: %v", pass, err)
		}
	}

	var notDone int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE schema_done = FALSE",
	).Scan(&notDone); err != nil {
		t.Fatalf("query schema_done: %v", err)
	}
	if notDone > 0 {
		t.Errorf("%d migration(s) shared avec schema_done=FALSE après 3 passes", notDone)
	}
	t.Logf("✅ idempotence shared vérifiée (3 passes)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Sprint 47 T19 — Player : vérifier tables après migration
// ─────────────────────────────────────────────────────────────────────────────

// TestRunForDB_Player_CoreTablesExist vérifie les tables player v5.1.
func TestRunForDB_Player_CoreTablesExist(t *testing.T) {
	db := openMemDB(t)

	if !playerBaseSchemaIsGlobal() {
		t.Skip("schéma de base player title-owned (Phase 1.5 b25) — couverture : TestTitleStepsRunEndToEnd_Player")
	}

	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("RunForDB(Player): %v", err)
	}

	tables := []string{
		"player_match_enrichment",
		"sessions",
		"match_citations",
		"career_progression",
		"sync_meta",
		// media_files supprimée par drop_media_from_player_db (migrée vers shared_social)
	}
	for _, tbl := range tables {
		if !assertTableExists(t, db, tbl) {
			t.Errorf("table player %q absente après migration", tbl)
		}
	}
	t.Logf("✅ %d tables player vérifiées", len(tables))
}

// TestRunForDB_Player_IdempotentOnExistingDB vérifie l'idempotence de TargetPlayer.
func TestRunForDB_Player_IdempotentOnExistingDB(t *testing.T) {
	db := openMemDB(t)

	if !playerBaseSchemaIsGlobal() {
		t.Skip("schéma de base player title-owned (Phase 1.5 b25) — couverture : TestTitleStepsRunEndToEnd_Player")
	}

	for pass := 1; pass <= 3; pass++ {
		if err := RunForDB(db, TargetPlayer); err != nil {
			t.Fatalf("RunForDB(Player) passe %d: %v", pass, err)
		}
	}

	var notDone int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE schema_done = FALSE",
	).Scan(&notDone); err != nil {
		t.Fatalf("query schema_done: %v", err)
	}
	if notDone > 0 {
		t.Errorf("%d migration(s) player avec schema_done=FALSE", notDone)
	}
	t.Logf("✅ idempotence player vérifiée (3 passes)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Sprint 47 T20 — SharedPvE : vérifier table pve_match_stats + seed
// ─────────────────────────────────────────────────────────────────────────────

// TestRunForDB_SharedPvE_TableExists vérifie la table Firefight après migration.
func TestRunForDB_SharedPvE_TableExists(t *testing.T) {
	db := openMemDB(t)

	pveMigs := ForTarget(TargetSharedPvE)
	hasCreateMig := false
	for _, m := range pveMigs {
		if m.Name == "add_pve_schema" {
			hasCreateMig = true
			break
		}
	}
	if !hasCreateMig {
		// add_pve_schema est title-owned (halo_infinite/migrations/steps.go) —
		// nécessite SetTitleStepsProvider, non disponible dans ce package (cycle).
		t.Skip("migration add_pve_schema non disponible — title steps provider requis")
	}

	if err := RunForDB(db, TargetSharedPvE); err != nil {
		t.Fatalf("RunForDB(SharedPvE): %v", err)
	}

	if !assertTableExists(t, db, "pve_match_stats") {
		t.Error("table pve_match_stats absente après migration shared_pve")
	}

	// Vérifier les colonnes Firefight attendues (kills par type d'ennemi)
	pveColumns := []string{
		"match_id", "xuid", "waves_completed", "boss_kills",
		"grunt_kills", "elite_kills", "jackal_kills",
	}
	for _, col := range pveColumns {
		var cnt int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'pve_match_stats' AND column_name = ?",
			col,
		).Scan(&cnt)
		if err != nil || cnt == 0 {
			t.Errorf("colonne pve_match_stats.%q absente", col)
		}
	}

	// Seed minimal : insérer une ligne et la relire
	_, err := db.Exec(
		`INSERT INTO pve_match_stats (match_id, xuid, waves_completed, boss_kills,
		 grunt_kills, elite_kills, jackal_kills, brute_kills, hunter_kills)
		 VALUES ('pve-match-001', 'xuid-001', 5, 2, 15, 3, 4, 1, 0)`,
	)
	if err != nil {
		t.Fatalf("seed pve_match_stats: %v", err)
	}

	var waves int
	if err := db.QueryRow("SELECT waves_completed FROM pve_match_stats WHERE match_id = 'pve-match-001'").Scan(&waves); err != nil {
		t.Fatalf("query pve_match_stats: %v", err)
	}
	if waves != 5 {
		t.Errorf("waves_completed attendu 5, obtenu %d", waves)
	}

	t.Logf("✅ pve_match_stats présente et fonctionnelle (seed + query)")
}
