// Package handlers — auth_device_flow_singleflight_test.go : régression du
// single-flight de POST /auth/device-flow/start.
//
// Bug corrigé (2026-07-13) : quand deux start concurrents partagent la même
// session (2e onglet, double-fire des effets React en dev), le second recevait
// la tentative pending AVANT que la requête créatrice ait terminé InitDeviceFlow
// → réponse 200 avec user_code/verification_uri VIDES, client bloqué sur le
// spinner « Génération du code… » (le GET status ne porte pas le code).
// Le handler doit attendre que la tentative soit initialisée (waitDeviceFlowReady).
//
// Test interne (package handlers) : appelle handleStartDeviceFlow directement
// avec des sessions injectées (middleware.InjectSession) — impossible à piloter
// proprement via HTTP car le cookie de session n'est émis qu'à la fin de la
// requête créatrice, qui est précisément celle qu'on bloque.
package handlers

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
)

// gatedTokenProvider bloque InitDeviceFlow jusqu'à la fermeture de release, et
// signale son entrée via started (permet de séquencer créateur / single-flight).
// Exchange bloque sur exchangeRelease : le stub DeviceFlow rend AcquireToken
// immédiatement, donc sans ce gate la goroutine pollDeviceFlow enchaînerait sur
// Exchange et marquerait l'attempt failed pendant les assertions du test.
type gatedTokenProvider struct {
	startedOnce     sync.Once
	started         chan struct{}
	release         chan struct{}
	exchangeRelease chan struct{}
}

var _ auth_platform.TokenProvider = (*gatedTokenProvider)(nil)

func (g *gatedTokenProvider) InitDeviceFlow(ctx context.Context) (auth_platform.DeviceFlow, error) {
	g.startedOnce.Do(func() { close(g.started) })
	select {
	case <-g.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return auth_platform.NewStubDeviceFlow("GATE1234", "https://microsoft.com/link", "Entrez GATE1234", 900, "sisu"), nil
}

func (g *gatedTokenProvider) TrySilentRefresh(context.Context, string) (string, error) {
	return "", nil
}
func (g *gatedTokenProvider) TryOAuthRefresh(context.Context, string) (string, error) {
	return "", nil
}
func (g *gatedTokenProvider) TryOAuthRefreshWithRotation(context.Context, string) (string, string, error) {
	return "", "", nil
}
func (g *gatedTokenProvider) Exchange(ctx context.Context, _ string) (*auth_platform.ExchangeResult, error) {
	select {
	case <-g.exchangeRelease:
	case <-ctx.Done():
	}
	return nil, context.Canceled
}

// TestStartDeviceFlow_SingleFlightWaitsForInit : un start concurrent sur la même
// session NE DOIT PAS répondre avec un user_code vide pendant que le créateur est
// encore dans InitDeviceFlow — il attend et renvoie la tentative remplie.
func TestStartDeviceFlow_SingleFlightWaitsForInit(t *testing.T) {
	provider := &gatedTokenProvider{
		started:         make(chan struct{}),
		release:         make(chan struct{}),
		exchangeRelease: make(chan struct{}),
	}
	t.Cleanup(func() { close(provider.exchangeRelease) }) // libère la goroutine pollDeviceFlow
	sessStore := session.NewStore(filepath.Join(t.TempDir(), "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := NewAuthHandler(sessStore, auth_platform.NewAttemptStore(), false, provider)

	// Deux SessionData distinctes portant le MÊME SessionID : c'est la réalité
	// HTTP (chaque requête charge sa propre copie de la session depuis le store).
	ctxCreator := middleware.InjectSession(context.Background(), &domain.SessionData{SessionID: "sess-sf"})
	ctxSecond := middleware.InjectSession(context.Background(), &domain.SessionData{SessionID: "sess-sf"})

	var wg sync.WaitGroup
	var creatorOut, secondOut *deviceFlowStartOutput
	var creatorErr, secondErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		creatorOut, creatorErr = h.handleStartDeviceFlow(ctxCreator, nil)
	}()

	// Attendre que le créateur soit ENTRÉ dans InitDeviceFlow (tentative pending
	// créée, champs encore vides) avant de lancer le start concurrent.
	select {
	case <-provider.started:
	case <-time.After(5 * time.Second):
		t.Fatal("le créateur n'a jamais atteint InitDeviceFlow")
	}

	secondDone := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		secondOut, secondErr = h.handleStartDeviceFlow(ctxSecond, nil)
		close(secondDone)
	}()

	// Le second start doit ATTENDRE (pas de réponse tant que le créateur bloque).
	select {
	case <-secondDone:
		t.Fatal("le start single-flight a répondu pendant l'InitDeviceFlow du créateur (réponse vide du bug d'origine)")
	case <-time.After(400 * time.Millisecond):
		// attendu : toujours en attente
	}

	close(provider.release)
	wg.Wait()

	if creatorErr != nil {
		t.Fatalf("créateur en erreur: %v", creatorErr)
	}
	if secondErr != nil {
		t.Fatalf("single-flight en erreur: %v", secondErr)
	}
	if creatorOut.Body.UserCode != "GATE1234" {
		t.Errorf("créateur user_code = %q, want GATE1234", creatorOut.Body.UserCode)
	}
	if secondOut.Body.UserCode != "GATE1234" {
		t.Errorf("single-flight user_code = %q, want GATE1234 (réponse vide = régression)", secondOut.Body.UserCode)
	}
	if secondOut.Body.AttemptID != creatorOut.Body.AttemptID {
		t.Errorf("attempt_id divergents: créateur %q vs single-flight %q", creatorOut.Body.AttemptID, secondOut.Body.AttemptID)
	}
	if secondOut.Body.VerificationURI == "" {
		t.Errorf("single-flight verification_uri vide")
	}
}

// TestStartDeviceFlow_SingleFlightPropagatesFailure : si le créateur échoue dans
// InitDeviceFlow, le start concurrent en attente reçoit une erreur (pas un
// payload vide ni un blocage jusqu'au timeout).
func TestStartDeviceFlow_SingleFlightPropagatesFailure(t *testing.T) {
	provider := &failAfterGateProvider{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	sessStore := session.NewStore(filepath.Join(t.TempDir(), "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	h := NewAuthHandler(sessStore, auth_platform.NewAttemptStore(), false, provider)

	ctxCreator := middleware.InjectSession(context.Background(), &domain.SessionData{SessionID: "sess-fail"})
	ctxSecond := middleware.InjectSession(context.Background(), &domain.SessionData{SessionID: "sess-fail"})

	var wg sync.WaitGroup
	var secondErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = h.handleStartDeviceFlow(ctxCreator, nil)
	}()
	select {
	case <-provider.started:
	case <-time.After(5 * time.Second):
		t.Fatal("le créateur n'a jamais atteint InitDeviceFlow")
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, secondErr = h.handleStartDeviceFlow(ctxSecond, nil)
	}()
	time.Sleep(200 * time.Millisecond) // laisser le second entrer dans l'attente
	close(provider.release)            // le créateur échoue
	wg.Wait()

	if secondErr == nil {
		t.Fatal("le single-flight devrait propager l'échec du créateur, a reçu nil")
	}
}

// failAfterGateProvider échoue InitDeviceFlow après la levée du gate.
type failAfterGateProvider struct {
	startedOnce sync.Once
	started     chan struct{}
	release     chan struct{}
}

var _ auth_platform.TokenProvider = (*failAfterGateProvider)(nil)

func (g *failAfterGateProvider) InitDeviceFlow(ctx context.Context) (auth_platform.DeviceFlow, error) {
	g.startedOnce.Do(func() { close(g.started) })
	select {
	case <-g.release:
	case <-ctx.Done():
	}
	return nil, context.DeadlineExceeded
}

func (g *failAfterGateProvider) TrySilentRefresh(context.Context, string) (string, error) {
	return "", nil
}
func (g *failAfterGateProvider) TryOAuthRefresh(context.Context, string) (string, error) {
	return "", nil
}
func (g *failAfterGateProvider) TryOAuthRefreshWithRotation(context.Context, string) (string, string, error) {
	return "", "", nil
}
func (g *failAfterGateProvider) Exchange(context.Context, string) (*auth_platform.ExchangeResult, error) {
	return nil, context.Canceled
}
