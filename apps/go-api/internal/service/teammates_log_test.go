// Test ciblé : le silent-drop de buildTeammateRowWithMatches doit émettre
// un slog "teammates_gamertag_not_found" quand un gamertag confirmé n'est
// pas trouvé dans LoadTopTeammates. Cause racine du bug frontend
// "Comparaison inactive même après sélection".
package service

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
	svc := NewTeammatesService(repo)

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
	svc := NewTeammatesService(repo)

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
