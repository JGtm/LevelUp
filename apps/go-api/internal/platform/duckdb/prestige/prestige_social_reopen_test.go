//go:build cgo

// Package prestige — prestige_social_reopen_test.go : régression « 500 permanent
// sur les lectures d'escouade » (shared_social.duckdb).
//
// Contexte (V721-08b, jumeau de prestige_metadata_reopen_test.go). Les repos
// cross-joueurs (PrestigeSocialRepo / PrestigeSquadRepo / PrestigeSquadChallengeRepo)
// partagent l'unique handle RW shared_social tenue par le serveur pour toute sa
// durée de vie — base également écrite par le sync engine (post-sync). Tant que
// les lectures mono-ligne passaient par `db.QueryRow` PLAT, une invalidation
// FATAL DuckDB (bug ART) ou une handle périmée (« sql: database is closed »
// après un Close/Reopen concurrent) les rendait définitivement en erreur jusqu'au
// prochain restart. Côté HTTP, l'erreur n'est aucune des sentinelles de
// PrestigeHandler.serviceError → branche `default` = 500 masqué : prestige du
// joueur (GET /prestige/user), détail d'escouade, détail d'un défi d'escouade
// (join / abandon / évaluation).
//
// Simulacre : db.Close() AVANT chaque appel (même procédé que
// prestige_player_reopen_test.go / prestige_metadata_reopen_test.go). Fichier
// réel (t.TempDir via newPrestigeSocialDB) car Reopen doit pouvoir rouvrir le
// DSN — :memory: ne survit pas à un close.
//
// Oracle : sur la handle fermée, chaque méthode doit réussir et rendre la ligne
// seedée. Le code pré-fix (QueryRow plat) remonte « sql: database is closed » →
// chaque appel FAIL. Le code post-fix (*Recovered) PASS.
package prestige

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

const (
	socialReopenSquadID     = "sq_reopen"
	socialReopenChallengeID = "scc_reopen"
	socialReopenUserID      = "u_reopen"
	socialReopenPP          = 125
)

// seedSocialReopenFixture insère un total PP (via EmitEvent), une escouade, un
// défi d'escouade et un participant actif sur la handle ouverte.
func seedSocialReopenFixture(t *testing.T, db *duckdb.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(48 * time.Hour)

	if err := NewPrestigeSocialRepo(db).EmitEvent(ctx, prestige.PrestigeEvent{
		ID: "pe_reopen", UserID: socialReopenUserID, TitleSlug: "halo_infinite",
		SourceType: prestige.SourceChallenge, SourceID: "ch_x",
		PPAmount: socialReopenPP, Tier: prestige.TierHeroic, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed EmitEvent: %v", err)
	}

	squads := NewPrestigeSquadRepo(db)
	if err := squads.Create(ctx, prestige.Squad{
		ID: socialReopenSquadID, Name: "Alpha", CreatedBy: socialReopenUserID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed squad Create: %v", err)
	}

	challenges := NewPrestigeSquadChallengeRepo(db)
	if err := challenges.Create(ctx, prestige.SquadChallenge{
		ID: socialReopenChallengeID, SquadID: socialReopenSquadID, TitleSlug: "halo_infinite",
		Mode: prestige.SquadCollective, EvalType: prestige.EvalThreshold,
		WindowType: prestige.WindowSession, WindowValue: "3", TargetPerMember: 5,
		ExpiresAt: &expires, CreatedBy: socialReopenUserID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed squad challenge Create: %v", err)
	}
	if err := challenges.AddParticipant(ctx, prestige.SquadChallengeParticipant{
		SquadChallengeID: socialReopenChallengeID, UserID: socialReopenUserID,
		ChosenTier: prestige.TierHeroic, DataTier: prestige.DataFull,
		CurrentValue: 0, IsPrivate: false, JoinedAt: now,
	}); err != nil {
		t.Fatalf("seed AddParticipant: %v", err)
	}
}

// TestPrestigeSocialRepos_RecoverAfterHandleClose ferme la handle shared_social
// AVANT chaque appel puis vérifie que les lectures mono-ligne du fichier
// récupèrent (Reopen + retry) au lieu d'échouer. Chaque close est ré-appliqué
// pour prouver la recovery de CHAQUE call site indépendamment.
//
// La 5e lecture d'origine (CountActiveParticipants) a disparu avec la méthode :
// code mort sans appelant de production, supprimée le 2026-07-25 (V721-14a / D-07).
func TestPrestigeSocialRepos_RecoverAfterHandleClose(t *testing.T) {
	db := newPrestigeSocialDB(t)
	seedSocialReopenFixture(t, db)

	social := NewPrestigeSocialRepo(db)
	squads := NewPrestigeSquadRepo(db)
	challenges := NewPrestigeSquadChallengeRepo(db)
	ctx := context.Background()

	// 1. GetUserPrestige — le prestige affiché sur le Home / la page Ascension.
	closeHandle(t, db, "GetUserPrestige")
	up, err := social.GetUserPrestige(ctx, socialReopenUserID, "halo_infinite")
	if err != nil {
		t.Fatalf("GetUserPrestige après close: attendu recovery, got %v", err)
	}
	if up.TotalPP != socialReopenPP {
		t.Fatalf("GetUserPrestige: total_pp=%d want %d", up.TotalPP, socialReopenPP)
	}

	// 2. GetUserPrestigeCrossTitle (agrégat cross-titre).
	closeHandle(t, db, "GetUserPrestigeCrossTitle")
	cross, err := social.GetUserPrestigeCrossTitle(ctx, socialReopenUserID)
	if err != nil {
		t.Fatalf("GetUserPrestigeCrossTitle après close: attendu recovery, got %v", err)
	}
	if cross.TotalPP != socialReopenPP {
		t.Fatalf("GetUserPrestigeCrossTitle: total_pp=%d want %d", cross.TotalPP, socialReopenPP)
	}

	// 3. SquadRepo.Get — le détail d'escouade.
	closeHandle(t, db, "SquadRepo.Get")
	sq, err := squads.Get(ctx, socialReopenSquadID)
	if err != nil {
		t.Fatalf("SquadRepo.Get après close: attendu recovery, got %v", err)
	}
	if sq.ID != socialReopenSquadID || sq.Name != "Alpha" {
		t.Fatalf("SquadRepo.Get: got %q/%q want %s/Alpha", sq.ID, sq.Name, socialReopenSquadID)
	}

	// 4. SquadChallengeRepo.Get — la garde d'appartenance de join / abandon /
	//    évaluation d'un défi d'escouade passe par cette lecture.
	closeHandle(t, db, "SquadChallengeRepo.Get")
	sc, err := challenges.Get(ctx, socialReopenChallengeID)
	if err != nil {
		t.Fatalf("SquadChallengeRepo.Get après close: attendu recovery, got %v", err)
	}
	if sc.SquadID != socialReopenSquadID || sc.Mode != prestige.SquadCollective {
		t.Fatalf("SquadChallengeRepo.Get: got squad=%q mode=%q", sc.SquadID, sc.Mode)
	}
}

// TestPrestigeSocialRepos_NotFoundSurvivesRecovery : la recovery ne doit pas
// transformer un « introuvable » en succès ni en erreur d'infrastructure. Les
// mappings domaine des callers (service : « escouade/défi introuvable » ;
// prestige vide pour un joueur sans PP) doivent survivre au Reopen.
func TestPrestigeSocialRepos_NotFoundSurvivesRecovery(t *testing.T) {
	db := newPrestigeSocialDB(t)
	seedSocialReopenFixture(t, db)

	social := NewPrestigeSocialRepo(db)
	squads := NewPrestigeSquadRepo(db)
	challenges := NewPrestigeSquadChallengeRepo(db)
	ctx := context.Background()

	// Escouade absente → sql.ErrNoRows intact (et surtout PAS ErrDBLocked, qui
	// donnerait un 503 à la place du « introuvable »).
	closeHandle(t, db, "SquadRepo.Get(absent)")
	_, err := squads.Get(ctx, "sq_absent")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("SquadRepo.Get(absent) après close: want sql.ErrNoRows, got %v", err)
	}
	if errors.Is(err, dblease.ErrDBLocked) {
		t.Error("SquadRepo.Get(absent): un not-found ne doit pas devenir ErrDBLocked (503)")
	}

	// Défi absent → sql.ErrNoRows intact.
	closeHandle(t, db, "SquadChallengeRepo.Get(absent)")
	if _, err := challenges.Get(ctx, "scc_absent"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("SquadChallengeRepo.Get(absent) après close: want sql.ErrNoRows, got %v", err)
	}

	// Joueur sans PP → prestige VIDE sans erreur (contrat GetUserPrestige).
	closeHandle(t, db, "GetUserPrestige(absent)")
	empty, err := social.GetUserPrestige(ctx, "u_absent", "halo_infinite")
	if err != nil {
		t.Fatalf("GetUserPrestige(absent) après close: want nil err, got %v", err)
	}
	if empty.TotalPP != 0 || empty.UserID != "u_absent" {
		t.Fatalf("GetUserPrestige(absent): got %+v want prestige vide pour u_absent", empty)
	}
}

// TestTranslateSocialLockErr_MapsFileLockToErrDBLocked verrouille la 2e moitié
// de la chaîne d'erreur : une contention fichier DuckDB (un AUTRE process tient
// shared_social.duckdb, ou Reopen ne peut pas rouvrir le DSN) doit ressortir en
// dblease.ErrDBLocked, seule sentinelle que PrestigeHandler.serviceError mappe
// en 503 + Retry-After (au lieu du 500 masqué de la branche `default`).
//
// La contention réelle exige un 2e process détenteur du fichier : non simulable
// in-process. On teste donc la traduction sur les signatures d'erreur DuckDB
// exactes reconnues par duckdb.IsFileLockError, et on vérifie que le reste passe
// INCHANGÉ (sql.ErrNoRows notamment, dont dépendent les mappings « introuvable »).
// Le maillon final (ErrDBLocked → 503) est couvert par
// api/handlers/prestige_test.go (TestPrestigeHandler_*_DBLocked_Returns503).
func TestTranslateSocialLockErr_MapsFileLockToErrDBLocked(t *testing.T) {
	ctx := context.Background()

	locked := []string{
		"Could not set lock on file \"shared_social.duckdb\"",
		"Conflicting lock is held in levelup-server.exe (PID 1234)",
		"IO Error: File is already open in server.exe (PID 4242)",
	}
	for _, msg := range locked {
		got := translateSocialLockErr(ctx, errors.New(msg))
		if !errors.Is(got, dblease.ErrDBLocked) {
			t.Errorf("translateSocialLockErr(%q) ne wrappe pas ErrDBLocked (→ 500 au lieu de 503), got %v", msg, got)
		}
	}

	// Pass-through : identité des erreurs non-lock préservée.
	if got := translateSocialLockErr(ctx, nil); got != nil {
		t.Errorf("translateSocialLockErr(nil) = %v, want nil", got)
	}
	if got := translateSocialLockErr(ctx, sql.ErrNoRows); !errors.Is(got, sql.ErrNoRows) {
		t.Errorf("translateSocialLockErr(sql.ErrNoRows) doit préserver ErrNoRows, got %v", got)
	}
	if got := translateSocialLockErr(ctx, sql.ErrNoRows); errors.Is(got, dblease.ErrDBLocked) {
		t.Error("translateSocialLockErr(sql.ErrNoRows) ne doit pas devenir ErrDBLocked")
	}
	boom := errors.New("syntax error near SELECT")
	if got := translateSocialLockErr(ctx, boom); got != boom {
		t.Errorf("translateSocialLockErr(err non-lock) = %v, want l'erreur d'origine", got)
	}
}
