// Package sync — engine_postsync_csr_warn_test.go : test du WARN log explicite
// quand csrSeasonID n'est pas configuré.
//
// Phase 10 du plan pipeline CSR : la régression silencieuse "snapshot vide
// éternellement" doit devenir visible aux ops via un log structuré au lieu
// d'un return silencieux.
package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestRunCSRSnapshotSync_EmptySeasonID_EmitsWarnWithGuidance(t *testing.T) {
	// Capture slog output via JSON handler dans buffer.
	var buf bytes.Buffer
	captureHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	prev := slog.Default()
	slog.SetDefault(slog.New(captureHandler))
	defer slog.SetDefault(prev)

	// SyncEngine minimal avec csrSeasonID vide (default).
	// On utilise la zero-value du struct ; les autres champs ne sont pas accédés
	// avant le early return sur le seasonID check.
	e := &SyncEngine{
		gamertag: "TestPlayer",
		// csrSeasonID = "" (zero value)
	}

	// Le second et troisième paramètres ne sont jamais déférencés en cas d'early
	// return : on peut passer nil sans plantage.
	e.runCSRSnapshotSync(context.Background(), nil, nil)

	// Parser les lignes JSON émises et vérifier qu'au moins une porte le bon
	// message WARN avec les indices d'action (csr_season_id, app_settings.json).
	out := buf.String()
	if out == "" {
		t.Fatal("aucun log émis ; le early return aurait dû produire un WARN")
	}
	var found bool
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("ligne non-JSON : %s", line)
			continue
		}
		level, _ := entry["level"].(string)
		msg, _ := entry["msg"].(string)
		if level == "WARN" && strings.Contains(msg, "csr_season_id") {
			found = true
			// Sanity checks supplémentaires : le message DOIT guider l'utilisateur
			// vers les 2 options de fix.
			if !strings.Contains(msg, "app_settings.json") {
				t.Error("WARN log devrait mentionner app_settings.json (action #1 fix)")
			}
			if !strings.Contains(msg, "LEVELUP_CSR_SEASON_ID") {
				t.Error("WARN log devrait mentionner l'env var LEVELUP_CSR_SEASON_ID (action #2 fix)")
			}
			if gamertag, _ := entry["gamertag"].(string); gamertag != "TestPlayer" {
				t.Errorf("WARN log devrait inclure attribut gamertag=TestPlayer, got %q", gamertag)
			}
		}
	}
	if !found {
		t.Errorf("aucun WARN log avec mention 'csr_season_id' trouvé.\nOutput complet :\n%s", out)
	}
}

func TestRunCSRSnapshotSync_NonEmptySeasonID_NoWarnEmitted(t *testing.T) {
	// Avec csrSeasonID configuré, pas de WARN explicite (le path normal va plus
	// loin et dépend de syncPlayerCSRs qui requiert un client Halo). On vérifie
	// juste qu'on ne tape pas le WARN du early return.
	var buf bytes.Buffer
	captureHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	prev := slog.Default()
	slog.SetDefault(slog.New(captureHandler))
	defer slog.SetDefault(prev)

	e := &SyncEngine{
		gamertag:    "TestPlayer",
		csrSeasonID: "CsrSeason13-1",
	}

	// Le path complet va planter sur client=nil dans syncPlayerCSRs, mais on
	// veut juste vérifier l'absence du WARN early-return. On capture la panic
	// via defer pour ne pas faire échouer le test.
	defer func() {
		_ = recover() // ignore panic du nil client
		out := buf.String()
		if strings.Contains(out, "csr_season_id non configuré") {
			t.Errorf("WARN early-return ne devrait PAS être émis quand csrSeasonID est défini.\nOutput :\n%s", out)
		}
	}()
	e.runCSRSnapshotSync(context.Background(), nil, nil)
}
