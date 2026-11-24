package pool

// pool_background_test.go — résolution des comptes en tâche de fond (PoolOptions.ResolveInBackground).
//
// Ce que ces tests tiennent : NewPool rend la main tout de suite, le pool se remplit ensuite ;
// une acquisition attend le compte visé (épinglée) ou le premier compte partageable (publique) au
// lieu d'échouer ; un compte qui n'a pas consenti à la mise en commun n'est ni servi aux autres ni
// ouvert à un emprunt pendant la résolution, et un retrait de consentement fait pendant la
// résolution survit ; chaque échec de résolution est journalisé ; l'arrêt annule la résolution et
// n'en laisse aucune goroutine.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ctxResolver : doublure de Resolver qui attend delay(gamertag) EN RESPECTANT le contexte, échoue pour
// les gamertags de failing et compte les résolutions en vol.
type ctxResolver struct {
	delay   func(gamertag string) time.Duration
	failing map[string]bool

	mu        sync.Mutex
	inFlight  int
	cancelled int
	attempted []string
	overlap   bool
	byToken   map[string]int
}

func newCtxResolver(delay func(string) time.Duration, failing ...string) *ctxResolver {
	f := make(map[string]bool, len(failing))
	for _, gt := range failing {
		f[gt] = true
	}
	return &ctxResolver{delay: delay, failing: f, byToken: make(map[string]int)}
}

func (r *ctxResolver) Resolve(ctx context.Context, src CredentialSource) (*ResolvedTokens, error) {
	r.mu.Lock()
	r.attempted = append(r.attempted, src.Gamertag)
	r.inFlight++
	r.byToken[src.RefreshToken]++
	if src.RefreshToken != "" && r.byToken[src.RefreshToken] > 1 {
		r.overlap = true
	}
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.inFlight--
		r.byToken[src.RefreshToken]--
		r.mu.Unlock()
	}()

	select {
	case <-time.After(r.delay(src.Gamertag)):
	case <-ctx.Done():
		r.mu.Lock()
		r.cancelled++
		r.mu.Unlock()
		return nil, ctx.Err()
	}
	if r.failing[src.Gamertag] {
		return nil, errors.New("AADSTS70000: refresh token révoqué")
	}
	return &ResolvedTokens{
		Gamertag:  src.Gamertag,
		XUID:      src.XUID,
		Tokens:    &domain.HaloTokens{SpartanToken: "spartan_" + src.Gamertag},
		ExpiresAt: time.Now().Add(time.Hour),
		Source:    src.Source,
	}, nil
}

func (r *ctxResolver) Refresh(_ context.Context, gamertag string) (*ResolvedTokens, error) {
	return nil, fmt.Errorf("Refresh(%s) non attendu", gamertag)
}

func (r *ctxResolver) stats() (inFlight, cancelled int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inFlight, r.cancelled
}

func bgSources(gamertags ...string) []CredentialSource {
	out := make([]CredentialSource, 0, len(gamertags))
	for i, gt := range gamertags {
		out = append(out, CredentialSource{
			Gamertag: gt, TitleSlug: "halo_infinite", XUID: fmt.Sprintf("%d", 1000+i),
			RefreshToken: "rt-" + gt, Source: "watcher_oauth",
		})
	}
	return out
}

func newBackgroundPool(t *testing.T, r Resolver, sources []CredentialSource, maxSize int) Pool {
	t.Helper()
	p, err := NewPool(context.Background(), r, sources, PoolOptions{
		MaxSize: maxSize, AllowEmpty: true, ResolveInBackground: true,
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("délai dépassé : %s", what)
}

// TestBackground_LaMainEstRenduToutDeSuite : avec douze comptes à 200 ms chacun (six en parallèle),
// la construction synchrone attend deux vagues ; en tâche de fond elle rend la main sans attendre, et le
// pool se remplit ensuite.
func TestBackground_LaMainEstRenduToutDeSuite(t *testing.T) {
	gts := []string{"A01", "A02", "A03", "A04", "A05", "A06", "A07", "A08", "A09", "A10", "A11", "A12"}
	delay := func(string) time.Duration { return 200 * time.Millisecond }

	debut := time.Now()
	sync1, err := NewPool(context.Background(), newCtxResolver(delay), bgSources(gts...), PoolOptions{AllowEmpty: true})
	avant := time.Since(debut)
	if err != nil {
		t.Fatal(err)
	}
	defer sync1.Close()

	debut = time.Now()
	p := newBackgroundPool(t, newCtxResolver(delay), bgSources(gts...), 0)
	apres := time.Since(debut)

	t.Logf("MESURE construction du pool, 12 comptes à 200 ms : synchrone %v, en tâche de fond %v", avant, apres)
	if apres > 100*time.Millisecond {
		t.Errorf("construction en tâche de fond = %v, attendu quasi immédiate", apres)
	}
	if avant < 350*time.Millisecond {
		t.Errorf("construction synchrone = %v, attendu deux vagues de 200 ms", avant)
	}
	if got := p.Size(); got != 0 && got != 12 {
		// Aucun compte n'a eu le temps de se résoudre : le pool est vide, ou déjà rempli sur une machine très lente.
		t.Logf("taille juste après la construction : %d", got)
	}
	waitFor(t, "les douze comptes entrent dans le pool", func() bool { return p.Size() == 12 })
}

// TestBackground_EpingleeAttendSonCompte : un compte épinglé encore en résolution est présent
// (HasPlayer) et son acquisition attend sa résolution au lieu d'échouer.
func TestBackground_EpingleeAttendSonCompte(t *testing.T) {
	p := newBackgroundPool(t, newCtxResolver(func(gt string) time.Duration {
		if gt == "Lent" {
			return 300 * time.Millisecond
		}
		return 0
	}), bgSources("Lent", "Vif"), 0)

	if !p.HasPlayer("Lent") {
		t.Fatal("HasPlayer(Lent) = false pendant la résolution, attendu true")
	}
	debut := time.Now()
	lease, err := p.Acquire(context.Background(), PolicyPinnedPlayer, "Lent")
	if err != nil {
		t.Fatalf("Acquire épinglée pendant la résolution : %v", err)
	}
	defer lease.Release()
	if lease.Gamertag != "Lent" || lease.Tokens == nil || lease.Tokens.SpartanToken != "spartan_Lent" {
		t.Errorf("lease = %+v, attendu le jeton de Lent", lease)
	}
	if d := time.Since(debut); d < 200*time.Millisecond {
		t.Errorf("l'acquisition est revenue en %v, avant la résolution du compte", d)
	}
}

// TestBackground_PubliqueAttendLePremier : l'acquisition publique rend le premier compte partageable résolu,
// sans attendre les plus lents.
func TestBackground_PubliqueAttendLePremier(t *testing.T) {
	p := newBackgroundPool(t, newCtxResolver(func(gt string) time.Duration {
		if gt == "Vif" {
			return 50 * time.Millisecond
		}
		return 2 * time.Second
	}), bgSources("Lent1", "Lent2", "Vif"), 0)

	debut := time.Now()
	lease, err := p.Acquire(context.Background(), PolicyAnyPublic, "")
	if err != nil {
		t.Fatalf("Acquire publique pendant la résolution : %v", err)
	}
	lease.Release()
	if lease.Gamertag != "Vif" {
		t.Errorf("compte servi = %q, attendu le premier résolu (Vif)", lease.Gamertag)
	}
	if d := time.Since(debut); d > time.Second {
		t.Errorf("attente = %v, attendu le temps du premier compte et non du dernier", d)
	}
}

// TestBackground_AcquisitionRespecteSonContexte : une acquisition qui attend rend la main quand son
// contexte expire, avec l'erreur du contexte.
func TestBackground_AcquisitionRespecteSonContexte(t *testing.T) {
	p := newBackgroundPool(t, newCtxResolver(func(string) time.Duration { return 5 * time.Second }),
		bgSources("Lent"), 0)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	for _, pol := range []AcquirePolicy{PolicyAnyPublic, PolicyPinnedPlayer} {
		if _, err := p.Acquire(ctx, pol, "Lent"); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("policy %v : err = %v, attendu context.DeadlineExceeded", pol, err)
		}
	}
}

// TestBackground_EchecJournaliseEtNeBloquePas : un compte dont la résolution échoue est journalisé, n'entre
// pas dans le pool, et son acquisition épinglée échoue sans attendre indéfiniment.
func TestBackground_EchecJournaliseEtNeBloquePas(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	ancien := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&lockedWriter{mu: &mu, w: &buf}, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(ancien)

	p := newBackgroundPool(t, newCtxResolver(func(string) time.Duration { return 10 * time.Millisecond }, "Mort"),
		bgSources("Mort", "Vivant"), 0)

	done := make(chan error, 1)
	go func() {
		_, err := p.Acquire(context.Background(), PolicyPinnedPlayer, "Mort")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Acquire du compte en échec a réussi")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Acquire du compte en échec n'est jamais revenue")
	}
	waitFor(t, "le compte vivant entre dans le pool", func() bool { return p.HasPlayer("Vivant") })
	if p.HasPlayer("Mort") {
		t.Error("le compte en échec est resté présent après sa résolution")
	}
	mu.Lock()
	logs := buf.String()
	mu.Unlock()
	if !strings.Contains(logs, "impossible de résoudre token au boot") || !strings.Contains(logs, "gamertag=Mort") {
		t.Errorf("l'échec de résolution n'est pas journalisé avec son gamertag :\n%s", logs)
	}
}

type lockedWriter struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(b []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(b)
}

// TestBackground_ArretPendantLaResolution : Close annule les résolutions en vol et les attend : aucune
// goroutine de résolution ne survit, et les acquisitions en attente rendent la main.
func TestBackground_ArretPendantLaResolution(t *testing.T) {
	r := newCtxResolver(func(string) time.Duration { return time.Minute })
	p, err := NewPool(context.Background(), r, bgSources("A", "B", "C", "D", "E", "F", "G", "H"),
		PoolOptions{AllowEmpty: true, ResolveInBackground: true})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "six résolutions en vol", func() bool { n, _ := r.stats(); return n == bootResolveConcurrency })

	attente := make(chan error, 1)
	go func() {
		_, err := p.Acquire(context.Background(), PolicyAnyPublic, "")
		attente <- err
	}()
	time.Sleep(30 * time.Millisecond)

	debut := time.Now()
	p.Close()
	if d := time.Since(debut); d > 2*time.Second {
		t.Errorf("Close a mis %v, attendu l'annulation immédiate", d)
	}
	if n, cancelled := r.stats(); n != 0 || cancelled != bootResolveConcurrency {
		t.Errorf("après Close : %d résolution(s) en vol, %d annulée(s), attendu 0 et %d", n, cancelled, bootResolveConcurrency)
	}
	select {
	case err := <-attente:
		if err == nil {
			t.Error("l'acquisition en attente a réussi après l'arrêt")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("l'acquisition en attente n'est pas revenue après Close")
	}
	r.mu.Lock()
	lances := len(r.attempted)
	r.mu.Unlock()
	if lances != bootResolveConcurrency {
		t.Errorf("%d résolutions lancées, attendu %d : aucune ne doit partir après l'arrêt", lances, bootResolveConcurrency)
	}
}

// TestBackground_ConsentementPendantLaResolution : la mise en commun se décide avant la résolution.
//   - un compte sans consentement est retenu (IsWithheld) dès le démarrage, donc il n'emprunte pas
//     les jetons des autres, et son jeton ne sert jamais le tour de rôle public ;
//   - un retrait de consentement fait PENDANT la résolution (SetShared) survit à son entrée dans le pool.
func TestBackground_ConsentementPendantLaResolution(t *testing.T) {
	src := bgSources("Prive", "Partage", "Retire")
	src[0].PoolExcluded = true // Prive n'a pas consenti
	p := newBackgroundPool(t, newCtxResolver(func(gt string) time.Duration {
		if gt == "Retire" {
			return 250 * time.Millisecond
		}
		return 20 * time.Millisecond
	}), src, 0)

	if !p.IsWithheld("Prive") {
		t.Error("IsWithheld(Prive) = false pendant la résolution : il emprunterait les jetons des autres")
	}
	if p.IsWithheld("Partage") {
		t.Error("IsWithheld(Partage) = true")
	}
	if !p.SetShared(src[2].XUID, false) {
		t.Fatal("SetShared sur un compte en cours de résolution n'a rien trouvé : le retrait serait perdu")
	}
	waitFor(t, "les trois comptes entrent dans le pool", func() bool { return p.Size() == 3 })

	if !p.IsWithheld("Retire") || !p.IsWithheld("Prive") {
		t.Error("un compte sans consentement est servi à la mise en commun après sa résolution")
	}
	// 30 tours de rôle public : seul Partage peut sortir.
	for i := 0; i < 30; i++ {
		lease, err := p.Acquire(context.Background(), PolicyAnyPublic, "")
		if err != nil {
			t.Fatalf("Acquire publique : %v", err)
		}
		if lease.Gamertag != "Partage" {
			t.Fatalf("le tour de rôle public a servi le jeton de %q, qui n'a pas consenti", lease.Gamertag)
		}
		lease.Release()
	}
	// Le propriétaire garde son propre jeton.
	for _, gt := range []string{"Prive", "Retire"} {
		l, err := p.Acquire(context.Background(), PolicyPinnedPlayer, gt)
		if err != nil || l.Gamertag != gt {
			t.Errorf("épinglée %s : lease=%v err=%v", gt, l, err)
		}
	}
}

// TestBackground_MemeJetonJamaisEnMemeTemps : deux comptes au même refresh token ne se résolvent jamais
// ensemble, en tâche de fond comme au démarrage synchrone (Microsoft fait tourner le jeton).
func TestBackground_MemeJetonJamaisEnMemeTemps(t *testing.T) {
	src := bgSources("A", "B", "C")
	src[1].RefreshToken = src[0].RefreshToken
	r := newCtxResolver(func(string) time.Duration { return 40 * time.Millisecond })
	p := newBackgroundPool(t, r, src, 0)
	waitFor(t, "trois comptes", func() bool { return p.Size() == 3 })
	if r.overlap {
		t.Error("deux comptes du même refresh token se sont résolus en même temps")
	}
}

// TestBackground_PlafondEtRescan : le plafond MaxSize tient en tâche de fond, et un ajout à chaud
// (re-scan) du même compte pendant sa résolution ne double pas son slot.
func TestBackground_PlafondEtRescan(t *testing.T) {
	r := newCtxResolver(func(string) time.Duration { return 30 * time.Millisecond })
	p := newBackgroundPool(t, r, bgSources("A", "B", "C", "D"), 2)
	waitFor(t, "le plafond est atteint", func() bool { return p.Size() == 2 })
	time.Sleep(150 * time.Millisecond)
	if got := p.Size(); got != 2 {
		t.Errorf("taille = %d, attendu le plafond de 2", got)
	}

	p2 := newBackgroundPool(t, newCtxResolver(func(string) time.Duration { return 100 * time.Millisecond }), bgSources("X", "Y"), 0)
	if err := p2.AddOrUpdateSource(context.Background(), bgSources("X")[0]); err != nil {
		t.Fatalf("AddOrUpdateSource pendant la résolution : %v", err)
	}
	waitFor(t, "X et Y", func() bool { return p2.Size() >= 2 })
	time.Sleep(150 * time.Millisecond)
	if got := p2.Size(); got != 2 {
		t.Errorf("taille = %d après ajout à chaud pendant la résolution, attendu 2 (pas de doublon)", got)
	}
}

// TestWaitResolved : l'attente de la fin de la résolution rend la main à la fin, ou au contexte ; sans
// résolution de fond, elle rend la main tout de suite.
func TestWaitResolved(t *testing.T) {
	p := newBackgroundPool(t, newCtxResolver(func(string) time.Duration { return 120 * time.Millisecond }),
		bgSources("A", "B"), 0)
	debut := time.Now()
	if !WaitResolved(context.Background(), p) {
		t.Error("WaitResolved = false avec un contexte vivant")
	}
	if d := time.Since(debut); d < 80*time.Millisecond {
		t.Errorf("WaitResolved est revenue en %v, avant la fin de la résolution", d)
	}
	if p.Size() != 2 {
		t.Errorf("taille = %d à la fin de la résolution, attendu 2", p.Size())
	}

	lent := newBackgroundPool(t, newCtxResolver(func(string) time.Duration { return 5 * time.Second }), bgSources("A"), 0)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if WaitResolved(ctx, lent) {
		t.Error("WaitResolved = true alors que la résolution n'est pas finie")
	}

	synchrone, err := NewPool(context.Background(), newCtxResolver(func(string) time.Duration { return 0 }),
		bgSources("A"), PoolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer synchrone.Close()
	if !WaitResolved(context.Background(), synchrone) || !WaitResolved(context.Background(), nil) {
		t.Error("WaitResolved doit rendre vrai tout de suite sans résolution de fond")
	}
}
