// Package presence — types.go : types partagés entre clients de présence.
//
// Ces types sont consommés par le REST poll (rest_client.go + rest_poller
// dans le package watcher) et la chaîne handler du watcher daemon.
package presence

import "time"

// PresenceEvent représente un événement de changement de présence d'un user.
type PresenceEvent struct {
	XUID           string
	PresenceState  string          // "Online", "Offline", "Away"
	PresenceDetail *PresenceDetail // nil si offline ou pas de jeu actif
	LastSeen       *LastSeenInfo   // non-nil si Xbox a renvoyé un bloc `lastSeen`
}

// PresenceDetail contient les informations du titre en cours.
type PresenceDetail struct {
	TitleID   string
	TitleName string
	IsGame    bool
	IsPrimary bool
	Device    string
	State     string // "Active", "Inactive"
}

// LastSeenInfo représente le bloc `lastSeen` renvoyé par Xbox quand le user
// est offline ou n'a pas de titre actif. Donne la dernière activité connue
// (titre + timestamp UTC). Utile pour afficher "vu il y a 2h sur Halo Infinite".
type LastSeenInfo struct {
	Timestamp  time.Time
	TitleID    string
	TitleName  string
	DeviceType string
}

// EventHandler est le callback appelé quand un event de présence est reçu.
type EventHandler func(event PresenceEvent)
