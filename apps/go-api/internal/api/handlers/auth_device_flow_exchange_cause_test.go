package handlers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	auth_platform "levelup/go-api/internal/platform/auth"
)

// exchangeFailingFlow : un device-flow dont l'attente réussit et dont l'échange Halo échoue avec err.
type exchangeFailingFlow struct {
	failingFlow
	err error
}

func (f exchangeFailingFlow) AcquireToken(context.Context) (string, error) { return "ms-token", nil }
func (f exchangeFailingFlow) ExchangeFlow(context.Context, string) (*auth_platform.ExchangeResult, error) {
	return nil, f.err
}

// La cause d'un échec de l'échange Halo se lit sur le code de la tentative : des services qui ne
// répondent pas (réessayer plus tard) ne se confondent pas avec un refus du compte.
func TestPollDeviceFlow_ExchangeFailureCode(t *testing.T) {
	for _, tc := range []struct {
		nom  string
		err  error
		want string
	}{
		{"délai dépassé", fmt.Errorf("sisu_provider: échange XBL: %w", context.DeadlineExceeded), errCodeHaloUnreachable},
		{"panne 503 des services", errors.New("XSTS token: HTTP 503 depuis https://xsts/x: indisponible"), errCodeHaloUnreachable},
		{"pas de profil Xbox", errors.New("XSTS token: HTTP 401 depuis https://xsts/x: XErr 2148916233"), errCodeHaloExchange},
		{"jeton absent", errors.New("XSTS token: token absent dans la réponse XSTS"), errCodeHaloExchange},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			attempts := auth_platform.NewAttemptStore()
			h := NewAuthHandler(nil, attempts, nil)
			a, _, err := attempts.GetOrCreate("sess-1", "203.0.113.1")
			if err != nil {
				t.Fatalf("GetOrCreate : %v", err)
			}
			h.pollDeviceFlow(a.AttemptID, exchangeFailingFlow{err: tc.err})
			snap := attempts.Snapshot(a.AttemptID)
			if snap == nil || snap.Status != auth_platform.AttemptStatusFailed || snap.ErrorCode != tc.want {
				t.Fatalf("code attendu %q, obtenu %+v", tc.want, snap)
			}
		})
	}
}
