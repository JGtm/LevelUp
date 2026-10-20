// Package timing chronomètre les SECTIONS d'une requête HTTP : où passe le temps d'une
// page, sans profileur (plan perf 2026-09-23, lot L1, décisions D1.2 et D1.3).
//
// # Qui fait quoi
//
// Le middleware HTTP (`middleware.SlogLogger`) pose un `*Timings` sur le contexte de CHAQUE
// requête (WithTimings), puis journalise ses sections quand la requête est lente ou que le
// niveau DEBUG est actif (ligne `http_timings`). Un service n'appelle que :
//
//	defer timing.FromContext(ctx).Section("nom")()
//
// ou, pour une étape au milieu d'une fonction (un `defer` ne se déclencherait qu'à sa
// sortie) :
//
//	stop := timing.FromContext(ctx).Section("nom")
//	rows, err := repo.Load(ctx)
//	stop()
//
// # Sans middleware, rien ne se passe
//
// Hors requête HTTP (tests, CLI, tâches de fond), le contexte ne porte aucun `*Timings` :
// FromContext rend nil, et toutes les méthodes d'un `*Timings` nil sont des no-op sûrs
// (Section rend une fonction vide, Snapshot et LogAttrs rendent nil).
//
// # Sections : des feuilles, cumulées par nom
//
// Un même nom appelé plusieurs fois (un builder par coéquipier, un bloc par session) cumule
// sa durée et compte ses appels. Les sections sont des FEUILLES : ne jamais en poser une
// autour d'un appel qui en contient déjà — `total_ms` (somme de toutes les sections) les
// compterait deux fois. Posées ainsi, sur un chemin séquentiel, `total_ms` ne dépasse pas la
// durée de la requête, et l'écart est le temps passé hors sections.
package timing

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// maxLoggedSections borne le nombre de sections détaillées dans `sections` (les plus
// longues d'abord) : une page à vingt-six sections resterait lisible sur une ligne de log.
// `total_ms` couvre toujours TOUTES les sections, y compris celles qui ne sont pas listées.
const maxLoggedSections = 15

// SectionStat est le cumul d'une section : durée totale et nombre d'appels.
type SectionStat struct {
	Name     string
	Duration time.Duration
	Calls    int
}

// Timings accumule les sections d'une requête. Sûr en concurrence (mutex) : une section
// peut se fermer depuis une autre goroutine que celle qui l'a ouverte.
type Timings struct {
	mu     sync.Mutex
	now    func() time.Time
	order  []string // ordre de première apparition (départage des égalités de durée)
	byName map[string]*SectionStat
}

type ctxKey struct{}

// noop est la fonction d'arrêt rendue par un `*Timings` nil : aucune allocation par appel.
func noop() {}

// WithTimings pose un nouveau `*Timings` sur le contexte et le rend.
func WithTimings(ctx context.Context) (context.Context, *Timings) {
	t := newTimings(time.Now)
	return context.WithValue(ctx, ctxKey{}, t), t
}

// newTimings construit un `*Timings` sur une horloge donnée (time.Now hors tests).
func newTimings(now func() time.Time) *Timings {
	return &Timings{now: now, byName: make(map[string]*SectionStat)}
}

// FromContext rend le `*Timings` du contexte, ou nil s'il n'y en a pas (tests, CLI,
// contexte sans middleware) — nil est un no-op sûr.
func FromContext(ctx context.Context) *Timings {
	if ctx == nil {
		return nil
	}
	t, _ := ctx.Value(ctxKey{}).(*Timings)
	return t
}

// Section ouvre la section `name` et rend la fonction qui l'arrête (à appeler UNE fois,
// typiquement par `defer`). Sur un `*Timings` nil, rend une fonction vide.
func (t *Timings) Section(name string) func() {
	if t == nil {
		return noop
	}
	start := t.now()
	return func() { t.record(name, t.now().Sub(start)) }
}

// record ajoute une durée au cumul de `name`.
func (t *Timings) record(name string, d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	stat, ok := t.byName[name]
	if !ok {
		stat = &SectionStat{Name: name}
		t.byName[name] = stat
		t.order = append(t.order, name)
	}
	stat.Duration += d
	stat.Calls++
}

// Snapshot rend une copie des cumuls, triée par durée décroissante (à durée égale, ordre
// de première apparition). nil sur un `*Timings` nil ou sans section.
func (t *Timings) Snapshot() []SectionStat {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	if len(t.order) == 0 {
		t.mu.Unlock()
		return nil
	}
	out := make([]SectionStat, 0, len(t.order))
	for _, name := range t.order {
		out = append(out, *t.byName[name])
	}
	t.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool { return out[i].Duration > out[j].Duration })
	return out
}

// LogAttrs rend les attributs slog de la ligne `http_timings` :
//
//   - `total_ms` : somme des cumuls de TOUTES les sections ;
//   - `sections` : "nom=ms" séparés par des espaces, les plus longues d'abord, au plus
//     maxLoggedSections ;
//   - `calls` : "nom=n" pour les sections listées appelées plus d'une fois (absent sinon).
//
// nil quand il n'y a aucune section (ou sur un `*Timings` nil) : rien à journaliser.
func (t *Timings) LogAttrs() []any {
	stats := t.Snapshot()
	if len(stats) == 0 {
		return nil
	}
	var total time.Duration
	for _, s := range stats {
		total += s.Duration
	}
	shown := stats
	if len(shown) > maxLoggedSections {
		shown = shown[:maxLoggedSections]
	}
	sections := make([]string, 0, len(shown))
	var calls []string
	for _, s := range shown {
		sections = append(sections, s.Name+"="+strconv.FormatInt(s.Duration.Milliseconds(), 10))
		if s.Calls > 1 {
			calls = append(calls, s.Name+"="+strconv.Itoa(s.Calls))
		}
	}
	attrs := []any{"total_ms", total.Milliseconds(), "sections", strings.Join(sections, " ")}
	if len(calls) > 0 {
		attrs = append(attrs, "calls", strings.Join(calls, " "))
	}
	return attrs
}
