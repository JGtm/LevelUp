// Package replaybuild — artifact_events.go : LE PUITS D'ÉVÉNEMENTS « artefact RANGÉ ».
//
// CE QUE C'EST. Un point d'observation unique, câblé au boot par le serveur, qui reçoit
// UN événement par artefact de rejeu RÉELLEMENT écrit sur le disque. Il alimente la
// notification Discord groupée (lot B v7.5) ; il ne décide de rien et ne bloque rien.
//
// POURQUOI UN PUITS DE PROCESS, ET PAS UN NOTIFIEUR PASSÉ EN PARAMÈTRE. Le point
// d'écriture (`writeArtifactBytes`) a QUATRE appelants répartis dans TROIS binaires :
// le fil de l'eau post-sync, le dépôt d'un ouvrier et l'action admin (tous trois dans le
// serveur), plus le CLI de rattrapage (process séparé). Faire descendre un notifieur
// jusqu'ici imposerait trois sites d'émission dans le serveur — la 3e copie d'un même
// geste, que la règle des <= 2 exemplaires interdit, et qui divergeraient au premier
// ajustement. Le patron retenu est celui que ce dépôt utilise DÉJÀ pour le même besoin :
// `notify.SetDefaultLabelsResolver` (resolver partagé câblé au boot, RWMutex, nil =
// repli failsafe) et `observability.IncCounter`, appelé à quelques lignes d'ici.
//
// CONSÉQUENCE VOULUE : seul le process qui câble le puits notifie. Le CLI
// `levelup backfill-replay` ne le câble pas, donc un backfill de masse (des centaines
// d'artefacts) ne produit AUCUN message. Et quand l'ouvrier tourne sur le poste de
// développement avec `--work` pointant le cache du dépôt, le même fichier est écrit deux
// fois — une fois par l'ouvrier, une fois par le serveur au dépôt — mais une seule des
// deux écritures est observée. Ne pas câbler ce puits ailleurs qu'au boot du serveur.
//
// GARDE-RAIL : `internal/archlint/no_second_artifact_sink_test.go` interdit tout appelant
// de production de SetArtifactStoredSink hors du câblage de boot. Un second câblage ne
// s'ajouterait pas au premier : il le REMPLACERAIT, et le gagnant dépendrait de l'ordre
// de boot.
package replaybuild

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// ArtifactStored décrit un artefact de rejeu qui VIENT D'ÊTRE ÉCRIT.
//
// MatchID est celui que l'APPELANT tient pour canonique (identité du job côté ouvrier,
// identité du registre côté post-sync et action admin), jamais celui que le document
// revendique : `match_registry` est indexé par match_id COMPLET, et un document déposé
// par un ouvrier peut porter la forme courte (cf. validateArtifact, qui ne compare que
// les formes courtes). Publier la forme du document rendrait le match introuvable en base
// au moment de construire le lien de la notification.
type ArtifactStored struct {
	TitleSlug     string
	MatchID       string
	Path          string
	Bytes         int
	Tracks        int
	SchemaVersion int
}

var (
	storedSinkMu sync.RWMutex
	storedSink   func(ArtifactStored)
)

// SetArtifactStoredSink câble le puits. Appelé UNE SEULE FOIS, au boot du serveur.
// fn nil remet le puits à l'état inerte (utile aux tests, qui restaurent l'état
// précédent en defer).
func SetArtifactStoredSink(fn func(ArtifactStored)) {
	storedSinkMu.Lock()
	storedSink = fn
	storedSinkMu.Unlock()
}

// publishArtifactStored notifie le puits. Sans puits câblé : no-op silencieux (cas
// nominal des CLI, de l'ouvrier et de tous les tests).
//
// FAILSAFE STRICT : un puits qui panique ne doit JAMAIS faire échouer une écriture
// d'artefact — le rejeu est le produit, la notification n'en est que l'annonce. La panique
// est récupérée et JOURNALISÉE (jamais avalée), l'écriture reste réussie.
func publishArtifactStored(ev ArtifactStored) {
	storedSinkMu.RLock()
	fn := storedSink
	storedSinkMu.RUnlock()
	if fn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(context.Background(),
				"replaybuild: puits d'artefact en panique — écriture conservée, notification perdue",
				"match_id", ev.MatchID, "title", ev.TitleSlug, "recover", fmt.Sprintf("%v", r))
		}
	}()
	fn(ev)
}
