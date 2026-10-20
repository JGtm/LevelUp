package pool

import (
	"context"
	"errors"
	"testing"
)

// pool_maxsize_test.go — MaxSize plafonne les slots SAINS (D2, plan robustesse 2026-09-16).
//
// Constat fondateur : `--token-pool-size 1` tronquait le scan AVANT de résoudre, prenait la
// première source venue (Chocoboflor, révoquée) et le pool échouait sur « aucun slot créé ».
// L'ordre du scan vient d'une map : il changeait d'une exécution à l'autre.
//
// Ces tests rougissent si la troncature `sources[:MaxSize]` revient (le pool ne se crée plus)
// ou si le tri par gamertag disparaît (le parc retenu dépend de l'ordre d'entrée).

// revokedResolver : toute source nommée dans `revoked` échoue à la résolution, les autres
// rendent un token de test (comportement de testResolver).
type revokedResolver struct {
	*testResolver
	revoked map[string]bool
}

func (r *revokedResolver) Resolve(ctx context.Context, src CredentialSource) (*ResolvedTokens, error) {
	if r.revoked[src.Gamertag] {
		return nil, errors.New("AADSTS70000: refresh token révoqué")
	}
	return r.testResolver.Resolve(ctx, src)
}

func newRevokedResolver(revoked ...string) *revokedResolver {
	m := make(map[string]bool, len(revoked))
	for _, gt := range revoked {
		m[gt] = true
	}
	return &revokedResolver{
		testResolver: &testResolver{resolved: make(map[string]*ResolvedTokens)},
		revoked:      m,
	}
}

// sourcesFor construit des sources de test dans l'ordre donné.
func sourcesFor(gamertags ...string) []CredentialSource {
	out := make([]CredentialSource, 0, len(gamertags))
	for i, gt := range gamertags {
		out = append(out, CredentialSource{
			Gamertag:     gt,
			XUID:         string(rune(2000 + i)),
			RefreshToken: "rt_" + gt,
			Source:       "test",
		})
	}
	return out
}

func TestNewPool_MaxSize1_PlafonneLesSlotsSains(t *testing.T) {
	// Chocoboflor est révoquée ET première dans l'ordre alphabétique : avec l'ancienne
	// troncature, elle était la seule tentée et le pool n'avait aucun slot.
	resolver := newRevokedResolver("Chocoboflor")
	p, err := NewPool(context.Background(), resolver,
		sourcesFor("Chocoboflor", "DankerGlue", "Bianca"), PoolOptions{MaxSize: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	if got := p.Size(); got != 1 {
		t.Fatalf("Size() = %d, attendu 1 slot sain", got)
	}
	// Première SAINE dans l'ordre alphabétique : Bianca (Chocoboflor est révoquée).
	if !p.HasPlayer("Bianca") {
		t.Errorf("attendu Bianca (première saine par ordre alphabétique), pool = %d slot(s)", p.Size())
	}
}

func TestNewPool_MaxSize2_DeuxSlotsSains(t *testing.T) {
	resolver := newRevokedResolver("Chocoboflor")
	p, err := NewPool(context.Background(), resolver,
		sourcesFor("DankerGlue", "Chocoboflor", "Bianca", "Alice"), PoolOptions{MaxSize: 2})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	if got := p.Size(); got != 2 {
		t.Fatalf("Size() = %d, attendu 2", got)
	}
	for _, gt := range []string{"Alice", "Bianca"} {
		if !p.HasPlayer(gt) {
			t.Errorf("slot attendu pour %s (deux premières saines par ordre alphabétique)", gt)
		}
	}
}

func TestNewPool_MaxSize0_TousLesSlotsSains(t *testing.T) {
	resolver := newRevokedResolver("Chocoboflor")
	p, err := NewPool(context.Background(), resolver,
		sourcesFor("DankerGlue", "Chocoboflor", "Bianca"), PoolOptions{MaxSize: 0})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	if got := p.Size(); got != 2 {
		t.Fatalf("Size() = %d, attendu 2 (toutes les sources saines)", got)
	}
	if p.HasPlayer("Chocoboflor") {
		t.Error("une source révoquée ne doit pas créer de slot")
	}
}

func TestNewPool_MaxSize_MemeParcQuelQueSoitLOrdreDEntree(t *testing.T) {
	ordres := [][]string{
		{"DankerGlue", "Chocoboflor", "Bianca", "Alice"},
		{"Alice", "Bianca", "Chocoboflor", "DankerGlue"},
		{"Bianca", "DankerGlue", "Alice", "Chocoboflor"},
	}
	for _, ordre := range ordres {
		resolver := newRevokedResolver("Chocoboflor")
		p, err := NewPool(context.Background(), resolver, sourcesFor(ordre...), PoolOptions{MaxSize: 2})
		if err != nil {
			t.Fatalf("NewPool(%v): %v", ordre, err)
		}
		if !p.HasPlayer("Alice") || !p.HasPlayer("Bianca") {
			t.Errorf("ordre d'entrée %v : parc retenu différent (Size=%d)", ordre, p.Size())
		}
		p.Close()
	}
}

func TestNewPool_ToutesLesSourcesRevoquees_ErreurExplicite(t *testing.T) {
	resolver := newRevokedResolver("Alice", "Bianca")
	_, err := NewPool(context.Background(), resolver, sourcesFor("Alice", "Bianca"), PoolOptions{MaxSize: 1})
	if err == nil {
		t.Fatal("attendu une erreur quand aucune source ne se résout")
	}
	if err.Error() != "pool: aucun slot créé (toutes les résolutions ont échoué)" {
		t.Errorf("message inattendu: %v", err)
	}
}
