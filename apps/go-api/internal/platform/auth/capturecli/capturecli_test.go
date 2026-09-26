// Package capturecli — capturecli_test.go : tests unitaires des helpers CLI.
// Pas de dépendance cgo (DuckDB) — capturecli opère sur []domain.PlayerSummary
// fourni par le caller.
package capturecli

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// trackingInvalidator est un CacheInvalidator de test qui enregistre les xuid
// invalidés (thread-safe).
type trackingInvalidator struct {
	mu     sync.Mutex
	called []string
}

func (t *trackingInvalidator) invalidate(xuid string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.called = append(t.called, xuid)
}

func (t *trackingInvalidator) snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.called))
	copy(out, t.called)
	return out
}

// playerSummaries construit une slice [domain.PlayerSummary] depuis un map
// gamertag → xuid pour les tests.
func playerSummaries(entries map[string]string) []domain.PlayerSummary {
	out := make([]domain.PlayerSummary, 0, len(entries))
	for gt, xuid := range entries {
		out = append(out, domain.PlayerSummary{
			PlayerSlug: gt,
			Gamertag:   gt,
			XUID:       xuid,
		})
	}
	return out
}

// ─── ResolveXUIDByGamertag ────────────────────────────────────────────────

func TestResolveXUIDByGamertag_ExactMatch(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "2533274858283686",
		"JGtm":        "2533274823110022",
	})

	xuid, canonical, err := ResolveXUIDByGamertag(players, "Madina97294")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if xuid != "2533274858283686" {
		t.Errorf("xuid = %q, want 2533274858283686", xuid)
	}
	if canonical != "Madina97294" {
		t.Errorf("canonical = %q, want Madina97294", canonical)
	}
}

func TestResolveXUIDByGamertag_CaseInsensitive(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "2533274858283686",
	})

	xuid, canonical, err := ResolveXUIDByGamertag(players, "madina97294")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if xuid != "2533274858283686" {
		t.Errorf("xuid = %q", xuid)
	}
	if canonical != "Madina97294" {
		t.Errorf("canonical = %q, want Madina97294 (casse db_profiles)", canonical)
	}
}

func TestResolveXUIDByGamertag_TrimsWhitespace(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "2533274858283686",
	})

	xuid, _, err := ResolveXUIDByGamertag(players, "  Madina97294  ")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if xuid != "2533274858283686" {
		t.Errorf("xuid = %q (trim attendu)", xuid)
	}
}

func TestResolveXUIDByGamertag_EmptyGamertag(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "2533274858283686",
	})

	if _, _, err := ResolveXUIDByGamertag(players, ""); err == nil {
		t.Error("gamertag vide → erreur attendue")
	}
	if _, _, err := ResolveXUIDByGamertag(players, "   "); err == nil {
		t.Error("gamertag whitespace-only → erreur attendue")
	}
}

func TestResolveXUIDByGamertag_NilPlayersList(t *testing.T) {
	if _, _, err := ResolveXUIDByGamertag(nil, "Madina97294"); err == nil {
		t.Error("players nil → erreur attendue")
	}
}

func TestResolveXUIDByGamertag_EmptyPlayersList(t *testing.T) {
	if _, _, err := ResolveXUIDByGamertag([]domain.PlayerSummary{}, "Madina97294"); err == nil {
		t.Error("players vide → erreur attendue")
	}
}

func TestResolveXUIDByGamertag_PlayerNotFound(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "2533274858283686",
	})

	_, _, err := ResolveXUIDByGamertag(players, "UnknownPlayer")
	if err == nil {
		t.Fatal("joueur absent → erreur attendue")
	}
	if !strings.Contains(err.Error(), "absent de db_profiles.json") {
		t.Errorf("err = %v, message doit mentionner 'absent de db_profiles.json'", err)
	}
}

func TestResolveXUIDByGamertag_PlayerWithEmptyXUID(t *testing.T) {
	players := []domain.PlayerSummary{
		{PlayerSlug: "BadPlayer", Gamertag: "BadPlayer", XUID: ""},
	}

	_, _, err := ResolveXUIDByGamertag(players, "BadPlayer")
	if err == nil {
		t.Fatal("xuid vide → erreur attendue")
	}
	if !strings.Contains(err.Error(), "xuid manquant") {
		t.Errorf("err = %v, message doit mentionner 'xuid manquant'", err)
	}
}

// ─── ParseRefreshTokenStdin ───────────────────────────────────────────────

func TestParseRefreshTokenStdin_PlainToken(t *testing.T) {
	r := strings.NewReader("M.C544_SN1.0.U.-Ct4xy\n")
	got, err := ParseRefreshTokenStdin(r)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "M.C544_SN1.0.U.-Ct4xy" {
		t.Errorf("got = %q", got)
	}
}

func TestParseRefreshTokenStdin_PlainTokenNoNewline(t *testing.T) {
	r := strings.NewReader("rt_no_newline")
	got, err := ParseRefreshTokenStdin(r)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "rt_no_newline" {
		t.Errorf("got = %q", got)
	}
}

func TestParseRefreshTokenStdin_EnvVarFormat(t *testing.T) {
	r := strings.NewReader("SPNKR_OAUTH_REFRESH_TOKEN_MADINA97294=M.C544_value\n")
	got, err := ParseRefreshTokenStdin(r)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "M.C544_value" {
		t.Errorf("got = %q, want M.C544_value (partie droite du =)", got)
	}
}

func TestParseRefreshTokenStdin_EnvVarTrimsValue(t *testing.T) {
	r := strings.NewReader("SPNKR_OAUTH_REFRESH_TOKEN_X=   value_with_spaces   \n")
	got, err := ParseRefreshTokenStdin(r)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "value_with_spaces" {
		t.Errorf("got = %q (trim attendu autour du =)", got)
	}
}

func TestParseRefreshTokenStdin_SkipsCommentsAndEmpty(t *testing.T) {
	input := `
# This is a comment
# Another comment

actual_token_value
ignored_after_first
`
	got, err := ParseRefreshTokenStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "actual_token_value" {
		t.Errorf("got = %q, want actual_token_value (1ère ligne non vide non-commentaire)", got)
	}
}

func TestParseRefreshTokenStdin_OnlyComments(t *testing.T) {
	input := "# comment 1\n# comment 2\n   \n"
	_, err := ParseRefreshTokenStdin(strings.NewReader(input))
	if err == nil {
		t.Error("uniquement commentaires/vides → erreur attendue")
	}
}

func TestParseRefreshTokenStdin_EmptyReader(t *testing.T) {
	_, err := ParseRefreshTokenStdin(strings.NewReader(""))
	if err == nil {
		t.Error("reader vide → erreur attendue")
	}
}

func TestParseRefreshTokenStdin_NilReader(t *testing.T) {
	_, err := ParseRefreshTokenStdin(nil)
	if err == nil {
		t.Error("reader nil → erreur attendue")
	}
}

func TestParseRefreshTokenStdin_EnvVarWithEqualsInValue(t *testing.T) {
	r := strings.NewReader("SPNKR_OAUTH_REFRESH_TOKEN_X=value=with=equals\n")
	got, err := ParseRefreshTokenStdin(r)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "value=with=equals" {
		t.Errorf("got = %q, want value=with=equals (split only first =)", got)
	}
}

func TestParseRefreshTokenStdin_NonEnvLineWithEquals(t *testing.T) {
	r := strings.NewReader("rt=part_of_token\n")
	got, err := ParseRefreshTokenStdin(r)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "rt=part_of_token" {
		t.Errorf("got = %q, want rt=part_of_token (pas de split env-var)", got)
	}
}

type errorReader struct{ err error }

func (e *errorReader) Read(_ []byte) (int, error) { return 0, e.err }

func TestParseRefreshTokenStdin_IOError(t *testing.T) {
	r := &errorReader{err: io.ErrUnexpectedEOF}
	_, err := ParseRefreshTokenStdin(r)
	if err == nil {
		t.Error("I/O error → erreur attendue")
	}
}

// ─── PersistRefreshToken ──────────────────────────────────────────────────

func tempStore(t *testing.T) *auth.MultiUserTokenStore {
	t.Helper()
	return auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
}

func TestPersistRefreshToken_NewEntry(t *testing.T) {
	store := tempStore(t)
	inv := &trackingInvalidator{}

	if err := PersistRefreshToken(store, "2533274858283686", "Madina97294", "rt-fresh", inv.invalidate); err != nil {
		t.Fatalf("err = %v", err)
	}

	user, err := store.Load("2533274858283686")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if user.OAuthRefreshToken != "rt-fresh" {
		t.Errorf("RT = %q", user.OAuthRefreshToken)
	}
	if user.Gamertag != "Madina97294" {
		t.Errorf("Gamertag = %q (devrait être complété)", user.Gamertag)
	}
	if calls := inv.snapshot(); len(calls) != 1 || calls[0] != "2533274858283686" {
		t.Errorf("invalidator calls = %v, want [2533274858283686]", calls)
	}
}

func TestPersistRefreshToken_ExistingEntryPreservesFields(t *testing.T) {
	store := tempStore(t)
	if err := store.Upsert(&auth.UserTokens{
		XUID:         "2533274858283686",
		Gamertag:     "Madina97294",
		XSTSToken:    "xsts-original",
		XSTSUserHash: "uhs-original",
	}); err != nil {
		t.Fatalf("Upsert pré-existant: %v", err)
	}

	if err := PersistRefreshToken(store, "2533274858283686", "Madina97294", "rt-new", nil); err != nil {
		t.Fatalf("err = %v", err)
	}

	user, _ := store.Load("2533274858283686")
	if user.OAuthRefreshToken != "rt-new" {
		t.Errorf("RT = %q (devrait être updated)", user.OAuthRefreshToken)
	}
	if user.XSTSToken != "xsts-original" {
		t.Errorf("XSTS écrasé : %q", user.XSTSToken)
	}
	if user.XSTSUserHash != "uhs-original" {
		t.Errorf("XSTSUserHash écrasé : %q", user.XSTSUserHash)
	}
}

func TestPersistRefreshToken_CompletesEmptyGamertag(t *testing.T) {
	store := tempStore(t)
	if err := store.UpdateOAuthRefreshToken("2533274858283686", "rt-initial"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := PersistRefreshToken(store, "2533274858283686", "Madina97294", "rt-new", nil); err != nil {
		t.Fatalf("err = %v", err)
	}

	user, _ := store.Load("2533274858283686")
	if user.Gamertag != "Madina97294" {
		t.Errorf("Gamertag = %q (devrait être complété)", user.Gamertag)
	}
	if user.OAuthRefreshToken != "rt-new" {
		t.Errorf("RT = %q", user.OAuthRefreshToken)
	}
}

func TestPersistRefreshToken_DoesNotOverwriteExistingGamertag(t *testing.T) {
	store := tempStore(t)
	if err := store.Upsert(&auth.UserTokens{
		XUID:     "2533274858283686",
		Gamertag: "Madina97294",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := PersistRefreshToken(store, "2533274858283686", "DifferentName", "rt-new", nil); err != nil {
		t.Fatalf("err = %v", err)
	}

	user, _ := store.Load("2533274858283686")
	if user.Gamertag != "Madina97294" {
		t.Errorf("Gamertag écrasé : %q (devrait rester Madina97294)", user.Gamertag)
	}
}

func TestPersistRefreshToken_InvalidatorCalled(t *testing.T) {
	store := tempStore(t)
	inv := &trackingInvalidator{}

	if err := PersistRefreshToken(store, "2533274858283686", "Madina97294", "rt-new", inv.invalidate); err != nil {
		t.Fatalf("err = %v", err)
	}

	calls := inv.snapshot()
	if len(calls) != 1 {
		t.Fatalf("invalidator calls = %d, want 1", len(calls))
	}
	if calls[0] != "2533274858283686" {
		t.Errorf("invalidator called with %q, want 2533274858283686", calls[0])
	}
}

func TestPersistRefreshToken_NilInvalidatorIsSafe(t *testing.T) {
	store := tempStore(t)
	if err := PersistRefreshToken(store, "2533274858283686", "Madina97294", "rt", nil); err != nil {
		t.Fatalf("invalidator nil ne doit pas paniquer ni échouer : %v", err)
	}
}

func TestPersistRefreshToken_InvalidatorNotCalledOnError(t *testing.T) {
	store := tempStore(t)
	inv := &trackingInvalidator{}

	// xuid unsafe → erreur AVANT l'appel invalidator
	_ = PersistRefreshToken(store, "../escape", "Hacker", "rt", inv.invalidate)

	if calls := inv.snapshot(); len(calls) != 0 {
		t.Errorf("invalidator ne devrait pas être appelé sur erreur, got %v", calls)
	}
}

func TestPersistRefreshToken_NilStore(t *testing.T) {
	if err := PersistRefreshToken(nil, "111", "Alice", "rt", nil); err == nil {
		t.Error("store nil → erreur attendue")
	}
}

func TestPersistRefreshToken_EmptyRT(t *testing.T) {
	store := tempStore(t)
	if err := PersistRefreshToken(store, "111", "Alice", "", nil); err == nil {
		t.Error("RT vide → erreur attendue")
	}
}

func TestPersistRefreshToken_UnsafeXUID(t *testing.T) {
	store := tempStore(t)
	if err := PersistRefreshToken(store, "../escape", "Hacker", "rt", nil); err == nil {
		t.Error("xuid unsafe → erreur attendue (propagée du store)")
	}
}

func TestPersistRefreshToken_EmptyGamertagSkipsCompletion(t *testing.T) {
	store := tempStore(t)
	if err := PersistRefreshToken(store, "2533274858283686", "", "rt-only", nil); err != nil {
		t.Fatalf("err = %v", err)
	}

	user, _ := store.Load("2533274858283686")
	if user.OAuthRefreshToken != "rt-only" {
		t.Errorf("RT = %q", user.OAuthRefreshToken)
	}
	if user.Gamertag != "" {
		t.Errorf("Gamertag = %q, want vide (skip completion si paramètre vide)", user.Gamertag)
	}
}

// ─── ResolveXUIDForRotation ───────────────────────────────────────────────

func TestResolveXUIDForRotation_FoundInStore(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "9999999999999999", // ne devrait pas être utilisé
	})
	store := tempStore(t)
	if err := store.Upsert(&auth.UserTokens{
		XUID:     "1111111111111111",
		Gamertag: "Madina97294",
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	got := ResolveXUIDForRotation(context.Background(), store, players, "Madina97294")
	if got != "1111111111111111" {
		t.Errorf("got = %q, want 1111111111111111 (store prioritaire)", got)
	}
}

func TestResolveXUIDForRotation_FallbackToPlayers(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "players-xuid",
	})
	store := tempStore(t)

	got := ResolveXUIDForRotation(context.Background(), store, players, "Madina97294")
	if got != "players-xuid" {
		t.Errorf("got = %q, want players-xuid (fallback)", got)
	}
}

func TestResolveXUIDForRotation_StoreLookupMissesGoesToPlayers(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "players-xuid",
	})
	store := tempStore(t)
	_ = store.Upsert(&auth.UserTokens{
		XUID:     "999",
		Gamertag: "OtherPlayer",
	})

	got := ResolveXUIDForRotation(context.Background(), store, players, "Madina97294")
	if got != "players-xuid" {
		t.Errorf("got = %q, want players-xuid (gamertag pas dans store)", got)
	}
}

func TestResolveXUIDForRotation_NotFoundAnywhere(t *testing.T) {
	players := playerSummaries(map[string]string{
		"OtherPlayer": "other-xuid",
	})
	store := tempStore(t)

	got := ResolveXUIDForRotation(context.Background(), store, players, "Madina97294")
	if got != "" {
		t.Errorf("got = %q, want vide (introuvable partout)", got)
	}
}

func TestResolveXUIDForRotation_NilStoreNilPlayers(t *testing.T) {
	got := ResolveXUIDForRotation(context.Background(), nil, nil, "Madina97294")
	if got != "" {
		t.Errorf("got = %q, want vide", got)
	}
}

func TestResolveXUIDForRotation_NilStoreUsePlayers(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "players-xuid",
	})

	got := ResolveXUIDForRotation(context.Background(), nil, players, "Madina97294")
	if got != "players-xuid" {
		t.Errorf("got = %q, want players-xuid", got)
	}
}

func TestResolveXUIDForRotation_CaseInsensitive(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "players-xuid",
	})

	got := ResolveXUIDForRotation(context.Background(), nil, players, "MADINA97294")
	if got != "players-xuid" {
		t.Errorf("got = %q (case-insensitive attendu)", got)
	}
}

func TestResolveXUIDForRotation_EmptyPlayersListReturnsEmpty(t *testing.T) {
	got := ResolveXUIDForRotation(context.Background(), nil, []domain.PlayerSummary{}, "Madina97294")
	if got != "" {
		t.Errorf("got = %q, want vide (players list vide)", got)
	}
}

// ─── Integration : workflow complet token-capture-like ────────────────────

func TestCaptureWorkflow_FullCycle(t *testing.T) {
	players := playerSummaries(map[string]string{
		"Madina97294": "2533274858283686",
	})
	store := tempStore(t)

	xuid, canonical, err := ResolveXUIDByGamertag(players, "madina97294")
	if err != nil {
		t.Fatalf("ResolveXUIDByGamertag: %v", err)
	}

	rt, err := ParseRefreshTokenStdin(strings.NewReader("rt-from-stdin\n"))
	if err != nil {
		t.Fatalf("ParseRefreshTokenStdin: %v", err)
	}

	inv := &trackingInvalidator{}
	if err := PersistRefreshToken(store, xuid, canonical, rt, inv.invalidate); err != nil {
		t.Fatalf("PersistRefreshToken: %v", err)
	}
	if calls := inv.snapshot(); len(calls) != 1 || calls[0] != xuid {
		t.Errorf("workflow: invalidator calls = %v, want [%s]", calls, xuid)
	}

	user, err := store.Load(xuid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if user.OAuthRefreshToken != "rt-from-stdin" {
		t.Errorf("RT = %q", user.OAuthRefreshToken)
	}
	if user.Gamertag != "Madina97294" {
		t.Errorf("Gamertag = %q", user.Gamertag)
	}
	if user.XUID != xuid {
		t.Errorf("XUID = %q, want %q", user.XUID, xuid)
	}

	byGamertag, err := store.LoadByGamertag("madina97294")
	if err != nil {
		t.Fatalf("LoadByGamertag: %v", err)
	}
	if byGamertag.XUID != xuid {
		t.Errorf("LoadByGamertag XUID = %q, want %q", byGamertag.XUID, xuid)
	}
}
