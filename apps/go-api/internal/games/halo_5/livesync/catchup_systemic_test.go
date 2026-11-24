package livesync

// catchup_systemic_test.go — une panne de l'API ne fait pas abandonner une page saine, et une page
// abandonnée est retentée après historystate.AbandonRetryAfter (trou noté, historique rouvert).

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"

	"github.com/JGtm/halo-api/halo5"
)

// setHistoryClock remplace l'horloge de historystate le temps du test ; le rendu avance d'une durée.
func setHistoryClock(t *testing.T) (advance func(time.Duration)) {
	t.Helper()
	now := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	old := historystate.Now
	historystate.Now = func() time.Time { return now }
	t.Cleanup(func() { historystate.Now = old })
	return func(d time.Duration) { now = now.Add(d) }
}

func pageErr(code int) error {
	return fmt.Errorf("GetPlayerMatches: %w", &halo5.HTTPError{StatusCode: code, URL: "u"})
}

// 429 ou réseau coupé pendant plus de maxCatchUpAttempts passes sur la page de la frontière : aucune
// tentative comptée, aucune page abandonnée ; l'API revenue, la fin se range et tout est rangé.
func TestRunDelta_Rattrapage_PanneDeLAPI_AucunAbandon(t *testing.T) {
	for name, outage := range map[string]error{
		"429":    pageErr(429),
		"réseau": fmt.Errorf("GetPlayerMatches: %w", &testNetErr{}),
	} {
		t.Run(name, func(t *testing.T) {
			w := newHistoryWorld(100)
			w.errAtStart[50] = outage
			before := observability.LoadCounter(metricAbandonedPages)
			for pass := 1; pass <= maxCatchUpAttempts+2; pass++ {
				res := w.run(t, domain.SyncOptions{})
				if w.marked != 0 || !res.HistoryPending {
					t.Fatalf("passe %d : fin rangée=%d pending=%v, want 0 et vrai", pass, w.marked, res.HistoryPending)
				}
				if w.streak.Attempts != 0 {
					t.Fatalf("passe %d : compte d'échecs %+v, want aucune tentative comptée", pass, w.streak)
				}
			}
			if got := observability.LoadCounter(metricAbandonedPages) - before; got != 0 || w.hole != nil {
				t.Fatalf("pages abandonnées +%d, trou noté %v, want 0 et aucun", got, w.hole != nil)
			}

			delete(w.errAtStart, 50) // l'API est revenue
			res := w.run(t, domain.SyncOptions{})
			if w.marked != 1 || res.HistoryPending || len(w.stored) != 100 {
				t.Errorf("après la panne : fin rangée=%d pending=%v rangés=%d, want 1, faux et 100", w.marked, res.HistoryPending, len(w.stored))
			}
		})
	}
}

// testNetErr : une erreur de liaison (net.Error) qui a dépassé son délai.
type testNetErr struct{}

func (*testNetErr) Error() string   { return "i/o timeout" }
func (*testNetErr) Timeout() bool   { return true }
func (*testNetErr) Temporary() bool { return true }

// Un 5xx durable sur une seule page est le défaut de cette page : abandonnée à la troisième passe.
func TestRunDelta_Rattrapage_5xxDurableSurUnePage_AbandonneeALaTroisiemePasse(t *testing.T) {
	w := newHistoryWorld(100)
	w.errAtStart[50] = pageErr(503)
	before := observability.LoadCounter(metricAbandonedPages)
	for pass := 1; pass <= maxCatchUpAttempts; pass++ {
		w.run(t, domain.SyncOptions{})
	}
	if got := observability.LoadCounter(metricAbandonedPages) - before; got != 1 || w.marked != 1 || w.hole == nil {
		t.Errorf("pages abandonnées +%d, fin rangée=%d, trou noté %v, want 1, 1 et oui", got, w.marked, w.hole != nil)
	}
}

// Une page où plus de 10 % des matchs échouent à la persistance (au moins cinq) est une passe de masse :
// aucune tentative, aucun abandon. Quelques matchs isolés en échec durable, eux, sont abandonnés.
func TestRunDelta_Rattrapage_PersistanceEnMasse_AucuneTentative(t *testing.T) {
	w := newHistoryWorld(60)
	for i := 0; i < 8; i++ {
		w.persistErr[fmt.Sprintf("m%04d", i)] = true
	}
	before := observability.LoadCounter(metricAbandonedPages)
	for pass := 1; pass <= maxCatchUpAttempts+1; pass++ {
		w.run(t, domain.SyncOptions{})
	}
	if got := observability.LoadCounter(metricAbandonedPages) - before; got != 0 || w.marked != 0 || w.streak.Attempts != 0 {
		t.Errorf("pages abandonnées +%d, fin rangée=%d, compte %+v, want 0, 0 et aucun", got, w.marked, w.streak)
	}
}

// Une page abandonnée est un trou noté : avant 24 h rien ne change, après, l'historique est rouvert, la
// liste relue depuis son début (connus sautés) et la page retentée.
func TestRunDelta_Rattrapage_PageAbandonneeRetenteApres24h(t *testing.T) {
	advance := setHistoryClock(t)
	w := newHistoryWorld(100)
	w.errAtStart[50] = errors.New("GetPlayerMatches: décodage")
	for pass := 1; pass <= maxCatchUpAttempts; pass++ {
		w.run(t, domain.SyncOptions{})
	}
	if w.marked != 1 || w.hole == nil || w.stored["m0050"] {
		t.Fatalf("préparation : fin rangée=%d, trou noté %v, m0050 rangé %v, want 1, oui et non", w.marked, w.hole != nil, w.stored["m0050"])
	}

	delete(w.errAtStart, 50) // la page est maintenant lisible
	advance(historystate.AbandonRetryAfter - time.Minute)
	w.run(t, domain.SyncOptions{})
	if w.reopened != 0 || w.stored["m0050"] {
		t.Fatalf("avant 24 h : réouvertures %d, m0050 rangé %v, want 0 et non", w.reopened, w.stored["m0050"])
	}

	advance(2 * time.Minute)
	res := w.run(t, domain.SyncOptions{})
	if w.reopened != 1 || !w.stored["m0050"] || len(w.stored) != 100 {
		t.Errorf("après 24 h : réouvertures %d, rangés %d, want 1 et 100 (page retentée)", w.reopened, len(w.stored))
	}
	if w.marked != 2 || res.HistoryPending || w.hole != nil {
		t.Errorf("fin rangée=%d pending=%v trou %v, want 2, faux et effacé", w.marked, res.HistoryPending, w.hole != nil)
	}
}

// Retentée en vain après 24 h : abandonnée de nouveau (nouveau trou daté de ce jour), la fin se range encore.
func TestRunDelta_Rattrapage_PageRetenteeEnVain_AbandonneeDeNouveau(t *testing.T) {
	advance := setHistoryClock(t)
	w := newHistoryWorld(100)
	w.errAtStart[50] = errors.New("GetPlayerMatches: décodage")
	for pass := 1; pass <= maxCatchUpAttempts; pass++ {
		w.run(t, domain.SyncOptions{})
	}
	advance(historystate.AbandonRetryAfter + time.Hour)
	for pass := 1; pass <= maxCatchUpAttempts+1; pass++ {
		w.run(t, domain.SyncOptions{})
	}
	if w.reopened != 1 || w.marked != 2 || w.hole == nil {
		t.Errorf("réouvertures %d, fin rangée=%d, trou noté %v, want 1, 2 et oui (nouveau trou)", w.reopened, w.marked, w.hole != nil)
	}
	if w.hole != nil && !w.hole.Equal(historystate.Now()) {
		t.Errorf("le nouveau trou date de %v, want de maintenant", w.hole)
	}
}
