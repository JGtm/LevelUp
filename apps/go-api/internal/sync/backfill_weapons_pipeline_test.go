//go:build integration

package sync

// backfill_weapons_pipeline_test.go — tests bout-à-bout de BackfillWeaponKillsForMatchAll
// et processWeaponKillsInline.
//
// Setup : DB in-memory DuckDB (openWeaponDB), mock HaloClient (weaponTestClient).
// Chunks vides → ScanFireEventsAll renvoie 0 events → tous les kills via formula_a
// (WeaponID=nil, Confidence=none). Les tests vérifient les comptes de lignes
// et les bitmasks, pas les valeurs d'armes.

import (
	"context"
	"fmt"
	"testing"
)

// filmAbsent retourne un client mock sans film (found=false).
func filmAbsent() *weaponTestClient { return &weaponTestClient{filmPresent: false} }

// filmEmpty retourne un client mock avec un chunk vide → 0 fire events.
func filmEmpty() *weaponTestClient {
	return &weaponTestClient{
		filmPresent: true,
		filmChunks:  map[int]FilmChunkData{0: {Data: []byte{}, StartMS: 0, DurationMS: 1000}},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BackfillWeaponKillsForMatchAll
// ─────────────────────────────────────────────────────────────────────────────

// TestBackfillWeaponKillsForMatchAll_NoFilm vérifie que l'absence de film
// → (false, nil) et MBitWeaponKillsNoFilm posé, MBitWeaponKills absent.
func TestBackfillWeaponKillsForMatchAll_NoFilm(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)

	found, err := BackfillWeaponKillsForMatchAll(context.Background(), filmAbsent(), db, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected found=false when film absent")
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKillsNoFilm) == 0 {
		t.Errorf("expected MBitWeaponKillsNoFilm set, got bits=%d", bits)
	}
	if bits&int(MBitWeaponKills) != 0 {
		t.Error("MBitWeaponKills must NOT be set when film absent")
	}
}

// TestBackfillWeaponKillsForMatchAll_FilmPresent_NoKills vérifie l'early-exit
// quand le film est présent mais highlight_events est vide.
//
// Garde-fou anti-perte de données (cf. thought_log 2026-05-09) : on NE
// MARQUE PAS bit21 quand 0 ligne weapon_kills n'a été insérée. Marquer ce
// bit alors que la table est vide a fait croire au scan --weapons que
// ~1010 matchs étaient "déjà traités" et a empêché toute relance, alors que
// les rows existantes avaient été simultanément vidées par DELETE+INSERT
// avec attrs=[]. Le contrat est désormais : bit21 set ⇔ rows présentes.
func TestBackfillWeaponKillsForMatchAll_FilmPresent_NoKills(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)

	found, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true when film present")
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKills) != 0 {
		t.Errorf("expected MBitWeaponKills NOT set (no rows inserted = no done), got bits=%d", bits)
	}

	var killCount int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1'").Scan(&killCount)
	if killCount != 0 {
		t.Errorf("expected 0 weapon_kills rows, got %d", killCount)
	}
}

// TestBackfillWeaponKillsForMatchAll_FilmError vérifie que GetMatchFilm erreur
// → BackfillWeaponKillsForMatchAll retourne (false, non-nil error).
func TestBackfillWeaponKillsForMatchAll_FilmError(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)

	client := &weaponTestClient{filmErr: fmt.Errorf("réseau indisponible")}
	_, err := BackfillWeaponKillsForMatchAll(context.Background(), client, db, "m1")
	if err == nil {
		t.Fatal("expected error from GetMatchFilm failure")
	}
}

// TestBackfillWeaponKillsForMatchAll_SingleParticipant_KillsInserted vérifie
// le pipeline complet : film + kills → weapon_kills insérés + bitmask.
func TestBackfillWeaponKillsForMatchAll_SingleParticipant_KillsInserted(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)

	found, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}

	var killCount int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&killCount)
	if killCount != 2 {
		t.Errorf("expected 2 weapon_kills for xuid1, got %d", killCount)
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKills) == 0 {
		t.Error("expected MBitWeaponKills set")
	}
}

// TestBackfillWeaponKillsForMatchAll_TwoParticipants_SeparatedCorrectly vérifie
// que les kills de xuid1 et xuid2 sont insérés séparément avec les bons comptes.
func TestBackfillWeaponKillsForMatchAll_TwoParticipants_SeparatedCorrectly(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid2', 0, 2)`)
	// xuid1 : 3 kills, xuid2 : 1 kill.
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 3000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 6000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 9000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid2', 'kill', 7000)`)

	found, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}

	var xuid1Count, xuid2Count, total int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&xuid1Count)
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid2'").Scan(&xuid2Count)
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1'").Scan(&total)

	if xuid1Count != 3 {
		t.Errorf("expected 3 weapon_kills for xuid1, got %d", xuid1Count)
	}
	if xuid2Count != 1 {
		t.Errorf("expected 1 weapon_kill for xuid2, got %d", xuid2Count)
	}
	if total != 4 {
		t.Errorf("expected 4 total weapon_kills, got %d", total)
	}
}

// TestBackfillWeaponKillsForMatchAll_ParticipantWithNoKills vérifie qu'un
// participant sans highlight_events reçoit 0 lignes weapon_kills (pas d'erreur).
func TestBackfillWeaponKillsForMatchAll_ParticipantWithNoKills(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid2', 0, 2)`) // pas de kills
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)

	_, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var xuid1Count, xuid2Count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&xuid1Count)
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid2'").Scan(&xuid2Count)

	if xuid1Count != 2 {
		t.Errorf("expected 2 kills for xuid1, got %d", xuid1Count)
	}
	if xuid2Count != 0 {
		t.Errorf("expected 0 kills for xuid2 (no events), got %d", xuid2Count)
	}
}

// TestBackfillWeaponKillsForMatchAll_DuplicateEventsNoDoubleRows — non-régression
// bug 2026-05-09 : doublons dans highlight_events ne doivent pas doubler weapon_kills.
// Avant le fix (GROUP BY manquant), 2 lignes identiques produisaient 2 attributions
// → 2 insertions pour ce kill.
func TestBackfillWeaponKillsForMatchAll_DuplicateEventsNoDoubleRows(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	// Même kill inséré deux fois (simule le bug observé sur match ac7ec523 en prod).
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`) // doublon
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)

	_, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 weapon_kills (not 3), got %d (regression: duplicate events inflating rows)", count)
	}
}

// TestBackfillWeaponKillsForMatchAll_Idempotent vérifie qu'un deuxième appel
// réécrit les lignes sans les doubler (DELETE + re-INSERT idempotent).
func TestBackfillWeaponKillsForMatchAll_Idempotent(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)

	for i := 0; i < 2; i++ {
		if _, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1"); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	// Append-only #23046 (Phase 2) : idempotence LOGIQUE via v_weapon_kills (dernière
	// génération). La table physique croît (chaque run = nouvelle génération), la vue
	// ne retourne que la génération MAX → 2 rows stables.
	var count int
	db.QueryRow("SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 rows (v_weapon_kills dernière génération) after 2 idempotent runs, got %d", count)
	}
}

// TestBackfillWeaponKillsForMatchAll_BitmaskIdempotent vérifie que les OR
// successifs ne créent pas de bits parasites dans backfill_completed.
func TestBackfillWeaponKillsForMatchAll_BitmaskIdempotent(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)

	for i := 0; i < 3; i++ {
		if _, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1"); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKills) == 0 {
		t.Error("expected MBitWeaponKills set")
	}
	extraBits := bits &^ (int(MBitWeaponKills) | int(MBitWeaponKillsNoFilm))
	if extraBits != 0 {
		t.Errorf("unexpected extra bits set: %b", extraBits)
	}
}

// TestBackfillWeaponKillsForMatchAll_MeleeKillInserted vérifie que les melee kills
// (event_type='melee_kill') produisent une ligne weapon_kills sans erreur.
func TestBackfillWeaponKillsForMatchAll_MeleeKillInserted(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'melee_kill', 5000)`)

	_, err := BackfillWeaponKillsForMatchAll(context.Background(), filmEmpty(), db, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='xuid1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 weapon_kill for melee, got %d", count)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// processWeaponKillsInline
// ─────────────────────────────────────────────────────────────────────────────

// TestProcessWeaponKillsInline_NoMatches vérifie le cas trivial : liste vide.
func TestProcessWeaponKillsInline_NoMatches(t *testing.T) {
	db := openWeaponDB(t)
	done, noFilm, err := processWeaponKillsInline(context.Background(), db, filmEmpty(), "xuid1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if done != 0 || noFilm != 0 {
		t.Errorf("expected done=0 noFilm=0, got done=%d noFilm=%d", done, noFilm)
	}
}

// TestProcessWeaponKillsInline_FilmAbsentCountsNoFilm vérifie que les matchs
// sans film incrémentent noFilm, pas done.
func TestProcessWeaponKillsInline_FilmAbsentCountsNoFilm(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m2')`)

	done, noFilm, err := processWeaponKillsInline(context.Background(), db, filmAbsent(), "xuid1", []string{"m1", "m2"})
	if err != nil {
		t.Fatal(err)
	}
	if done != 0 {
		t.Errorf("expected done=0, got %d", done)
	}
	if noFilm != 2 {
		t.Errorf("expected noFilm=2, got %d", noFilm)
	}
}

// TestProcessWeaponKillsInline_FilmPresentCountsDone vérifie que done est
// incrémenté pour les matchs avec film.
func TestProcessWeaponKillsInline_FilmPresentCountsDone(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m1')`)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m2')`)

	done, noFilm, err := processWeaponKillsInline(context.Background(), db, filmEmpty(), "xuid1", []string{"m1", "m2"})
	if err != nil {
		t.Fatal(err)
	}
	if done != 2 {
		t.Errorf("expected done=2, got %d", done)
	}
	if noFilm != 0 {
		t.Errorf("expected noFilm=0, got %d", noFilm)
	}
}

// TestProcessWeaponKillsInline_ContextCancelled vérifie que l'annulation de
// contexte interrompt la boucle et renvoie une erreur non-nil.
func TestProcessWeaponKillsInline_ContextCancelled(t *testing.T) {
	db := openWeaponDB(t)
	for i := 1; i <= 5; i++ {
		db.Exec(fmt.Sprintf("INSERT INTO match_registry (match_id) VALUES ('m%d')", i))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annuler immédiatement avant de lancer la boucle

	matchIDs := []string{"m1", "m2", "m3", "m4", "m5"}
	_, _, err := processWeaponKillsInline(ctx, db, filmEmpty(), "xuid1", matchIDs)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
