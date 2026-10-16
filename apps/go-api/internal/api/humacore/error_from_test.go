package humacore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

const secretError = "sql: open /data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb: no such table match_stats"

func TestFixedMessage(t *testing.T) {
	cases := []struct {
		status int
		code   string
		want   string
	}{
		{http.StatusNotFound, "player_not_found", "player not found"},
		{http.StatusBadRequest, "invalid_body", "invalid body"},
		{http.StatusConflict, "campaign_already_active", "campaign already active"},
		{http.StatusInternalServerError, "achievements_error", "internal error"},
		{http.StatusServiceUnavailable, "db_unavailable", "internal error"},
	}
	for _, tc := range cases {
		if got := FixedMessage(tc.status, tc.code); got != tc.want {
			t.Errorf("FixedMessage(%d, %q) = %q, attendu %q", tc.status, tc.code, got, tc.want)
		}
	}
}

// L9-5 : le client reçoit le message fixe du code, la cause va au journal.
func TestNewErrorFrom_FixedMessageAndLoggedCause(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	for _, status := range []int{http.StatusNotFound, http.StatusBadRequest, http.StatusInternalServerError} {
		logs.Reset()
		got := NewErrorFrom(context.Background(), status, "some_code", errors.New(secretError))

		body, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("sérialisation : %v", err)
		}
		if strings.Contains(string(body), "duckdb") || strings.Contains(string(body), "match_stats") {
			t.Errorf("statut %d : la réponse laisse fuir la cause : %s", status, body)
		}
		var decoded struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("corps illisible : %v", err)
		}
		if decoded.Code != "some_code" || decoded.Message != FixedMessage(status, "some_code") {
			t.Errorf("statut %d : corps %+v", status, decoded)
		}
		if got.GetStatus() != status {
			t.Errorf("statut = %d, attendu %d", got.GetStatus(), status)
		}
		if !strings.Contains(logs.String(), "match_stats") {
			t.Errorf("statut %d : la cause doit figurer au journal : %q", status, logs.String())
		}
	}
}
