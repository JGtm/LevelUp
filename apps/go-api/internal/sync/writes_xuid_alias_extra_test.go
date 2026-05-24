//go:build integration

// Tests pour UpsertXUIDAlias — extension D.14 du plan de tests.
// Couvre les cas manquants au test existant TestUpsertXUIDAlias :
//   - bot normalisation (`bid(3.0)` → BotDisplayName depuis analysis.BotDisplayName)
//   - last_seen update entre 2 upserts
//   - xuid vide / gamertag vide → no-op silencieux
package sync_test

import (
	"database/sql"
	"testing"
	"time"

	intsync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/testutil"
)

func TestUpsertXUIDAlias_BotNormalization(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	// Bot xuid avec un gamertag brut API (probablement vide ou random).
	if err := intsync.UpsertXUIDAlias(t.Context(), db, "bid(3.0)", "RawBotName"); err != nil {
		t.Fatalf("upsert bot: %v", err)
	}

	var stored string
	if err := db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "bid(3.0)").Scan(&stored); err != nil {
		t.Fatalf("query: %v", err)
	}
	// Le gamertag stocke doit etre la version BotDisplayName, pas le brut.
	if stored == "RawBotName" {
		t.Errorf("gamertag = %q, le bot devrait etre normalise (RawBotName est le brut API)", stored)
	}
	// Sentinelle : BotDisplayName de "bid(3.0)" produit typiquement quelque
	// chose comme "343 Bot 3" ou similaire (cf. analysis.BotDisplayName).
	if stored == "" {
		t.Errorf("gamertag bot vide apres normalisation")
	}
	t.Logf("bot normalise : bid(3.0) -> %q", stored)
}

func TestUpsertXUIDAlias_BotNormalizationOverridesEvenIfRawProvided(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	// Test que la normalisation overrride MEME un gamertag legitime fourni.
	if err := intsync.UpsertXUIDAlias(t.Context(), db, "bid(7.0)", "OtherName"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	var stored string
	_ = db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "bid(7.0)").Scan(&stored)
	if stored == "OtherName" {
		t.Error("bot gamertag doit toujours etre normalise, pas le raw API")
	}
}

func TestUpsertXUIDAlias_LastSeenUpdatedOnSecondUpsert(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	if err := intsync.UpsertXUIDAlias(t.Context(), db, "xuid-aaa", "Alice"); err != nil {
		t.Fatalf("1er upsert: %v", err)
	}
	var firstSeen time.Time
	if err := db.QueryRow("SELECT last_seen FROM xuid_aliases WHERE xuid = ?", "xuid-aaa").Scan(&firstSeen); err != nil {
		t.Fatalf("query 1: %v", err)
	}

	// Attendre 50ms pour garantir une difference observable.
	time.Sleep(50 * time.Millisecond)

	if err := intsync.UpsertXUIDAlias(t.Context(), db, "xuid-aaa", "Alice"); err != nil {
		t.Fatalf("2eme upsert: %v", err)
	}
	var secondSeen time.Time
	if err := db.QueryRow("SELECT last_seen FROM xuid_aliases WHERE xuid = ?", "xuid-aaa").Scan(&secondSeen); err != nil {
		t.Fatalf("query 2: %v", err)
	}

	if !secondSeen.After(firstSeen) {
		t.Errorf("last_seen pas mis a jour : 1er=%v 2eme=%v", firstSeen, secondSeen)
	}
}

func TestUpsertXUIDAlias_EmptyXUID_NoOp(t *testing.T) {
	db := testutil.NewInMemoryShared(t)
	if err := intsync.UpsertXUIDAlias(t.Context(), db, "", "SomeName"); err != nil {
		t.Errorf("xuid vide doit etre no-op silencieux, got err: %v", err)
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM xuid_aliases").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("xuid vide ne doit rien inserer, got %d rows", n)
	}
}

func TestUpsertXUIDAlias_EmptyGamertag_NoOp(t *testing.T) {
	db := testutil.NewInMemoryShared(t)
	if err := intsync.UpsertXUIDAlias(t.Context(), db, "xuid-zzz", ""); err != nil {
		t.Errorf("gamertag vide doit etre no-op silencieux, got err: %v", err)
	}
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM xuid_aliases").Scan(&n)
	if n != 0 {
		t.Errorf("gamertag vide ne doit rien inserer, got %d rows", n)
	}
}

func TestUpsertXUIDAlias_NonBotPreservesGamertag(t *testing.T) {
	db := testutil.NewInMemoryShared(t)
	if err := intsync.UpsertXUIDAlias(t.Context(), db, "2533274823110022", "JGtm"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	var stored string
	_ = db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "2533274823110022").Scan(&stored)
	if stored != "JGtm" {
		t.Errorf("gamertag humain doit etre preserve tel quel, got %q want JGtm", stored)
	}
}

// keep sql import alive even if not directly used in other helpers.
var _ = sql.ErrNoRows
