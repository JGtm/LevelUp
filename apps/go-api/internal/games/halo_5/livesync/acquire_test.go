package livesync

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/auth/pool"
)

// acqPool implémente pool.Pool : Acquire est scénarisé par policy (pinned vs public)
// pour piloter le fallback d'AcquireRunner sans réseau.
type acqPool struct {
	pinnedErr error
	publicErr error
	acquired  []pool.AcquirePolicy
}

func (p *acqPool) Acquire(_ context.Context, policy pool.AcquirePolicy, _ string) (*pool.Lease, error) {
	p.acquired = append(p.acquired, policy)
	switch policy {
	case pool.PolicyPinnedPlayer:
		if p.pinnedErr != nil {
			return nil, p.pinnedErr
		}
		return &pool.Lease{Tokens: &domain.HaloTokens{SpartanToken: "pinned"}, Release: func() {}}, nil
	default: // PolicyAnyPublic
		if p.publicErr != nil {
			return nil, p.publicErr
		}
		return &pool.Lease{Tokens: &domain.HaloTokens{SpartanToken: "pool"}, Release: func() {}}, nil
	}
}

func (p *acqPool) Size() int                                                      { return 1 }
func (p *acqPool) HasPlayer(string) bool                                          { return true }
func (p *acqPool) MarkUnhealthy(string, error)                                    {}
func (p *acqPool) OnHTTPError(int, time.Duration)                                 {}
func (p *acqPool) AddOrUpdateSource(context.Context, pool.CredentialSource) error { return nil }
func (p *acqPool) Close()                                                         {}

// TestAcquireRunner_PinnedDead_FallsBackToPool : RT du joueur mort (pinned KO) →
// AcquireRunner retombe sur un token POOL (PolicyAnyPublic) et réussit, en taguant le
// ctx avec le xuid du joueur CONSULTÉ. C'est ce qui débloque le sync rang/XP des
// joueurs au refresh_token mort (stats publiques poolables, non gatées par leur token).
func TestAcquireRunner_PinnedDead_FallsBackToPool(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	p := &acqPool{pinnedErr: errors.New("ErrNoTokenForPlayer (RT mort)")}

	r, runCtx, release, err := AcquireRunner(context.Background(), p, cfg, halo5.TitleSlug, "Chocoboflor", "xuid-choco")
	if err != nil {
		t.Fatalf("AcquireRunner: fallback attendu, err: %v", err)
	}
	if r == nil {
		t.Fatal("runner nil après fallback")
	}
	defer release()

	tok := ctxkeys.HaloTokens(runCtx)
	if tok == nil || tok.SpartanToken != "pool" {
		t.Errorf("ctx token = %v, want token pool (fallback)", tok)
	}
	if got := ctxkeys.HaloXUID(runCtx); got != "xuid-choco" {
		t.Errorf("ctx xuid = %q, want xuid-choco (joueur consulté, pas le compte pool)", got)
	}
	if len(p.acquired) != 2 || p.acquired[0] != pool.PolicyPinnedPlayer || p.acquired[1] != pool.PolicyAnyPublic {
		t.Errorf("policies tentées = %v, want [pinned, anyPublic]", p.acquired)
	}
}

// TestAcquireRunner_PinnedOK_NoFallback : RT du joueur sain → pinned utilisé, AUCUN
// fallback pool (pas de régression ni de charge pool inutile pour les joueurs vivants).
func TestAcquireRunner_PinnedOK_NoFallback(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	p := &acqPool{}

	_, runCtx, release, err := AcquireRunner(context.Background(), p, cfg, halo5.TitleSlug, "JGtm", "xuid-jg")
	if err != nil {
		t.Fatalf("AcquireRunner: %v", err)
	}
	defer release()
	if tok := ctxkeys.HaloTokens(runCtx); tok == nil || tok.SpartanToken != "pinned" {
		t.Errorf("ctx token = %v, want token pinned", tok)
	}
	if len(p.acquired) != 1 || p.acquired[0] != pool.PolicyPinnedPlayer {
		t.Errorf("policies tentées = %v, want [pinned] seul", p.acquired)
	}
}

// TestAcquireRunner_BothFail_Errors : pinned ET pool KO → erreur (sync skippé, comme
// avant le fallback — aucune dégradation silencieuse).
func TestAcquireRunner_BothFail_Errors(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	p := &acqPool{pinnedErr: errors.New("no token"), publicErr: errors.New("pool vide")}
	if _, _, _, err := AcquireRunner(context.Background(), p, cfg, halo5.TitleSlug, "X", "xX"); err == nil {
		t.Error("want erreur quand pinned ET pool échouent")
	}
}
