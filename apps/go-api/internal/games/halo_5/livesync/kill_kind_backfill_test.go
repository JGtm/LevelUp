package livesync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
)

// openKillKindBackfillDB : DuckDB in-memory reproduisant les tables + la vue + la séquence
// touchées par le backfill kill_kind. La définition de v_weapon_kills et de la séquence est
// ALIGNÉE sur la source de vérité (internal/migration/steps_shared_append_only_weapon_kills.go
// + steps_shared_h5_weapon_kill_kind.go) — la vue ne garde que la génération MAX par
// (match_id, xuid). weapon_kills porte les colonnes réelles écrites par persistWeaponKills.
func openKillKindBackfillDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	stmts := []string{
		`CREATE SEQUENCE weapon_kills_generation_seq START 1`,
		`CREATE TABLE weapon_kills (
			match_id VARCHAR, xuid VARCHAR, time_ms INTEGER,
			weapon_id UBIGINT, reconciled_as UBIGINT,
			delta_ms INTEGER, confidence VARCHAR, attribution_path VARCHAR,
			swap_detected BOOLEAN, delayed_damage BOOLEAN, player_index INTEGER,
			generation_id BIGINT DEFAULT 0, written_at TIMESTAMP DEFAULT now(),
			kill_kind VARCHAR)`,
		`CREATE VIEW v_weapon_kills AS
			SELECT * EXCLUDE (rk) FROM (
				SELECT *,
				       COALESCE(reconciled_as, weapon_id) AS effective_weapon_id,
				       DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
				FROM weapon_kills
			) WHERE rk = 1`,
		`CREATE TABLE match_registry (match_id VARCHAR, start_time_utc TIMESTAMP, start_time TIMESTAMP, game_variant_id VARCHAR)`,
		`CREATE TABLE xuid_aliases (gamertag VARCHAR, xuid VARCHAR)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("schema: %v\n%s", err, s)
		}
	}
	return db
}

// kkNextGen alloue une génération via la séquence réelle (comme persistWeaponKills), pour
// que la génération SEED reste STRICTEMENT INFÉRIEURE à celle allouée ensuite par le
// backfill — ordre garanti en prod (chaque collecte a fait avancer la séquence).
func kkNextGen(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var g int64
	if err := db.QueryRow(`SELECT nextval('weapon_kills_generation_seq')`).Scan(&g); err != nil {
		t.Fatalf("nextval: %v", err)
	}
	return g
}

// kkSeedKill insère un weapon_kill legacy (kill_kind NULL sauf si fourni) à une génération donnée.
func kkSeedKill(t *testing.T, db *sql.DB, match, xuid string, timeMs int, weapon uint64, gen int64, kind any) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, confidence, attribution_path,
			swap_detected, delayed_damage, generation_id, kill_kind)
		 VALUES (?, ?, ?, ?, 'native', 'h5_native', FALSE, FALSE, ?, ?)`,
		match, xuid, timeMs, weapon, gen, kind); err != nil {
		t.Fatalf("seed kill (%s,%s,gen%d): %v", match, xuid, gen, err)
	}
}

func kkCount(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v\n%s", err, q)
	}
	return n
}

// killEvent construit un event kill canonique (killer/victim gamertag-keyés, arme stock,
// mécanique) comme le produit mapH5Events.
func killEvent(killer, victim string, timeMs int, weaponID string, kind canonical.KillKind) canonical.MatchEvent {
	return canonical.MatchEvent{
		Type:   canonical.MatchEventKill,
		TimeMs: timeMs,
		Killer: &canonical.PlayerIdentity{Gamertag: killer},
		Victim: &canonical.PlayerIdentity{Gamertag: victim},
		Weapon: &canonical.AssetReference{Kind: "weapon", ID: weaponID},
		Kind:   kind,
	}
}

// TestMatchesMissingKillKind_SelectionAndCampaignExclusion : sélectionne les matchs dont
// la génération courante a kill_kind NULL ; exclut ceux déjà backfillés (kill_kind NOT
// NULL) et les matchs Campagne (game_variant_id masqué). Récents d'abord.
func TestMatchesMissingKillKind_SelectionAndCampaignExclusion(t *testing.T) {
	db := openKillKindBackfillDB(t)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time_utc, game_variant_id) VALUES
		('mNull1', TIMESTAMP '2024-01-01', 'arena-variant'),
		('mNull2', TIMESTAMP '2024-02-01', 'arena-variant'),
		('mDone',  TIMESTAMP '2024-03-01', 'arena-variant'),
		('mCampaign', TIMESTAMP '2024-04-01', '00000003-0000-0010-8000-00aa00389b71')`); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	g := kkNextGen(t, db)
	kkSeedKill(t, db, "mNull1", "x1", 1000, 100, g, nil)     // kill_kind NULL → candidat
	kkSeedKill(t, db, "mNull2", "x1", 1000, 100, g, nil)     // idem
	kkSeedKill(t, db, "mDone", "x1", 1000, 100, g, "weapon") // déjà backfillé → exclu
	kkSeedKill(t, db, "mCampaign", "x1", 1000, 100, g, nil)  // Campagne → exclu par variant

	ids, err := matchesMissingKillKind(ctx, db, halo5.TitleSlug, 0)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids=%v, attendu 2 (mDone + mCampaign exclus)", ids)
	}
	if ids[0] != "mNull2" || ids[1] != "mNull1" { // récents d'abord
		t.Errorf("ordre=%v, attendu [mNull2 mNull1]", ids)
	}
}

// TestRunKillKindBackfill_ReDeriveNewGenerationComplete : re-dérive weapon_kills COMPLET
// (2 couples, tous les kills, avec kill_kind) et l'insère en NOUVELLE génération.
// Prouve l'invariant append-only : ancienne génération intacte en table physique, la vue
// renvoie la nouvelle génération complète avec kill_kind ; anti-perte (ré-insertion de
// TOUS les kills du couple, pas d'un sous-ensemble).
func TestRunKillKindBackfill_ReDeriveNewGenerationComplete(t *testing.T) {
	db := openKillKindBackfillDB(t)
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO match_registry (match_id, start_time_utc, game_variant_id) VALUES
		('m1', TIMESTAMP '2024-01-01', 'arena-variant'),
		('mDone', TIMESTAMP '2024-02-01', 'arena-variant')`); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO xuid_aliases VALUES ('JGtm','xJ'), ('Friend','xF')`); err != nil {
		t.Fatalf("seed aliases: %v", err)
	}

	// m1 : génération legacy (kill_kind NULL) — 3 kills xJ + 2 kills xF.
	gOld := kkNextGen(t, db)
	kkSeedKill(t, db, "m1", "xJ", 1000, 100, gOld, nil)
	kkSeedKill(t, db, "m1", "xJ", 2000, 100, gOld, nil)
	kkSeedKill(t, db, "m1", "xJ", 3000, 200, gOld, nil)
	kkSeedKill(t, db, "m1", "xF", 1500, 300, gOld, nil)
	kkSeedKill(t, db, "m1", "xF", 2500, 300, gOld, nil)
	// mDone : déjà backfillé (kill_kind présent) → doit être ignoré par la sélection.
	gDone := kkNextGen(t, db)
	kkSeedKill(t, db, "mDone", "xJ", 1000, 100, gDone, "weapon")

	// Re-fetch : timeline complète de m1 (mêmes couples/kills, désormais AVEC kill_kind).
	fetch := func(_ context.Context, matchID string) ([]canonical.MatchEvent, error) {
		if matchID != "m1" {
			t.Fatalf("fetch inattendu pour %q (mDone aurait dû être exclu)", matchID)
		}
		return []canonical.MatchEvent{
			killEvent("JGtm", "Foe", 1000, "100", canonical.KillKindWeapon),
			killEvent("JGtm", "Foe", 2000, "100", canonical.KillKindMelee),
			killEvent("JGtm", "Foe", 3000, "200", canonical.KillKindWeapon),
			killEvent("Friend", "Foe", 1500, "300", canonical.KillKindWeapon),
			killEvent("Friend", "Foe", 2500, "300", canonical.KillKindShoulderBash),
		}, nil
	}

	stats, err := RunKillKindBackfill(ctx, db, fetch, halo5.TitleSlug, 0, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.Matches != 1 || stats.Updated != 1 || stats.KillRows != 5 {
		t.Fatalf("stats=%+v, attendu Matches=1 Updated=1 KillRows=5", stats)
	}

	// Table physique : ancienne génération INTACTE (5 lignes gOld) + nouvelle (5) = 10.
	if n := kkCount(t, db, `SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1'`); n != 10 {
		t.Errorf("weapon_kills physique m1 = %d, attendu 10 (append pur, ancienne gen intacte)", n)
	}
	if n := kkCount(t, db, `SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND generation_id=?`, gOld); n != 5 {
		t.Errorf("ancienne génération m1 = %d, attendu 5 (préservée)", n)
	}
	// Vue : nouvelle génération COMPLÈTE, kill_kind partout, par couple.
	if n := kkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1'`); n != 5 {
		t.Errorf("v_weapon_kills m1 = %d, attendu 5 (nouvelle génération complète)", n)
	}
	if n := kkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='xJ'`); n != 3 {
		t.Errorf("v_weapon_kills m1/xJ = %d, attendu 3 (couple complet)", n)
	}
	if n := kkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='xF'`); n != 2 {
		t.Errorf("v_weapon_kills m1/xF = %d, attendu 2 (couple complet)", n)
	}
	if n := kkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND kill_kind IS NULL`); n != 0 {
		t.Errorf("v_weapon_kills m1 kill_kind NULL = %d, attendu 0 (tout re-dérivé)", n)
	}
	// Mécaniques précises re-dérivées (melee + shoulderbash captés).
	if n := kkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND kill_kind='melee'`); n != 1 {
		t.Errorf("v_weapon_kills m1 melee = %d, attendu 1", n)
	}
	if n := kkCount(t, db, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND kill_kind='shoulderbash'`); n != 1 {
		t.Errorf("v_weapon_kills m1 shoulderbash = %d, attendu 1", n)
	}

	// Idempotence : 2e passe → m1 sort de la sélection (kill_kind NOT NULL), rien réécrit.
	stats2, err := RunKillKindBackfill(ctx, db, fetch, halo5.TitleSlug, 0, nil)
	if err != nil {
		t.Fatalf("run2: %v", err)
	}
	if stats2.Matches != 0 || stats2.Updated != 0 {
		t.Fatalf("2e passe stats=%+v, attendu Matches=0 Updated=0 (idempotent)", stats2)
	}
	if n := kkCount(t, db, `SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1'`); n != 10 {
		t.Errorf("weapon_kills physique m1 après 2e passe = %d, attendu 10 (inchangé)", n)
	}
}
