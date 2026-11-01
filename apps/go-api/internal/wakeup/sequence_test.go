package wakeup

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/clock"
)

// fakeTokens : une vérification de jetons scriptée ; chaque appel consomme l'issue suivante (la
// dernière se répète) et note les xuid demandés.
type fakeTokens struct {
	reports []TokenReport
	asked   [][]string
}

func (f *fakeTokens) Check(_ context.Context, only []string) TokenReport {
	f.asked = append(f.asked, only)
	i := len(f.asked) - 1
	if i >= len(f.reports) {
		i = len(f.reports) - 1
	}
	return f.reports[i]
}

// fakeSync : un ordonnanceur dont l'âge de la dernière synchronisation est fixé par le test.
type fakeSync struct {
	last     time.Time
	interval time.Duration
	requests []string
	sawNow   []time.Time
}

func (f *fakeSync) CatchUpDue(now time.Time) (bool, time.Time, time.Duration) {
	f.sawNow = append(f.sawNow, now)
	return now.Sub(f.last) > f.interval, f.last, f.interval
}

func (f *fakeSync) RequestCatchUp(reason string) bool {
	f.requests = append(f.requests, reason)
	return true
}

var (
	alive       = TokenReport{Results: []TokenResult{{XUID: "100", Outcome: TokenAlive}}}
	noNetwork   = TokenReport{Results: []TokenResult{{XUID: "100", Outcome: TokenUnreachable}, {XUID: "200", Outcome: TokenUnreachable}}}
	partialNet  = TokenReport{Results: []TokenResult{{XUID: "100", Outcome: TokenAlive}, {XUID: "200", Outcome: TokenUnreachable}}}
	noProfile   = TokenReport{}
	noon        = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	hourly      = time.Hour
	newSequence = func(tokens *fakeTokens, sync *fakeSync) *Sequence {
		return &Sequence{Tokens: tokens, Sync: sync, Clock: clock.Fake{T: noon}, Sleep: NewSleepDetector(newFakeClocks())}
	}
)

// TestWake_RattrapageSeulementSiEnRetard : au lancement, le premier tick est immédiat quand la dernière
// synchronisation est plus vieille que l'intervalle, sinon il reste à l'heure prévue — l'horloge est
// injectée.
func TestWake_RattrapageSeulementSiEnRetard(t *testing.T) {
	cas := []struct {
		name     string
		age      time.Duration
		wantReqs []string
	}{
		{"à l'heure (10 minutes)", 10 * time.Minute, nil},
		{"en retard (8 heures)", 8 * time.Hour, []string{ReasonLaunch}},
		{"jamais synchronisé (date zéro)", -1, []string{ReasonLaunch}},
	}
	for _, c := range cas {
		t.Run(c.name, func(t *testing.T) {
			sync := &fakeSync{last: noon.Add(-c.age), interval: hourly}
			if c.age < 0 {
				sync.last = time.Time{}
			}
			s := newSequence(&fakeTokens{reports: []TokenReport{alive}}, sync)
			s.Wake(context.Background(), ReasonLaunch)

			if len(sync.requests) != len(c.wantReqs) {
				t.Fatalf("demandes = %v, attendu %v", sync.requests, c.wantReqs)
			}
			if len(sync.sawNow) != 1 || !sync.sawNow[0].Equal(noon) {
				t.Errorf("l'âge n'est pas mesuré à l'horloge injectée : %v", sync.sawNow)
			}
		})
	}
}

// TestWake_LesJetonsPassentAvantLeRattrapage : la vérification des jetons se fait d'abord, une seule
// fois, sur tous les profils.
func TestWake_LesJetonsPassentAvantLeRattrapage(t *testing.T) {
	tokens := &fakeTokens{reports: []TokenReport{alive}}
	sync := &fakeSync{last: noon.Add(-8 * time.Hour), interval: hourly}
	newSequence(tokens, sync).Wake(context.Background(), ReasonLaunch)

	if len(tokens.asked) != 1 || tokens.asked[0] != nil {
		t.Errorf("vérifications = %v, attendu une seule sur tous les profils", tokens.asked)
	}
}

// TestWake_SansProfilSuiviLeRattrapageSuit : aucun profil à vérifier n'empêche pas la suite.
func TestWake_SansProfilSuiviLeRattrapageSuit(t *testing.T) {
	sync := &fakeSync{last: noon.Add(-8 * time.Hour), interval: hourly}
	newSequence(&fakeTokens{reports: []TokenReport{noProfile}}, sync).Wake(context.Background(), ReasonLaunch)
	if len(sync.requests) != 1 {
		t.Errorf("demandes = %v, attendu 1", sync.requests)
	}
}

// TestWake_ReseauAbsent : sans réponse de Microsoft, la synchronisation attend avec les jetons ; aux
// tics suivants (de plus en plus espacés) seuls les joueurs sans réponse sont revérifiés, et le
// rattrapage part dès que le réseau revient.
func TestWake_ReseauAbsent(t *testing.T) {
	tokens := &fakeTokens{reports: []TokenReport{noNetwork, noNetwork, alive}}
	sync := &fakeSync{last: noon.Add(-8 * time.Hour), interval: hourly}
	s := newSequence(tokens, sync)
	ctx := context.Background()

	s.Wake(ctx, ReasonResume)
	if len(sync.requests) != 0 {
		t.Fatalf("rattrapage lancé sans réseau : %v", sync.requests)
	}

	// Tic 1 : reprise (première attente d'un tic) ; le réseau est toujours absent.
	s.onTick(ctx)
	// Tic 2 : l'attente a doublé (2 tics), donc rien encore ; tic 3 : reprise, le réseau est revenu.
	s.onTick(ctx)
	if len(tokens.asked) != 2 {
		t.Fatalf("vérifications après 2 tics = %d, attendu 2 (lancement + une reprise)", len(tokens.asked))
	}
	s.onTick(ctx)

	if len(tokens.asked) != 3 {
		t.Fatalf("vérifications = %d, attendu 3", len(tokens.asked))
	}
	for i, only := range tokens.asked[1:] {
		if len(only) != 2 || only[0] != "100" || only[1] != "200" {
			t.Errorf("reprise %d : xuid demandés = %v, attendu les deux joueurs sans réponse", i+1, only)
		}
	}
	if len(sync.requests) != 1 || sync.requests[0] != ReasonResume {
		t.Errorf("demandes = %v, attendu un rattrapage au retour du réseau", sync.requests)
	}

	s.onTick(ctx)
	s.onTick(ctx)
	if len(tokens.asked) != 3 {
		t.Errorf("la reprise continue après le retour du réseau : %d vérifications", len(tokens.asked))
	}
}

// TestWake_ReseauPartiel : un joueur répond, l'autre pas — le réseau est là, le rattrapage part et
// seul le muet sera revérifié.
func TestWake_ReseauPartiel(t *testing.T) {
	tokens := &fakeTokens{reports: []TokenReport{partialNet, alive}}
	sync := &fakeSync{last: noon.Add(-8 * time.Hour), interval: hourly}
	s := newSequence(tokens, sync)
	ctx := context.Background()

	s.Wake(ctx, ReasonLaunch)
	if len(sync.requests) != 1 {
		t.Fatalf("demandes = %v, attendu 1", sync.requests)
	}
	s.onTick(ctx)
	if len(tokens.asked) != 2 || len(tokens.asked[1]) != 1 || tokens.asked[1][0] != "200" {
		t.Errorf("reprise = %v, attendu le seul joueur 200", tokens.asked)
	}
	if len(sync.requests) != 1 {
		t.Errorf("rattrapage redemandé à la reprise : %v", sync.requests)
	}
}

// TestRun_SortieDeVeille : après le lancement, un tic qui voit l'horloge murale devancer l'horloge
// monotone de plus de 2 minutes rejoue toute la séquence (jetons puis rattrapage) ; un tic ordinaire
// non.
func TestRun_SortieDeVeille(t *testing.T) {
	clocks := newFakeClocks()
	tokens := &fakeTokens{reports: []TokenReport{alive}}
	sync := &fakeSync{last: noon.Add(-10 * time.Minute), interval: hourly} // à l'heure au lancement
	ticks := make(chan time.Time)
	s := &Sequence{Tokens: tokens, Sync: sync, Clock: clock.Fake{T: noon}, Sleep: NewSleepDetector(clocks), Ticks: ticks}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	tick := func() { ticks <- time.Time{} }

	clocks.advance(SleepTick)
	tick() // tic ordinaire
	clocks.advance(SleepTick)
	clocks.sleep(8 * time.Hour)
	sync.last = noon.Add(-8 * time.Hour) // pendant la veille, la dernière sync a vieilli
	tick()                               // sortie de veille
	tick()                               // un tic de plus : le canal non tamponné garantit que le précédent est traité

	cancel()
	<-done

	if len(tokens.asked) != 2 {
		t.Errorf("vérifications des jetons = %d, attendu 2 (lancement + sortie de veille)", len(tokens.asked))
	}
	if len(sync.requests) != 1 || sync.requests[0] != ReasonResume {
		t.Errorf("demandes = %v, attendu un seul rattrapage, à la sortie de veille", sync.requests)
	}
}
