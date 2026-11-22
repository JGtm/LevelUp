package wire

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/pool"
)

// resolveurJetonsParCompte rend, pour chaque compte, un Spartan token nommé d'après lui : le test
// lit ainsi QUEL compte a servi.
type resolveurJetonsParCompte struct{}

func (resolveurJetonsParCompte) Resolve(_ context.Context, src pool.CredentialSource) (*pool.ResolvedTokens, error) {
	return &pool.ResolvedTokens{
		Gamertag:  src.Gamertag,
		XUID:      src.XUID,
		Tokens:    &domain.HaloTokens{SpartanToken: "spartan_" + src.Gamertag},
		ExpiresAt: time.Now().Add(time.Hour),
		Source:    src.Source,
	}, nil
}

func (resolveurJetonsParCompte) Refresh(ctx context.Context, gamertag string) (*pool.ResolvedTokens, error) {
	return resolveurJetonsParCompte{}.Resolve(ctx, pool.CredentialSource{Gamertag: gamertag})
}

// registreAvecMiseEnCommun câble un vrai pool : `retenus` sont les comptes sans accord de mise en
// commun, les autres ont consenti.
func registreAvecMiseEnCommun(t *testing.T, consentants []string, retenus []string) *ServiceRegistry {
	t.Helper()
	var sources []pool.CredentialSource
	for _, gt := range consentants {
		sources = append(sources, pool.CredentialSource{Gamertag: gt, XUID: "xuid-" + gt, RefreshToken: "rt-" + gt, Source: "test"})
	}
	for _, gt := range retenus {
		sources = append(sources, pool.CredentialSource{Gamertag: gt, XUID: "xuid-" + gt, RefreshToken: "rt-" + gt, Source: "test", PoolExcluded: true})
	}
	p, err := pool.NewPool(context.Background(), resolveurJetonsParCompte{}, sources, pool.PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(p.Close)
	return (&ServiceRegistry{}).WithTokenPool(p)
}

// Un compte retenu (sans accord) n'est jamais servi, quel que soit le nombre de prises.
func TestHaloTokensForDrain_CompteRetenuJamaisServi(t *testing.T) {
	reg := registreAvecMiseEnCommun(t, []string{"Ouvert"}, []string{"Retenu"})
	for i := 0; i < 20; i++ {
		tokens, err := reg.haloTokensForDrain(context.Background())
		if err != nil {
			t.Fatalf("prise %d : %v", i, err)
		}
		if tokens.SpartanToken != "spartan_Ouvert" {
			t.Fatalf("prise %d : jeton %q servi, seul le compte consentant le peut", i, tokens.SpartanToken)
		}
	}
}

// Sans aucun compte consentant : rien n'est servi, erreur typée portant la cause du pool.
func TestHaloTokensForDrain_AucunCompteConsentant(t *testing.T) {
	reg := registreAvecMiseEnCommun(t, nil, []string{"Retenu", "AutreRetenu"})
	tokens, err := reg.haloTokensForDrain(context.Background())
	if tokens != nil {
		t.Fatalf("jeton servi alors qu'aucun compte n'a consenti : %+v", tokens)
	}
	if !errors.Is(err, ErrNoPublicReadToken) {
		t.Fatalf("erreur = %v, attendu ErrNoPublicReadToken", err)
	}
	if !errors.Is(err, pool.ErrNoHealthySlot) {
		t.Fatalf("erreur = %v, la cause du pool doit rester atteignable", err)
	}
}

// Sans source câblée : erreur typée, et aucun profil n'est lu (le registre n'a pas de configuration :
// toute lecture des profils paniquerait).
func TestHaloTokensForDrain_SansSourceCablee(t *testing.T) {
	reg := &ServiceRegistry{}
	tokens, err := reg.haloTokensForDrain(context.Background())
	if tokens != nil || !errors.Is(err, ErrNoPublicReadToken) {
		t.Fatalf("(%v, %v), attendu (nil, ErrNoPublicReadToken)", tokens, err)
	}
}

func TestHaloTokensForDrain_SourceSansJeton(t *testing.T) {
	for nom, stub := range map[string]*stubPoolSource{
		"erreur":       {err: errors.New("pool vide")},
		"jeton absent": {},
	} {
		t.Run(nom, func(t *testing.T) {
			reg := &ServiceRegistry{publicReadTokenSrc: stub}
			tokens, err := reg.haloTokensForDrain(context.Background())
			if tokens != nil || !errors.Is(err, ErrNoPublicReadToken) {
				t.Fatalf("(%v, %v), attendu (nil, ErrNoPublicReadToken)", tokens, err)
			}
			if stub.calls != 1 {
				t.Fatalf("la source a été sollicitée %d fois, attendu 1", stub.calls)
			}
		})
	}
}

func TestHaloTokensForDrain_RendLeJetonDeLaSource(t *testing.T) {
	want := freshTokens("sp-pool")
	reg := &ServiceRegistry{publicReadTokenSrc: &stubPoolSource{tokens: want, gamertag: "Ouvert"}}
	got, err := reg.haloTokensForDrain(context.Background())
	if err != nil || got != want {
		t.Fatalf("(%v, %v), attendu le jeton de la source", got, err)
	}
}
