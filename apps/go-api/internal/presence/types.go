// Package presence — types.go : types partagés entre clients de présence.
//
// Ces types sont consommés par le REST poll (rest_client.go + rest_poller
// dans le package watcher) et la chaîne handler du watcher daemon.
package presence

// PresenceEvent représente un événement de changement de présence d'un user.
type PresenceEvent struct {
	XUID           string
	PresenceState  string          // "Online", "Offline", "Away"
	PresenceDetail *PresenceDetail // nil si offline ou pas de jeu actif
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

// EventHandler est le callback appelé quand un event de présence est reçu.
type EventHandler func(event PresenceEvent)
