// auto_sync_zero_insert_voies_test.go — l'alerte « zero-insert prolongé » ne voit pas que les matchs
// rangés par le cycle : un match arrivé par une autre voie (le déclencheur de fin de partie du suivi
// en direct, un import) a déjà été rangé quand le cycle passe, qui n'a plus rien à insérer. Le compteur
// lit donc aussi le dernier match connu du joueur ; l'alerte ne sonne que si aucun match n'est arrivé,
// par aucune voie, sur la fenêtre, puis une fois par épisode au lieu d'à chaque cycle.

package scheduler_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/scheduler"
)

// dernierMatchScripte : doublure de port.LastMatchReader qui rend, cycle après cycle, la valeur scriptée.
type dernierMatchScripte struct {
	mu     sync.Mutex
	valeur time.Time // zéro : le joueur n'a aucun match (absent du résultat)
	err    error
	appels int
	xuids  []string
}

func (d *dernierMatchScripte) fixer(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.valeur = t
}

func (d *dernierMatchScripte) LastMatchByXUID(_ context.Context, _, _ string, xuids []string) (map[string]time.Time, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.appels++
	d.xuids = append(d.xuids, xuids...)
	if d.err != nil {
		return map[string]time.Time{}, d.err
	}
	out := map[string]time.Time{}
	if !d.valeur.IsZero() {
		for _, x := range xuids {
			out[x] = d.valeur
		}
	}
	return out, nil
}

// capturerJournal remplace le journal par défaut le temps du test et rend ses lignes.
func capturerJournal(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var buf bytes.Buffer
	ancien := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&lockedBuf{mu: &mu, b: &buf}, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(ancien) })
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return strings.Split(buf.String(), "\n")
	}
}

type lockedBuf struct {
	mu *sync.Mutex
	b  *bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func alertesZeroInsert(lignes []string) int {
	n := 0
	for _, l := range lignes {
		if strings.Contains(l, "level=WARN") && strings.Contains(l, "zero-insert prolongé") {
			n++
		}
	}
	return n
}

func schedulerZeroInsert(t *testing.T, lecteur *dernierMatchScripte) *scheduler.AutoSyncScheduler {
	t.Helper()
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")
	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	if lecteur != nil {
		s.WithLastMatchReader(lecteur)
	}
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 0, MatchesSkipped: 25}}
	}
	return s
}

func compteur(s *scheduler.AutoSyncScheduler) int {
	return s.Snapshot().Players[0].ConsecutiveZeroInserts
}

// Des matchs rangés par une autre voie entre deux cycles : le cycle n'insère rien, mais le dernier match
// connu a avancé. Le compteur retombe à zéro et l'alerte ne sonne jamais, même sur beaucoup de cycles.
func TestZeroInsert_MatchRangeParUneAutreVoie_NeSonnePas(t *testing.T) {
	journal := capturerJournal(t)
	lecteur := &dernierMatchScripte{}
	s := schedulerZeroInsert(t, lecteur)
	debut := time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC)

	for cycle := 0; cycle < 3*scheduler.ConsecutiveZeroInsertWarnThreshold; cycle++ {
		// Un nouveau match est arrivé (déclencheur de fin de partie) avant chaque cycle.
		lecteur.fixer(debut.Add(time.Duration(cycle) * 30 * time.Minute))
		s.RunOnce(context.Background())
		if got := compteur(s); cycle > 0 && got != 0 {
			t.Fatalf("cycle %d : compteur = %d, attendu 0 : un match est arrivé par une autre voie", cycle, got)
		}
	}
	if n := alertesZeroInsert(journal()); n != 0 {
		t.Errorf("%d alerte(s) alors que des matchs arrivaient à chaque cycle, attendu 0", n)
	}
}

// Une vraie panne : ni le cycle ni aucune autre voie ne range de match. Une alerte au franchissement du
// seuil, une seule, puis le silence de l'épisode ; elle reprend après le délai de rappel.
func TestZeroInsert_VraiePanne_UneAlerteParEpisode(t *testing.T) {
	journal := capturerJournal(t)
	lecteur := &dernierMatchScripte{}
	lecteur.fixer(time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC)) // constant : aucun match ne vient
	s := schedulerZeroInsert(t, lecteur)

	seuil := scheduler.ConsecutiveZeroInsertWarnThreshold
	for cycle := 1; cycle < seuil; cycle++ {
		s.RunOnce(context.Background())
	}
	if n := alertesZeroInsert(journal()); n != 0 {
		t.Fatalf("%d alerte(s) avant le seuil, attendu 0", n)
	}
	s.RunOnce(context.Background()) // franchit le seuil
	if n := alertesZeroInsert(journal()); n != 1 {
		t.Fatalf("%d alerte(s) au franchissement du seuil, attendu 1", n)
	}
	for cycle := 0; cycle < scheduler.ZeroInsertRewarnEvery-1; cycle++ {
		s.RunOnce(context.Background())
	}
	if n := alertesZeroInsert(journal()); n != 1 {
		t.Errorf("%d alerte(s) pendant l'épisode, attendu 1 (pas une par cycle)", n)
	}
	s.RunOnce(context.Background()) // un délai de rappel après le franchissement
	if n := alertesZeroInsert(journal()); n != 2 {
		t.Errorf("%d alerte(s) après le délai de rappel, attendu 2", n)
	}
	if got := compteur(s); got < seuil {
		t.Errorf("compteur = %d : il doit rester lisible par le tableau de bord (>= seuil %d)", got, seuil)
	}

	// Un match arrive : l'épisode se clôt ; une nouvelle panne alerte de nouveau au seuil.
	lecteur.fixer(time.Date(2026, 11, 21, 20, 0, 0, 0, time.UTC))
	s.RunOnce(context.Background())
	if got := compteur(s); got != 0 {
		t.Fatalf("compteur = %d après l'arrivée d'un match, attendu 0", got)
	}
	avant := alertesZeroInsert(journal())
	for cycle := 0; cycle < seuil; cycle++ {
		s.RunOnce(context.Background())
	}
	if n := alertesZeroInsert(journal()); n != avant+1 {
		t.Errorf("%d alerte(s) pour la nouvelle panne, attendu une de plus (%d)", n-avant, 1)
	}
}

// Lecture du dernier match impossible : le compteur retombe sur la règle du cycle seul (aucune voie
// secondaire), sans erreur ni arrêt, et la dégradation est journalisée.
func TestZeroInsert_LectureEnEchec_RegleDuCycleSeul(t *testing.T) {
	journal := capturerJournal(t)
	lecteur := &dernierMatchScripte{err: errors.New("base partagée illisible")}
	s := schedulerZeroInsert(t, lecteur)
	for cycle := 1; cycle <= 3; cycle++ {
		s.RunOnce(context.Background())
		if got := compteur(s); got != cycle {
			t.Fatalf("cycle %d : compteur = %d, attendu %d", cycle, got, cycle)
		}
	}
	vu := false
	for _, l := range journal() {
		if strings.Contains(l, "dernier match connu illisible") {
			vu = true
		}
	}
	if !vu {
		t.Error("la lecture en échec du dernier match n'est pas journalisée")
	}
}

// Un match arrivé pendant un cycle sauté (suivi en direct actif) est vu au cycle suivant : la dernière
// position connue survit aux cycles sautés.
func TestZeroInsert_MatchArriveePendantUnCycleSaute(t *testing.T) {
	lecteur := &dernierMatchScripte{}
	t1 := time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC)
	lecteur.fixer(t1)
	s := schedulerZeroInsert(t, lecteur)
	s.RunOnce(context.Background()) // position de départ T1, compteur 1
	s.RunOnce(context.Background()) // compteur 2
	if got := compteur(s); got != 2 {
		t.Fatalf("compteur = %d, attendu 2", got)
	}

	s.ActivityChecker = &fakeActivityChecker{activeGamertags: map[string]bool{"Player1": true}}
	lecteur.fixer(t1.Add(40 * time.Minute)) // le suivi en direct range un match pendant que le cycle cède
	s.RunOnce(context.Background())
	s.ActivityChecker = nil
	s.RunOnce(context.Background())
	if got := compteur(s); got != 0 {
		t.Errorf("compteur = %d, attendu 0 : un match est arrivé pendant le cycle sauté", got)
	}
}

// Un joueur sans match connu n'est pas une arrivée : le compteur suit la règle du cycle seul.
func TestZeroInsert_JoueurSansMatch(t *testing.T) {
	s := schedulerZeroInsert(t, &dernierMatchScripte{})
	for cycle := 1; cycle <= 3; cycle++ {
		s.RunOnce(context.Background())
		if got := compteur(s); got != cycle {
			t.Fatalf("cycle %d : compteur = %d, attendu %d", cycle, got, cycle)
		}
	}
}

// Sans lecteur câblé (anciens sites de construction), l'alerte garde sa règle d'avant mais ne sonne plus à
// chaque cycle.
func TestZeroInsert_SansLecteur_UneAlerteParEpisode(t *testing.T) {
	journal := capturerJournal(t)
	s := schedulerZeroInsert(t, nil)
	for cycle := 0; cycle < scheduler.ConsecutiveZeroInsertWarnThreshold+5; cycle++ {
		s.RunOnce(context.Background())
	}
	if n := alertesZeroInsert(journal()); n != 1 {
		t.Errorf("%d alerte(s), attendu 1 : une par épisode", n)
	}
}
