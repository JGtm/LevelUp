// Test ciblé : le silent-drop de buildTeammateRowWithMatches doit émettre
// un slog "teammates_gamertag_not_found" quand un gamertag confirmé n'est
// pas trouvé dans LoadTopTeammates. Cause racine du bug frontend
// "Comparaison inactive même après sélection".
package teammates

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// withCapturedLogs remplace temporairement le default slog logger par un
// JSON handler qui écrit dans buf, et restaure l'ancien à la fin du test.
func withCapturedLogs(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)
	fn()
	return buf.String()
}

func TestTeammatesService_GamertagNotFound_EmitsLog(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "x1", Gamertag: "KnownAlly", GamesTogether: 5},
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	logs := withCapturedLogs(t, func() {
		req := domain.TeammatesQueryRequest{
			SelectedGamertags: []string{"GhostPlayer"},
		}
		_, err := svc.GetPage(context.Background(), "player-xuid", req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(logs, "teammates_gamertag_not_found") {
		t.Errorf("expected log 'teammates_gamertag_not_found' in output, got:\n%s", logs)
	}
	if !strings.Contains(logs, `"gamertag":"GhostPlayer"`) {
		t.Errorf("expected gamertag field in log, got:\n%s", logs)
	}
	if !strings.Contains(logs, `"top_rows_count":1`) {
		t.Errorf("expected top_rows_count=1 in log, got:\n%s", logs)
	}
	if !strings.Contains(logs, `"player_xuid":"player-xuid"`) {
		t.Errorf("expected player_xuid in log, got:\n%s", logs)
	}
}

func TestTeammatesService_KnownGamertag_NoSilentDropLog(t *testing.T) {
	// Cas opposé : si le gamertag est dans topRows, aucun log
	// 'teammates_gamertag_not_found' ne doit être émis.
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "KnownAlly", GamesTogether: 5},
		},
		// squadRows vide → row construite avec 0 matches mais pas de drop
		squadRows: nil,
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	logs := withCapturedLogs(t, func() {
		req := domain.TeammatesQueryRequest{
			SelectedGamertags: []string{"KnownAlly"},
		}
		_, err := svc.GetPage(context.Background(), "player-xuid", req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(logs, "teammates_gamertag_not_found") {
		t.Errorf("did not expect 'teammates_gamertag_not_found' for a known gamertag, got:\n%s", logs)
	}
}

// TestTeammatesService_CaseInsensitiveTopMatch : le matching dans topRows
// doit être case-insensitive — le user saisit "Madina97294" mais la DB peut
// avoir "madina97294" (Halo API renvoie tantôt l'un tantôt l'autre).
func TestTeammatesService_CaseInsensitiveTopMatch(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "madina97294", GamesTogether: 42},
		},
		squadRows: []domain.SquadMatchRow{
			{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5},
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), "player-xuid",
		domain.TeammatesQueryRequest{SelectedGamertags: []string{"Madina97294"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Teammates) != 1 {
		t.Fatalf("expected 1 teammate row, got %d", len(resp.Teammates))
	}
	if resp.Teammates[0].EncounterCount != 42 {
		t.Errorf("expected encounterCount=42 from topRows, got %d", resp.Teammates[0].EncounterCount)
	}
	if resp.Teammates[0].WithKPIs.MatchCount != 1 {
		t.Errorf("expected withKPIs.MatchCount=1, got %d", resp.Teammates[0].WithKPIs.MatchCount)
	}
}

// TestTeammatesService_AliasFallback : couvre le cas exact rapporté par
// l'utilisateur — le coéquipier (Madina97294) n'est pas dans le top 50
// (mockée vide) mais shared.xuid_aliases le connaît. La row doit être
// construite avec les KPIs des matchs communs et encounterCount calculé
// depuis len(squadMatches).
//
// C'EST LE TEST QUI AURAIT DÛ EXISTER POUR DÉTECTER LE BUG "aucun graphe
// affiché malgré 2 amis sélectionnés".
func TestTeammatesService_AliasFallback_BuildsRowFromOutOfTopGamertag(t *testing.T) {
	repo := &mockSquadRepo{
		// Top vide intentionnellement : on simule un user qui sélectionne
		// quelqu'un hors top 50 OU avec une casse mismatch.
		topRows: []domain.TopTeammateRow{},
		// L'alias DB connaît bien le gamertag.
		lookupAliases: map[string]string{
			"madina97294": "tm-madina-xuid",
		},
		// Les matchs communs avec ce coéquipier existent.
		squadRows: []domain.SquadMatchRow{
			{MatchID: "m1", Outcome: domain.OutcomeWin, Kills: 12, Deaths: 4},
			{MatchID: "m2", Outcome: domain.OutcomeWin, Kills: 8, Deaths: 6},
			{MatchID: "m3", Outcome: domain.OutcomeLoss, Kills: 5, Deaths: 9},
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	logs := withCapturedLogs(t, func() {
		resp, err := svc.GetPage(context.Background(), "player-xuid",
			domain.TeammatesQueryRequest{SelectedGamertags: []string{"Madina97294"}},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Teammates) != 1 {
			t.Fatalf("expected 1 teammate row built via alias fallback, got %d", len(resp.Teammates))
		}
		row := resp.Teammates[0]
		if row.Gamertag != "Madina97294" {
			t.Errorf("expected gamertag preserved as input casing, got %q", row.Gamertag)
		}
		if row.XUID == nil || *row.XUID != "tm-madina-xuid" {
			t.Errorf("expected XUID resolved via alias = tm-madina-xuid, got %v", row.XUID)
		}
		if row.EncounterCount != 3 {
			t.Errorf("expected encounterCount=3 (= len(squadMatches)) when fallback used, got %d",
				row.EncounterCount)
		}
		if row.WithKPIs.MatchCount != 3 {
			t.Errorf("expected withKPIs.MatchCount=3, got %d", row.WithKPIs.MatchCount)
		}
		if row.WithKPIs.Wins != 2 {
			t.Errorf("expected 2 wins, got %d", row.WithKPIs.Wins)
		}
	})

	// Le silent-drop ne doit PAS se déclencher quand le fallback réussit.
	if strings.Contains(logs, "teammates_gamertag_not_found") {
		t.Errorf("alias fallback succeeded but 'not_found' log was emitted: %s", logs)
	}
}

// TestTeammatesService_AliasFallback_StillNotFound : si ni topRows ni les
// aliases ne contiennent le gamertag, on log + on drop proprement (pas de
// crash, juste un teammate manquant).
func TestTeammatesService_AliasFallback_StillNotFound(t *testing.T) {
	repo := &mockSquadRepo{
		topRows:       []domain.TopTeammateRow{},
		lookupAliases: map[string]string{}, // alias inconnu
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	logs := withCapturedLogs(t, func() {
		resp, err := svc.GetPage(context.Background(), "player-xuid",
			domain.TeammatesQueryRequest{SelectedGamertags: []string{"GhostPlayer42"}},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Teammates) != 0 {
			t.Errorf("expected 0 teammate rows when alias unknown, got %d", len(resp.Teammates))
		}
	})

	if !strings.Contains(logs, "teammates_gamertag_not_found") {
		t.Errorf("expected silent-drop log when gamertag truly unknown, got:\n%s", logs)
	}
}
