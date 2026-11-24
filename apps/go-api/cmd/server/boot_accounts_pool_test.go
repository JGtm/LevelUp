// boot_accounts_pool_test.go — le démarrage n'attend plus la résolution des comptes : le pool de l'auto-sync
// naît vide, se remplit en tâche de fond, s'arrête avec le serveur, et les boucles qui ne se rejouent pas
// avant longtemps attendent la fin de la résolution avant leur première passe.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/pool"
)

// resolveurLent : doublure de pool.Resolver qui met `delai` à résoudre chaque compte, en respectant son
// contexte, et échoue pour les comptes de `enEchec`.
type resolveurLent struct {
	delai   time.Duration
	enEchec map[string]bool

	mu       sync.Mutex
	enVol    int
	annulees int
}

func (r *resolveurLent) Resolve(ctx context.Context, src pool.CredentialSource) (*pool.ResolvedTokens, error) {
	r.mu.Lock()
	r.enVol++
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.enVol--
		r.mu.Unlock()
	}()
	select {
	case <-time.After(r.delai):
	case <-ctx.Done():
		r.mu.Lock()
		r.annulees++
		r.mu.Unlock()
		return nil, ctx.Err()
	}
	if r.enCas(src.Gamertag) {
		return nil, errors.New("AADSTS70000: refresh token révoqué")
	}
	return &pool.ResolvedTokens{
		Gamertag:  src.Gamertag,
		XUID:      src.XUID,
		Tokens:    &domain.HaloTokens{SpartanToken: "spartan_" + src.Gamertag},
		ExpiresAt: time.Now().Add(time.Hour),
		Source:    src.Source,
	}, nil
}

func (r *resolveurLent) enCas(gamertag string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enEchec[gamertag]
}

func (r *resolveurLent) Refresh(context.Context, string) (*pool.ResolvedTokens, error) {
	return nil, errors.New("Refresh non attendu")
}

func (r *resolveurLent) etat() (enVol, annulees int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enVol, r.annulees
}

func douzeComptes() []pool.CredentialSource {
	out := make([]pool.CredentialSource, 0, 12)
	for i := 1; i <= 12; i++ {
		out = append(out, pool.CredentialSource{
			Gamertag: fmt.Sprintf("Compte%02d", i), TitleSlug: "halo_infinite", XUID: fmt.Sprintf("%d", 2000+i),
			RefreshToken: fmt.Sprintf("rt-%d", i), Source: "watcher_oauth",
		})
	}
	return out
}

// TestNewAutoSyncPool_LePortNAttendPasLaResolution : douze comptes à 300 ms (six en parallèle) : avant, le
// démarrage attendait deux vagues ; maintenant le pool est rendu tout de suite, vide, et se remplit.
func TestNewAutoSyncPool_LePortNAttendPasLaResolution(t *testing.T) {
	delai := 300 * time.Millisecond

	// AVANT : la construction synchrone d'origine (mêmes options, sans résolution de fond).
	debut := time.Now()
	ancien, err := pool.NewPool(context.Background(), &resolveurLent{delai: delai}, douzeComptes(),
		pool.PoolOptions{PerTokenRPS: 5, AllowEmpty: true})
	avant := time.Since(debut)
	if err != nil {
		t.Fatal(err)
	}
	defer ancien.Close()

	// APRÈS : ce que le démarrage appelle.
	r := &resolveurLent{delai: delai}
	debut = time.Now()
	p := newAutoSyncPool(context.Background(), r, douzeComptes())
	apres := time.Since(debut)
	if p == nil {
		t.Fatal("newAutoSyncPool a rendu nil")
	}
	defer p.Close()

	t.Logf("MESURE temps avant l'ouverture, 12 comptes à %v : avant %v, après %v", delai, avant, apres)
	if avant < 2*delai-50*time.Millisecond {
		t.Errorf("construction d'origine = %v, attendu deux vagues de %v", avant, delai)
	}
	if apres > delai/2 {
		t.Errorf("newAutoSyncPool a mis %v : il attend la résolution des comptes", apres)
	}
	if got := p.Size(); got == 12 {
		t.Errorf("le pool est déjà plein à la création : la résolution n'est pas en tâche de fond")
	}
	if !pool.WaitResolved(context.Background(), p) || p.Size() != 12 {
		t.Errorf("après la résolution : taille %d, attendu 12", p.Size())
	}
}

// TestNewAutoSyncPool_ResolutionEnEchec : tous les comptes échouent : pas d'erreur fatale, un pool vide qui
// se remplira à chaud, chaque échec journalisé par le pool (cf. paquet pool).
func TestNewAutoSyncPool_ResolutionEnEchec(t *testing.T) {
	r := &resolveurLent{delai: time.Millisecond, enEchec: map[string]bool{}}
	for _, src := range douzeComptes() {
		r.enEchec[src.Gamertag] = true
	}
	p := newAutoSyncPool(context.Background(), r, douzeComptes())
	if p == nil {
		t.Fatal("pool nil : l'auto-sync serait condamné jusqu'au redémarrage")
	}
	defer p.Close()
	if !pool.WaitResolved(context.Background(), p) {
		t.Fatal("la résolution ne se termine pas")
	}
	if p.Size() != 0 {
		t.Errorf("taille = %d, attendu 0 : tous les comptes ont échoué", p.Size())
	}
	// Le pool vide se remplit à chaud (connexion d'un joueur, re-scan).
	r.mu.Lock()
	r.enEchec = map[string]bool{}
	r.mu.Unlock()
	if err := p.AddOrUpdateSource(context.Background(), douzeComptes()[0]); err != nil {
		t.Errorf("AddOrUpdateSource sur le pool vide : %v", err)
	}
	if p.Size() != 1 {
		t.Errorf("taille = %d après l'ajout à chaud, attendu 1", p.Size())
	}
}

// TestNewAutoSyncPool_ArretPendantLaResolution : l'annulation du contexte d'arrêt interrompt la résolution :
// aucune résolution ne reste en vol, aucune n'est lancée ensuite, et l'attente de fin rend la main.
func TestNewAutoSyncPool_ArretPendantLaResolution(t *testing.T) {
	r := &resolveurLent{delai: time.Minute}
	ctx, arreter := context.WithCancel(context.Background())
	p := newAutoSyncPool(ctx, r, douzeComptes())
	defer p.Close()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if n, _ := r.etat(); n > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	arreter()

	fini := make(chan bool, 1)
	go func() { fini <- pool.WaitResolved(context.Background(), p) }()
	select {
	case ok := <-fini:
		if !ok {
			t.Error("WaitResolved = false après l'arrêt")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("la résolution ne s'arrête pas avec le contexte d'arrêt")
	}
	if n, annulees := r.etat(); n != 0 || annulees == 0 {
		t.Errorf("en vol = %d, annulées = %d : des résolutions survivent à l'arrêt", n, annulees)
	}
}

// TestAfterAccountsResolved : une boucle différée ne part qu'une fois les comptes résolus, part tout de
// suite sans résolution de fond, et ne part pas si l'arrêt vient pendant l'attente.
func TestAfterAccountsResolved(t *testing.T) {
	t.Run("attend la fin de la résolution", func(t *testing.T) {
		p := newAutoSyncPool(context.Background(), &resolveurLent{delai: 150 * time.Millisecond}, douzeComptes()[:2])
		defer p.Close()
		var tailleAuDepart int
		debut := time.Now()
		afterAccountsResolved(p, func(context.Context) { tailleAuDepart = p.Size() })(context.Background())
		if tailleAuDepart != 2 {
			t.Errorf("la boucle est partie avec %d compte(s) dans le pool, attendu 2", tailleAuDepart)
		}
		if time.Since(debut) < 100*time.Millisecond {
			t.Error("la boucle n'a pas attendu la résolution")
		}
	})
	t.Run("sans pool : part tout de suite", func(t *testing.T) {
		parti := false
		afterAccountsResolved(nil, func(context.Context) { parti = true })(context.Background())
		if !parti {
			t.Error("la boucle ne part pas sans pool")
		}
	})
	t.Run("arrêt pendant l'attente : ne part pas", func(t *testing.T) {
		p := newAutoSyncPool(context.Background(), &resolveurLent{delai: time.Minute}, douzeComptes()[:1])
		defer p.Close()
		ctx, arreter := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer arreter()
		parti := false
		afterAccountsResolved(p, func(context.Context) { parti = true })(ctx)
		if parti {
			t.Error("la boucle est partie alors que l'arrêt est demandé")
		}
	})
}

// Arrêt demandé alors que la résolution est déjà terminée : les deux canaux de WaitResolved sont prêts, le select
// en choisit un au hasard. La boucle ne doit jamais partir avec un contexte déjà annulé.
func TestAfterAccountsResolved_ContexteDejaAnnule(t *testing.T) {
	p := newAutoSyncPool(context.Background(), &resolveurLent{delai: time.Millisecond}, douzeComptes()[:1])
	defer p.Close()
	pool.WaitResolved(context.Background(), p)
	ctx, arreter := context.WithCancel(context.Background())
	arreter()
	for i := 0; i < 300; i++ {
		afterAccountsResolved(p, func(context.Context) { t.Fatal("la boucle est partie avec un contexte annulé") })(ctx)
	}
}
