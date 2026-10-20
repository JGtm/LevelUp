package pool

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// pool_boot_resolve_test.go — résolution parallèle des comptes au démarrage (NewPool).
//
// Ce que ces tests tiennent : la durée de construction du pool est celle du compte le plus
// lent et non la somme ; le parc obtenu (sources tentées, slots, ordre, plafond MaxSize) est
// celui de la résolution en série ; l'échec d'un compte n'affecte pas les autres ; deux
// sources au même refresh token ne sont jamais échangées en même temps.

// timedResolver : doublure de Resolver qui attend delay(gamertag) avant de répondre, échoue
// pour les gamertags de failing, et mesure la concurrence (globale et par refresh token).
type timedResolver struct {
	delay   func(gamertag string) time.Duration
	failing map[string]bool

	mu            sync.Mutex
	attempted     []string
	inFlight      int
	maxInFlight   int
	tokenInFlight map[string]int
	tokenOverlap  bool
}

func newTimedResolver(delay func(string) time.Duration, failing ...string) *timedResolver {
	f := make(map[string]bool, len(failing))
	for _, gt := range failing {
		f[gt] = true
	}
	return &timedResolver{delay: delay, failing: f, tokenInFlight: make(map[string]int)}
}

func (r *timedResolver) Resolve(_ context.Context, src CredentialSource) (*ResolvedTokens, error) {
	r.mu.Lock()
	r.attempted = append(r.attempted, src.Gamertag)
	r.inFlight++
	if r.inFlight > r.maxInFlight {
		r.maxInFlight = r.inFlight
	}
	r.tokenInFlight[src.RefreshToken]++
	if r.tokenInFlight[src.RefreshToken] > 1 {
		r.tokenOverlap = true
	}
	r.mu.Unlock()

	time.Sleep(r.delay(src.Gamertag))

	r.mu.Lock()
	r.inFlight--
	r.tokenInFlight[src.RefreshToken]--
	r.mu.Unlock()

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

func (r *timedResolver) Refresh(_ context.Context, gamertag string) (*ResolvedTokens, error) {
	return nil, fmt.Errorf("Refresh(%s) non attendu au démarrage", gamertag)
}

func (r *timedResolver) attemptedSet() map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]bool, len(r.attempted))
	for _, gt := range r.attempted {
		out[gt] = true
	}
	return out
}

// sequentialReference rend ce que donnait la boucle en série : sources tentées et slots
// retenus (dans l'ordre alphabétique), pour un plafond maxSize.
func sequentialReference(gamertags []string, failing map[string]bool, maxSize int) (attempted map[string]bool, slots []string) {
	sorted := append([]string(nil), gamertags...)
	sort.Strings(sorted)
	attempted = make(map[string]bool)
	for _, gt := range sorted {
		if maxSize > 0 && len(slots) == maxSize {
			break
		}
		attempted[gt] = true
		if !failing[gt] {
			slots = append(slots, gt)
		}
	}
	return attempted, slots
}

func slotGamertags(t *testing.T, p Pool) []string {
	t.Helper()
	impl, ok := p.(*poolImpl)
	if !ok {
		t.Fatalf("pool de type %T, attendu *poolImpl", p)
	}
	impl.slotMu.RLock()
	defer impl.slotMu.RUnlock()
	out := make([]string, 0, len(impl.slots))
	for _, s := range impl.slots {
		out = append(out, s.gamertag)
	}
	return out
}

func gamertagsN(n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("Compte%02d", i))
	}
	return out
}

func TestNewPool_ResolutionParallele_DureeDuPlusLentPasLaSomme(t *testing.T) {
	// Nombre de comptes FIXE : le dériver de bootResolveConcurrency ferait passer le test
	// même en série (bootResolveConcurrency = 1 → un seul compte).
	const perAccount = 150 * time.Millisecond
	gts := gamertagsN(7)
	resolver := newTimedResolver(func(string) time.Duration { return perAccount })

	start := time.Now()
	p, err := NewPool(context.Background(), resolver, sourcesFor(gts...), PoolOptions{})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	// Somme en série : 7 × 150 ms = 1 050 ms. En parallèle (6 en vol) : deux vagues, ~300 ms ;
	// la borne laisse de la marge à un poste chargé sans jamais accepter la somme.
	if limit := 4 * perAccount; elapsed >= limit {
		t.Errorf("construction du pool en %v pour %d comptes de %v chacun : attendu < %v "+
			"(le plus lent, pas la somme)", elapsed, len(gts), perAccount, limit)
	}
	if got := p.Size(); got != len(gts) {
		t.Errorf("Size() = %d, attendu %d", got, len(gts))
	}
}

func TestNewPool_ResolutionParallele_ConcurrenceBornee(t *testing.T) {
	gts := gamertagsN(3*bootResolveConcurrency + 1)
	resolver := newTimedResolver(func(string) time.Duration { return 20 * time.Millisecond })

	p, err := NewPool(context.Background(), resolver, sourcesFor(gts...), PoolOptions{})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	if resolver.maxInFlight > bootResolveConcurrency {
		t.Errorf("%d résolutions simultanées, borne = %d", resolver.maxInFlight, bootResolveConcurrency)
	}
	if resolver.maxInFlight < 2 {
		t.Errorf("%d résolution(s) simultanée(s) au plus : la résolution n'est pas parallèle",
			resolver.maxInFlight)
	}
	if got := p.Size(); got != len(gts) {
		t.Errorf("Size() = %d, attendu %d", got, len(gts))
	}
}

// TestNewPool_ResolutionParallele_MemeParcQuEnSerie : quel que soit l'ordre dans lequel les
// résolutions se terminent (les dernières sources répondent les premières), le parc et les
// sources tentées sont ceux de la boucle en série, pour chaque plafond.
func TestNewPool_ResolutionParallele_MemeParcQuEnSerie(t *testing.T) {
	gts := []string{"Hector", "Bianca", "Chocoboflor", "Gaston", "Alice", "DankerGlue", "Fabien", "Edgar", "Ines"}
	failing := map[string]bool{"Chocoboflor": true, "Fabien": true, "Alice": true}
	failingList := []string{"Chocoboflor", "Fabien", "Alice"}
	rank := map[string]int{}
	sorted := append([]string(nil), gts...)
	sort.Strings(sorted)
	for i, gt := range sorted {
		rank[gt] = i
	}
	// Les dernières sources répondent les premières : l'ordre d'achèvement est l'inverse
	// de l'ordre de lancement.
	delay := func(gt string) time.Duration { return time.Duration(len(sorted)-rank[gt]) * 4 * time.Millisecond }

	for _, maxSize := range []int{0, 1, 2, 3, 5, 6, 20} {
		t.Run(fmt.Sprintf("MaxSize=%d", maxSize), func(t *testing.T) {
			resolver := newTimedResolver(delay, failingList...)
			p, err := NewPool(context.Background(), resolver, sourcesFor(gts...), PoolOptions{MaxSize: maxSize})
			if err != nil {
				t.Fatalf("NewPool: %v", err)
			}
			defer p.Close()

			wantAttempted, wantSlots := sequentialReference(gts, failing, maxSize)
			if got := slotGamertags(t, p); fmt.Sprint(got) != fmt.Sprint(wantSlots) {
				t.Errorf("slots = %v, attendu (série) %v", got, wantSlots)
			}
			gotAttempted := resolver.attemptedSet()
			if fmt.Sprint(sortedKeys(gotAttempted)) != fmt.Sprint(sortedKeys(wantAttempted)) {
				t.Errorf("sources tentées = %v, attendu (série) %v — une résolution réseau de plus "+
					"qu'en série", sortedKeys(gotAttempted), sortedKeys(wantAttempted))
			}
		})
	}
}

func TestNewPool_ResolutionParallele_UnEchecNAffectePasLesAutres(t *testing.T) {
	gts := []string{"Alice", "Bianca", "Chocoboflor", "DankerGlue"}
	// Le compte en échec est aussi le plus lent : les autres ne l'attendent pas pour réussir.
	delay := func(gt string) time.Duration {
		if gt == "Bianca" {
			return 80 * time.Millisecond
		}
		return 10 * time.Millisecond
	}
	resolver := newTimedResolver(delay, "Bianca")
	p, err := NewPool(context.Background(), resolver, sourcesFor(gts...), PoolOptions{})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	if got, want := slotGamertags(t, p), []string{"Alice", "Chocoboflor", "DankerGlue"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("slots = %v, attendu %v", got, want)
	}
	for _, gt := range []string{"Alice", "Chocoboflor", "DankerGlue"} {
		if _, err := p.Acquire(context.Background(), PolicyPinnedPlayer, gt); err != nil {
			t.Errorf("Acquire(%s) : %v", gt, err)
		}
	}
	if p.HasPlayer("Bianca") {
		t.Error("le compte en échec ne doit pas avoir de slot")
	}
}

// TestNewPool_ResolutionParallele_JamaisDeuxEchangesDuMemeJeton : deux sources au même
// refresh token (un même compte déclaré deux fois) sont résolues l'une après l'autre, dans
// l'ordre alphabétique, pendant que les autres comptes avancent en parallèle.
func TestNewPool_ResolutionParallele_JamaisDeuxEchangesDuMemeJeton(t *testing.T) {
	sources := sourcesFor("Alice", "alice", "Bianca", "Chocoboflor")
	sources[1].RefreshToken = sources[0].RefreshToken // même compte, gamertag écrit autrement
	resolver := newTimedResolver(func(string) time.Duration { return 30 * time.Millisecond })

	p, err := NewPool(context.Background(), resolver, sources, PoolOptions{})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	if resolver.tokenOverlap {
		t.Error("deux échanges simultanés du même refresh token")
	}
	if resolver.maxInFlight < 2 {
		t.Errorf("les autres comptes doivent avancer en parallèle (max en vol = %d)", resolver.maxInFlight)
	}
	pos := map[string]int{}
	for i, gt := range resolver.attempted {
		pos[gt] = i
	}
	if pos["Alice"] > pos["alice"] {
		t.Errorf("ordre de résolution du jeton partagé = %v, attendu Alice avant alice", resolver.attempted)
	}
	if got := p.Size(); got != 4 {
		t.Errorf("Size() = %d, attendu 4", got)
	}
}

func TestPreviousSameRefreshToken(t *testing.T) {
	sources := []CredentialSource{
		{Gamertag: "A", RefreshToken: "rt1"},
		{Gamertag: "B", RefreshToken: ""},
		{Gamertag: "C", RefreshToken: "rt1"},
		{Gamertag: "D", RefreshToken: ""},
		{Gamertag: "E", RefreshToken: "rt1"},
		{Gamertag: "F", RefreshToken: "rt2"},
	}
	got := previousSameRefreshToken(sources)
	want := []int{-1, -1, 0, -1, 2, -1}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("previousSameRefreshToken = %v, attendu %v (sans jeton : n'attend personne)", got, want)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestNewPool_MaxSizeNegatif_SansPlafond : un plafond négatif (`--token-pool-size -1`, passé
// tel quel par le CLI) veut dire « sans plafond », comme 0 : tous les comptes sont résolus et
// chaque slot porte ses tokens. Jamais un slot sans tokens compté comme succès.
func TestNewPool_MaxSizeNegatif_SansPlafond(t *testing.T) {
	gts := []string{"Alice", "Bianca", "Chocoboflor"}
	resolver := newTimedResolver(func(string) time.Duration { return 5 * time.Millisecond })
	p, err := NewPool(context.Background(), resolver, sourcesFor(gts...), PoolOptions{MaxSize: -1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer p.Close()

	if got := resolver.attemptedSet(); len(got) != len(gts) {
		t.Errorf("sources tentées = %v, attendu les %d (plafond négatif = sans plafond)", sortedKeys(got), len(gts))
	}
	if got := slotGamertags(t, p); fmt.Sprint(got) != fmt.Sprint(gts) {
		t.Errorf("slots = %v, attendu %v", got, gts)
	}
	for _, s := range p.(*poolImpl).slots {
		if s.resolved == nil || s.resolved.Tokens == nil {
			t.Errorf("slot %s sans tokens : une source non résolue comptée comme succès", s.gamertag)
		}
	}
}

// TestResolveBootSources_SourceNonLanceeJamaisSucces : appelé sans la normalisation de
// NewPool, un plafond négatif ne lance rien ; aucune source non lancée ne doit ressortir
// comme résolue.
func TestResolveBootSources_SourceNonLanceeJamaisSucces(t *testing.T) {
	resolver := newTimedResolver(func(string) time.Duration { return 0 })
	for _, rs := range resolveBootSources(context.Background(), resolver, sourcesFor("Alice", "Bianca"), -1) {
		if rs.resolved == nil {
			t.Errorf("source %s rendue comme résolue sans tokens", rs.src.Gamertag)
		}
	}
}
