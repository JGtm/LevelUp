package sharedprovider

// hold_journal.go — un dépassement de seuil se dit UNE fois par épisode, pas à chaque occurrence.
//
// Un rejeu de notes prend le writer RW des centaines de fois de suite, chacune au-delà du seuil
// du watchdog : une ligne par prise noyait le journal de démarrage sans rien apprendre de plus
// que la première. Un épisode est une série de dépassements de la même clé (détenteur du writer,
// appelant d'une lecture) séparés de moins de episodeSilence. Le premier dépassement d'un épisode
// est journalisé ; les suivants sont comptés (les compteurs expvar, eux, voient tout) et leur
// nombre est rendu avec la première ligne de l'épisode d'après.

import (
	"sync"
	"sync/atomic"
	"time"
)

// episodeSilence : écart au-delà duquel un nouveau dépassement ouvre un nouvel épisode.
const episodeSilence = 5 * time.Minute

// episodesMax borne la table d'une clé à cardinalité libre (appelants de lectures) : au-delà, la
// table repart de zéro, ce qui journalise de nouveau au plus une fois par clé.
const episodesMax = 256

type episodeDepassement struct {
	dernier time.Time
	tus     int
}

// journalDepassements décide quel dépassement mérite une ligne. Valeur zéro utilisable.
type journalDepassements struct {
	mu       sync.Mutex
	episodes map[string]*episodeDepassement
	now      func() time.Time
}

// signaler enregistre un dépassement de la clé. Rend vrai si une ligne doit être écrite, avec le
// nombre de dépassements tus de l'épisode précédent de la même clé.
func (j *journalDepassements) signaler(cle string) (journaliser bool, tusAvant int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	maintenant := time.Now()
	if j.now != nil {
		maintenant = j.now()
	}
	if j.episodes == nil || len(j.episodes) >= episodesMax {
		j.episodes = make(map[string]*episodeDepassement)
	}
	ep, ok := j.episodes[cle]
	if ok && maintenant.Sub(ep.dernier) < episodeSilence {
		ep.dernier = maintenant
		ep.tus++
		return false, 0
	}
	if ok {
		tusAvant = ep.tus
	}
	j.episodes[cle] = &episodeDepassement{dernier: maintenant}
	return true, tusAvant
}

// rwHoldRecord suit UNE acquisition RW : le watchdog (timer, sans verrou) y pose « annoncé » s'il
// a écrit la ligne de dépassement, Get y compte les lectures qui ont dû attendre la fin du swap.
// Un timer d'une acquisition précédente ne touche que son propre enregistrement.
type rwHoldRecord struct {
	annonce          atomic.Bool
	lecturesRetenues atomic.Int64
}
