//go:build cgo

// Package prestige — prestige_writes_checkpoint_test.go (ADR 0022).
//
// Valide que les mutations Prestige sur shared_social (events/user_prestige,
// squad/squad_member, squad_challenge/participant) flushent le WAL via le
// CHECKPOINT immédiat ajouté dans prestige_social_repo.go (helper
// execCheckpointed). Même classe de bug et même oracle que les notifications.
//
// Oracle = taille du fichier .wal (un CHECKPOINT le tronque ; un write non
// flushé le laisse >0 et serait perdu à la quarantaine d'un WAL orphelin #7659).
// walSize() est dupliqué en pied de fichier : depuis l'extraction du sous-package
// prestige (K3a) ce test ne partage plus le package de test de duckdb-root où vit
// l'original (notifications_writes_checkpoint_test.go).
//
// Noms `TestSet...PersistsAfter` → matchent la regex du gate CI.
package prestige

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// walSize retourne la taille du fichier .wal (0 si absent). Oracle de flush.
// Dupliqué de notifications_writes_checkpoint_test.go (K3a — packages de test
// disjoints après extraction du sous-package prestige).
func walSize(t *testing.T, dbPath string) int64 {
	t.Helper()
	info, err := os.Stat(dbPath + ".wal")
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("stat wal: %v", err)
	}
	return info.Size()
}

// newPrestigeSocialDB crée une shared_social.duckdb avec le schéma Prestige
// complet (migrations réelles, donc squad_member rekeyé avec xuid) + un
// CHECKPOINT initial (WAL des migrations vidé).
func newPrestigeSocialDB(t *testing.T) *duckdb.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "shared_social.duckdb")

	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := migration.RunForDB(raw, migration.TargetSharedSocial); err != nil {
		raw.Close()
		t.Fatalf("RunForDB(shared_social): %v", err)
	}
	raw.Close()

	db, err := duckdb.OpenReadWriteShared(path, "")
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint initial: %v", err)
	}
	return db
}

// TestSetPrestigeEvent_PersistsAfterReopen : EmitEvent (INSERT prestige_events +
// UPSERT user_prestige) → WAL flushé (~0) par le CHECKPOINT du bump.
func TestSetPrestigeEvent_PersistsAfterReopen(t *testing.T) {
	db := newPrestigeSocialDB(t)
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	ev := prestige.PrestigeEvent{
		ID: "pe_1", UserID: "u1", TitleSlug: "halo_infinite",
		SourceType: prestige.SourceChallenge, SourceID: "ch_x",
		PPAmount: 125, Tier: prestige.TierHeroic, CreatedAt: now,
	}
	if err := repo.EmitEvent(ctx, ev); err != nil {
		t.Fatalf("EmitEvent: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après EmitEvent (size=%d want 0) — CHECKPOINT manquant", sz)
	}
	// Sanity : le total a bien été bumpé.
	up, err := repo.GetUserPrestige(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("GetUserPrestige: %v", err)
	}
	if up.TotalPP != 125 {
		t.Errorf("total_pp=%d want 125", up.TotalPP)
	}
}

// TestPrestigeEmitEvent_Atomic_RollsBackOnBumpFailure : si la 2e écriture
// (snapshot user_prestige_history) échoue, l'INSERT prestige_events doit être
// rollback (transaction atomique). APPEND-ONLY : EmitEvent fait INSERT…SELECT FROM
// user_prestige_latest → on casse la 2e écriture en supprimant la vue _latest.
func TestPrestigeEmitEvent_Atomic_RollsBackOnBumpFailure(t *testing.T) {
	db := newPrestigeSocialDB(t)
	if _, err := db.SQLDb().Exec("DROP VIEW user_prestige_latest"); err != nil {
		t.Fatalf("drop user_prestige_latest: %v", err)
	}
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	err := repo.EmitEvent(ctx, prestige.PrestigeEvent{
		ID: "pe_atomic", UserID: "u1", TitleSlug: "halo_infinite",
		SourceType: prestige.SourceChallenge, PPAmount: 50, CreatedAt: now,
	})
	if err == nil {
		t.Fatal("EmitEvent devrait échouer (user_prestige absente)")
	}
	var n int
	if err := db.SQLDb().QueryRow(
		"SELECT COUNT(*) FROM prestige_events WHERE id = 'pe_atomic'").Scan(&n); err != nil {
		t.Fatalf("count prestige_events: %v", err)
	}
	if n != 0 {
		t.Errorf("prestige_events non rollback (n=%d want 0) — EmitEvent pas atomique", n)
	}
}

// TestSetPrestigeUserPrestige_PersistsAfterReopen : UpsertUserPrestige → WAL flushé.
func TestSetPrestigeUserPrestige_PersistsAfterReopen(t *testing.T) {
	db := newPrestigeSocialDB(t)
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.UpsertUserPrestige(ctx, prestige.UserPrestige{
		UserID: "u1", TitleSlug: "halo_infinite", TotalPP: 1000, CurrentLevel: 3, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertUserPrestige: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après UpsertUserPrestige (size=%d want 0)", sz)
	}
}

// TestSetPrestigeSquad_PersistsAfterReopen : Create squad + AddMember + RemoveMember
// → WAL flushé après chaque mutation.
func TestSetPrestigeSquad_PersistsAfterReopen(t *testing.T) {
	db := newPrestigeSocialDB(t)
	repo := NewPrestigeSquadRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Create(ctx, prestige.Squad{ID: "sq1", Name: "Alpha", CreatedBy: "u1", CreatedAt: now}); err != nil {
		t.Fatalf("Create squad: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après Create squad (size=%d want 0)", sz)
	}
	if err := repo.AddMember(ctx, prestige.SquadMember{SquadID: "sq1", Xuid: "xuid-1", UserID: "u1", JoinedAt: now}); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après AddMember (size=%d want 0)", sz)
	}
	if err := repo.RemoveMember(ctx, "sq1", "xuid-1"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après RemoveMember (size=%d want 0)", sz)
	}
}

// TestSetPrestigeSquadChallenge_PersistsAfterReopen : Create challenge +
// AddParticipant + UpdateParticipantProgress → WAL flushé.
func TestSetPrestigeSquadChallenge_PersistsAfterReopen(t *testing.T) {
	db := newPrestigeSocialDB(t)
	repo := NewPrestigeSquadChallengeRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	expires := now.Add(48 * time.Hour)
	sc := prestige.SquadChallenge{
		ID: "scc1", SquadID: "sq1", TitleSlug: "halo_infinite",
		Mode: prestige.SquadCollective, EvalType: prestige.EvalThreshold,
		WindowType: prestige.WindowSession, WindowValue: "3", TargetPerMember: 5,
		ExpiresAt: &expires, CreatedBy: "u1", CreatedAt: now,
	}
	if err := repo.Create(ctx, sc); err != nil {
		t.Fatalf("Create challenge: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après Create challenge (size=%d want 0)", sz)
	}
	if err := repo.AddParticipant(ctx, prestige.SquadChallengeParticipant{
		SquadChallengeID: "scc1", UserID: "u1", ChosenTier: prestige.TierHeroic,
		DataTier: prestige.DataFull, CurrentValue: 0, IsPrivate: false, JoinedAt: now,
	}); err != nil {
		t.Fatalf("AddParticipant: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après AddParticipant (size=%d want 0)", sz)
	}
	if err := repo.UpdateParticipantProgress(ctx, "scc1", "u1", 3.0, nil); err != nil {
		t.Fatalf("UpdateParticipantProgress: %v", err)
	}
	if sz := walSize(t, db.Path()); sz != 0 {
		t.Errorf("WAL non flushé après UpdateParticipantProgress (size=%d want 0)", sz)
	}
}
