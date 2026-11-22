// Package shieldcycle compte, par joueur, les bris et les recharges complètes de bouclier d'un match.
//
// Le paquet est pur : il reçoit les mesures de bouclier de chaque vie (une piste du film) et rend
// des événements datés en images. Il ne lit ni base, ni horloge, ni document de rejeu, et ne porte
// aucun libellé. Le passage d'un document à [Life] est dans `internal/replayshield`.
//
// Règles (une vie = un corps, de sa naissance à sa fin ; un respawn ouvre une nouvelle vie) :
//
//   - Bris : une mesure passe d'une valeur > 0 à 0. La première mesure d'une vie compte, quand elle
//     vaut 0. Plusieurs mesures à 0 de suite ne font qu'un bris.
//   - Recharge : après un bris, la première mesure >= 1 (plein, surbouclier compris) de la même vie.
//     Au plus une recharge par bris. Une remontée partielle n'est ni une recharge ni la fin du bris :
//     le bris reste en attente jusqu'au plein ou jusqu'à la fin de la vie.
//   - Une vie qui finit avec un bris en attente ne rend pas de recharge : l'état d'une vie ne passe
//     jamais à la vie suivante.
package shieldcycle

import "sort"

// FullShield est la mesure à partir de laquelle le bouclier est plein.
const FullShield float32 = 1

// Kind distingue les deux événements.
type Kind uint8

const (
	// Break est un bris de bouclier.
	Break Kind = iota + 1
	// Recharge est une recharge complète qui suit un bris.
	Recharge
)

// Measure est une mesure de bouclier : l'image du film où elle est écrite et la fraction mesurée.
type Measure struct {
	Frame  int
	Shield float32
}

// Life est une vie nommée : le joueur et ses mesures de bouclier, dans l'ordre du film.
type Life struct {
	XUID     string
	Measures []Measure
}

// Event est un bris ou une recharge. Frame est l'image de la mesure qui l'a établi ; l'appelant la
// convertit en millisecondes avec l'intervalle d'image du document.
type Event struct {
	Frame int
	Kind  Kind
}

// Cycles rend, par xuid, les événements de bouclier de ses vies, triés par image (à image égale :
// l'ordre de l'entrée). Une clé est présente dès que le joueur a au moins une mesure, même sans
// événement : l'absence d'une clé dit « non mesuré ». Les vies sans xuid sont ignorées.
func Cycles(lives []Life) map[string][]Event {
	out := map[string][]Event{}
	for i := range lives {
		l := &lives[i]
		if l.XUID == "" || len(l.Measures) == 0 {
			continue
		}
		out[l.XUID] = append(out[l.XUID], lifeEvents(l.Measures)...)
	}
	for _, ev := range out {
		sort.SliceStable(ev, func(a, b int) bool { return ev[a].Frame < ev[b].Frame })
	}
	return out
}

// lifeEvents applique les règles de bris et de recharge aux mesures d'une seule vie.
func lifeEvents(ms []Measure) []Event {
	var events []Event
	seen, pending := false, false
	var prev float32
	for _, m := range ms {
		switch {
		case m.Shield == 0 && (!seen || prev > 0):
			events = append(events, Event{Frame: m.Frame, Kind: Break})
			pending = true
		case pending && m.Shield >= FullShield:
			events = append(events, Event{Frame: m.Frame, Kind: Recharge})
			pending = false
		}
		seen, prev = true, m.Shield
	}
	return events
}

// Totals compte les bris et les recharges d'une liste d'événements.
func Totals(events []Event) (breaks, recharges int) {
	for _, e := range events {
		switch e.Kind {
		case Break:
			breaks++
		case Recharge:
			recharges++
		}
	}
	return breaks, recharges
}
