// Package observability — matchsteps.go : collecteur des instants mesurés du chemin d'un
// match (domain.MatchStep), vidé vers la base monitoring par le serveur.
//
// POURQUOI UN COLLECTEUR ET PAS UNE ÉCRITURE DIRECTE. Les étapes se produisent dans le
// post-sync, le watcher, la file de construction et les dérivations : leur faire acquérir le
// writer monitoring ajouterait une attente de lease au chemin même qu'on mesure. L'instant est
// donc CAPTÉ ici, au moment où l'étape a lieu, et écrit plus tard par la boucle de flush
// monitoring (même writer, même lease que la file de construction).
//
// UN SEUL ÉCRIVAIN DE LA BASE MONITORING : le serveur arme le collecteur au boot
// ([EnableMatchSteps]). Un CLI ne l'arme jamais : ses [ReportMatchStep] ne coûtent rien et ne
// laissent rien.
package observability

import (
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

// MatchStepsBufferCap borne le collecteur entre deux vidages. Une dizaine d'étapes par match :
// 4 096 couvrent des centaines de matchs, bien au-delà d'une minute de travail réel.
const MatchStepsBufferCap = 4096

// CompteurMatchStepsPerdus : instants écartés parce que le collecteur était plein.
const CompteurMatchStepsPerdus = "match_steps_dropped_total"

type matchStepsCollector struct {
	mu      sync.Mutex
	enabled bool
	buf     []domain.MatchStepEvent
}

var matchSteps = &matchStepsCollector{}

// EnableMatchSteps arme (true) ou désarme (false) le collecteur. Désarmer vide le tampon :
// rien ne doit survivre à l'arrêt du store qui le vidait.
func EnableMatchSteps(on bool) {
	matchSteps.mu.Lock()
	matchSteps.enabled = on
	if !on {
		matchSteps.buf = nil
	}
	matchSteps.mu.Unlock()
}

// ReportMatchStep capte l'instant d'une étape pour un match, MAINTENANT.
func ReportMatchStep(titleSlug, matchID string, step domain.MatchStep, detail string) {
	ReportMatchStepAt(titleSlug, matchID, step, time.Now(), detail)
}

// ReportMatchStepAt capte une étape dont l'instant est connu par ailleurs (fin de match lue au
// registre). Sans effet si le collecteur n'est pas armé ou si le match n'est pas nommé.
func ReportMatchStepAt(titleSlug, matchID string, step domain.MatchStep, at time.Time, detail string) {
	matchID = strings.TrimSpace(matchID)
	if matchID == "" || at.IsZero() {
		return
	}
	matchSteps.mu.Lock()
	defer matchSteps.mu.Unlock()
	if !matchSteps.enabled {
		return
	}
	if len(matchSteps.buf) >= MatchStepsBufferCap {
		IncCounter(CompteurMatchStepsPerdus)
		return
	}
	matchSteps.buf = append(matchSteps.buf, domain.MatchStepEvent{
		TitleSlug: titleSlug, MatchID: matchID, Step: step, At: at.UTC(), Detail: detail,
	})
}

// DrainMatchSteps rend les instants captés depuis le dernier vidage et vide le collecteur.
func DrainMatchSteps() []domain.MatchStepEvent {
	matchSteps.mu.Lock()
	defer matchSteps.mu.Unlock()
	out := matchSteps.buf
	matchSteps.buf = nil
	return out
}

// RequeueMatchSteps remet en tête du collecteur des instants dont l'écriture a échoué, dans la
// limite de sa borne (le surplus est compté perdu). Sans effet si le collecteur est désarmé.
func RequeueMatchSteps(events []domain.MatchStepEvent) {
	if len(events) == 0 {
		return
	}
	matchSteps.mu.Lock()
	defer matchSteps.mu.Unlock()
	if !matchSteps.enabled {
		return
	}
	place := MatchStepsBufferCap - len(matchSteps.buf)
	if place < len(events) {
		AddInt(CompteurMatchStepsPerdus, int64(len(events)-max(place, 0)))
		events = events[:max(place, 0)]
	}
	matchSteps.buf = append(append([]domain.MatchStepEvent{}, events...), matchSteps.buf...)
}
