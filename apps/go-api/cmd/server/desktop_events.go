// desktop_events.go — les événements du contrat coquille <-> serveur (ADR 0045 D-2).
//
// En -mode desktop, la coquille lit la sortie standard du serveur ligne à ligne : UNE ligne JSON par
// événement, et rien d'autre (tout journal console va sur la sortie d'erreur, cf.
// newConsoleLogHandler). Les événements :
//
//	{"event":"listening","port":N}            le port est lié, avant les migrations
//	{"event":"ready","port":N}                le routeur est ouvert (bootServer.openRouter)
//	{"event":"failed","state":"<état>"}       le démarrage est refusé (bootFailureState)
//	{"event":"stopping"}                      la séquence d'arrêt commence
//	{"event":"stopped","clean":true|false}    la séquence est finie, journaux fermés
//	{"event":"notification","title":"…","body":"…","kind":"sync"}
//	                                          une notification de la cloche à montrer comme notification
//	                                          Windows (réglage native_notifications) ; texte déjà composé
//	                                          dans la langue du joueur
//	{"event":"shell_settings","native_notifications":false,"close_to_tray":false,
//	 "start_with_windows":false,"locale":"fr"}
//	                                          les réglages que la coquille applique (zone de notification,
//	                                          démarrage avec Windows) et la langue de ses libellés ; écrit une
//	                                          fois le serveur prêt puis à chaque changement
//	{"event":"resources","phase":"downloading","fingerprint":"…","expected":"…","active":"…","percent":42,
//	 "code":"","restart_needed":false}
//	                                          l'avancement de la préparation du paquet de ressources (resourcepack.Status) :
//	                                          la coquille attend `ready` ou `failed` avant d'installer le moteur qui l'exige
//
// L'écrivain est unique et tient un verrou : deux goroutines (démarrage, signal d'arrêt) ne
// mélangent jamais leurs octets. Un *desktopEvents nil est un écrivain désactivé : hors du mode
// desktop, aucun appelant n'a de branche à écrire.
package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"

	"levelup/go-api/internal/resourcepack"
)

// Noms des événements (valeur du champ "event").
const (
	eventListening = "listening"
	eventReady     = "ready"
	eventFailed    = "failed"
	eventStopping  = "stopping"
	eventStopped   = "stopped"
	// eventNotification : voir l'en-tête (L21.b).
	eventNotification = "notification"
	// eventShellSettings : voir l'en-tête (L21.c).
	eventShellSettings = "shell_settings"
	// eventResources : voir l'en-tête (L24).
	eventResources = "resources"
)

// desktopEvents écrit les événements du contrat sur w, une ligne chacun, sous un verrou.
type desktopEvents struct {
	mu sync.Mutex
	// w est la sortie standard du processus. os.Stdout n'est pas tamponné par Go : chaque Write
	// part au tube immédiatement, il n'y a rien à vider (un Sync sur un tube Windows bloquerait
	// jusqu'à la lecture complète par la coquille).
	w io.Writer
}

// newDesktopEvents crée l'écrivain d'événements sur w.
func newDesktopEvents(w io.Writer) *desktopEvents { return &desktopEvents{w: w} }

// Les formes JSON, une par événement : l'ordre des clés est celui des champs.
type (
	eventPort struct {
		Event string `json:"event"`
		Port  int    `json:"port"`
	}
	eventState struct {
		Event string           `json:"event"`
		State bootFailureState `json:"state"`
	}
	eventBare struct {
		Event string `json:"event"`
	}
	eventStoppedBody struct {
		Event string `json:"event"`
		Clean bool   `json:"clean"`
	}
	eventShellSettingsBody struct {
		Event string `json:"event"`
		shellSettings
	}
	eventResourcesBody struct {
		Event string `json:"event"`
		resourcepack.Status
	}
	eventNotificationBody struct {
		Event string `json:"event"`
		Title string `json:"title"`
		Body  string `json:"body"`
		Kind  string `json:"kind"`
	}
)

// Listening annonce le port lié.
func (e *desktopEvents) Listening(port int) { e.emit(eventPort{Event: eventListening, Port: port}) }

// Ready annonce le routeur ouvert, sur le port lié.
func (e *desktopEvents) Ready(port int) { e.emit(eventPort{Event: eventReady, Port: port}) }

// Failed annonce le refus de démarrer, avec son état.
func (e *desktopEvents) Failed(state bootFailureState) {
	e.emit(eventState{Event: eventFailed, State: state})
}

// Stopping annonce le début de la séquence d'arrêt.
func (e *desktopEvents) Stopping() { e.emit(eventBare{Event: eventStopping}) }

// Stopped annonce la fin de la séquence d'arrêt ; clean est la définition de runShutdown.
func (e *desktopEvents) Stopped(clean bool) {
	e.emit(eventStoppedBody{Event: eventStopped, Clean: clean})
}

// Notification annonce une notification de la cloche à montrer comme notification Windows : titre et
// corps déjà composés dans la langue du joueur, kind le type (journal de la coquille).
func (e *desktopEvents) Notification(title, body, kind string) {
	e.emit(eventNotificationBody{Event: eventNotification, Title: title, Body: body, Kind: kind})
}

// ShellSettings annonce les réglages que la coquille applique.
func (e *desktopEvents) ShellSettings(s shellSettings) {
	e.emit(eventShellSettingsBody{Event: eventShellSettings, shellSettings: s})
}

// Resources annonce l'avancement de la préparation du paquet de ressources.
func (e *desktopEvents) Resources(s resourcepack.Status) {
	e.emit(eventResourcesBody{Event: eventResources, Status: s})
}

// emit écrit v en une ligne JSON, d'un seul Write. Un événement perdu (sortie fermée par la
// coquille) est journalisé sur la sortie d'erreur et ne bloque jamais le serveur.
func (e *desktopEvents) emit(v any) {
	if e == nil {
		return
	}
	line, err := json.Marshal(v)
	if err != nil {
		slog.ErrorContext(context.Background(), "événement desktop : encodage impossible", "err", err)
		return
	}
	line = append(line, '\n')
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, err := e.w.Write(line); err != nil {
		slog.WarnContext(context.Background(), "événement desktop : écriture impossible", "err", err,
			"event", string(line))
	}
}
