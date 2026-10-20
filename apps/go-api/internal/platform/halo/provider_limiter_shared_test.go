package halo

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// Les clones d'un fournisseur (WithTokens, WithAssetResolver...) partagent son limiteur
// sortant : construire le fournisseur une fois suffit à borner le débit de tous ses
// appels. Seul WithRateLimit en pose un neuf.
func TestProviderClonesShareTheOutboundLimiter(t *testing.T) {
	base := NewHaloProvider()
	if got := base.WithTokens(&domain.HaloTokens{}).limiter; got != base.limiter {
		t.Fatal("WithTokens doit partager le limiteur du fournisseur d'origine")
	}
	if got := base.WithTitleSlug("halo_infinite").limiter; got != base.limiter {
		t.Fatal("WithTitleSlug doit partager le limiteur du fournisseur d'origine")
	}
	if got := base.WithRateLimit(5).limiter; got == base.limiter {
		t.Fatal("WithRateLimit doit poser un limiteur neuf")
	}
}
