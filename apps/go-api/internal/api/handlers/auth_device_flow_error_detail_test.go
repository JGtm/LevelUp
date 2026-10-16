package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	auth_platform "levelup/go-api/internal/platform/auth"
)

// secretCause : texte d'une erreur interne (endpoint, jeton) qui ne doit jamais atteindre le client.
const secretCause = "post https://login.microsoftonline.com/consumers/oauth2/v2.0/token: invalid_grant token=abc123"

// failingFlow : un device-flow dont l'attente de l'utilisateur échoue avec secretCause.
type failingFlow struct{}

func (failingFlow) GetMessage() string         { return "" }
func (failingFlow) GetUserCode() string        { return "ABCD1234" }
func (failingFlow) GetVerificationURL() string { return "https://microsoft.com/link" }
func (failingFlow) GetExpiresIn() int          { return 60 }
func (failingFlow) GetFlowType() string        { return "sisu" }
func (failingFlow) AcquireToken(context.Context) (string, error) {
	return "", errors.New(secretCause)
}

// L9-3 : l'état d'une tentative en échec rend le code seul ; le texte de l'erreur
// (réponse Microsoft, jeton, endpoint) reste au journal.
func TestDeviceFlowStatus_FailureCarriesCodeOnly(t *testing.T) {
	attempts := auth_platform.NewAttemptStore()
	h := NewAuthHandler(nil, attempts, nil)
	a, _, err := attempts.GetOrCreate("sess-1", "203.0.113.1")
	if err != nil {
		t.Fatalf("GetOrCreate : %v", err)
	}

	h.pollDeviceFlow(a.AttemptID, failingFlow{})

	snap := attempts.Snapshot(a.AttemptID)
	if snap == nil || snap.Status != auth_platform.AttemptStatusFailed || snap.ErrorCode != errCodeDeviceFlowAcquire {
		t.Fatalf("tentative attendue en échec %q, obtenu %+v", errCodeDeviceFlowAcquire, snap)
	}
	body, err := json.Marshal(deviceFlowStatusResponse(snap))
	if err != nil {
		t.Fatalf("sérialisation : %v", err)
	}
	for _, leaked := range []string{secretCause, "invalid_grant", "abc123", "error_detail"} {
		if strings.Contains(string(body), leaked) {
			t.Errorf("la réponse d'état laisse fuir %q : %s", leaked, body)
		}
	}
	if !strings.Contains(string(body), errCodeDeviceFlowAcquire) {
		t.Errorf("le code d'erreur manque : %s", body)
	}
}
