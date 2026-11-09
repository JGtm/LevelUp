package wakeup

import "time"

const (
	// SleepTick : la période du tic qui échantillonne les horloges.
	SleepTick = 30 * time.Second
	// SleepGap : l'écart entre horloge murale et horloge monotone, depuis le tic précédent, au-delà
	// duquel la machine est réputée sortir de veille. Un tic en retard de quelques secondes (charge,
	// ordonnancement) ne l'atteint jamais.
	SleepGap = 2 * time.Minute
)

// Clocks : les deux horloges que compare le détecteur. Séparées pour être injectées en test.
type Clocks interface {
	// Wall rend l'heure murale. Elle avance pendant la veille.
	Wall() time.Time
	// Mono rend le temps monotone écoulé depuis une origine fixe. Sous Windows (il lit le compteur
	// d'interruptions, qui ne compte pas la veille : ADR 0045 A24) comme sous Linux
	// (CLOCK_MONOTONIC), il N'avance PAS pendant la veille.
	Mono() time.Duration
}

// SystemClocks est la paire de production : l'heure murale du système et la lecture monotone de Go.
type SystemClocks struct {
	origin time.Time // porte une lecture monotone
}

// NewSystemClocks fixe l'origine du temps monotone à maintenant.
func NewSystemClocks() SystemClocks { return SystemClocks{origin: time.Now()} }

// Wall rend l'heure murale, SANS lecture monotone : un temps qui porte les deux ne se compare à un
// autre qu'en monotone, ce qui ferait disparaître l'écart que ce paquet cherche.
func (SystemClocks) Wall() time.Time { return time.Now().Round(0) }

// Mono rend le temps monotone écoulé depuis la création.
func (c SystemClocks) Mono() time.Duration { return time.Since(c.origin) }

// SleepDetector compare, d'un échantillon au suivant, l'avance de l'horloge murale et celle de
// l'horloge monotone : hors veille elles avancent ensemble ; une veille n'avance que la première.
// Pas de goroutine : l'appelant échantillonne à son rythme ([SleepTick]).
type SleepDetector struct {
	clocks   Clocks
	lastWall time.Time
	lastMono time.Duration
}

// NewSleepDetector prend le premier échantillon.
func NewSleepDetector(c Clocks) *SleepDetector {
	return &SleepDetector{clocks: c, lastWall: c.Wall(), lastMono: c.Mono()}
}

// Check prend un échantillon et rend le temps de veille depuis le précédent (l'avance murale moins
// l'avance monotone) ; woke est vrai quand il dépasse [SleepGap]. Une horloge murale qui recule (réglage
// manuel, synchronisation NTP) donne un écart négatif : ce n'est pas une veille.
func (d *SleepDetector) Check() (gap time.Duration, woke bool) {
	wall, mono := d.clocks.Wall(), d.clocks.Mono()
	gap = wall.Sub(d.lastWall) - (mono - d.lastMono)
	d.lastWall, d.lastMono = wall, mono
	return gap, gap > SleepGap
}
