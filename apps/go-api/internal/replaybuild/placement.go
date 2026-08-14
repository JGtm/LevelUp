// Package replaybuild — placement.go : OÙ SE CONSTRUIT UN REJEU.
//
// LE POINT DE DÉCISION UNIQUE. Trois appelants veulent construire un rejeu :
// l'étape post-sync (fil de l'eau), l'action admin, et le CLI de backfill. Sans
// cette fonction, chacun porterait son propre `if` sur l'environnement et le
// réglage — trois copies qui divergeraient au premier changement de règle. Ici la
// question se pose UNE fois : je construis, je mets en file, ou je ne fais rien.
//
// LE CLI DE BACKFILL NE PASSE PAS PAR ICI, ET C'EST VOULU : c'est un outil
// d'OPÉRATEUR, pas un chemin de service. Quelqu'un qui tape `levelup
// backfill-replay` a déjà décidé où il construit — sur sa machine, avec ses films
// en cache. Le soumettre au réglage lui ferait mettre en file un rattrapage de
// 951 matchs qu'aucun ouvrier n'a les films pour traiter.
package replaybuild

import (
	"fmt"
	"log/slog"
)

// Placement dit ce qu'un appelant doit faire d'un match à construire.
type Placement string

const (
	// PlacementLocal : ce processus décode lui-même, en mémoire de ce serveur.
	// Défaut EN DÉVELOPPEMENT, interdit en production.
	PlacementLocal Placement = "local"
	// PlacementWorker : ce processus MET EN FILE et ne décode jamais. Défaut en
	// production (« le VPS web ne décode JAMAIS », piste F §1).
	PlacementWorker Placement = "worker"
	// PlacementOff : aucune construction. Le rejeu se contente de ce qui existe
	// déjà — la dégradation par absence d'artefact est écrite depuis le lot 1.
	PlacementOff Placement = "off"
)

// ErrLocalBuildInProduction : le réglage demande une construction LOCALE sur une
// instance de production. Refus EXPLICITE, jamais silencieux : l'admin qui pose ce
// réglage doit apprendre pourquoi il ne s'appliquera pas.
//
// Le VPS web a un petit CPU et un disque sous tension ; un décodage de film y
// dure ~50 s et bloque le cycle. C'est une règle de conception du chantier, pas
// une préférence.
var ErrLocalBuildInProduction = fmt.Errorf(
	"construction locale interdite en production : le VPS web ne décode jamais un film (choisir « worker » ou « off »)")

// PlacementEnv décrit l'instance qui pose la question.
type PlacementEnv struct {
	// Production : instance de production (config.AppConfig.IsProduction()).
	Production bool
	// WorkerConfigured : le protocole ouvrier est OUVERT (jeton configuré). Sans
	// lui, personne ne viendra prendre le travail — mettre en file ne ferait
	// qu'allonger une file que rien ne vide, et résoudre des manifestes Halo pour
	// rien à chaque cycle.
	WorkerConfigured bool
}

// DecidePlacement résout le réglage en une décision. `setting` est la valeur
// brute d'app_settings (vide = défaut de l'instance).
//
// Rend (placement effectif, erreur explicative). L'erreur ne signale JAMAIS une
// panne : elle porte le motif d'une décision dégradée, à journaliser et à montrer
// à l'admin. Le placement rendu avec une erreur est toujours sûr (PlacementOff).
func DecidePlacement(setting string, env PlacementEnv) (Placement, error) {
	switch Placement(setting) {
	case PlacementOff:
		return PlacementOff, nil
	case PlacementLocal:
		if env.Production {
			return PlacementOff, ErrLocalBuildInProduction
		}
		return PlacementLocal, nil
	case PlacementWorker:
		if !env.WorkerConfigured {
			return PlacementOff, fmt.Errorf(
				"mise en file demandée mais le protocole ouvrier n'est pas ouvert (LEVELUP_BUILD_WORKER_TOKEN absent) : rien ne viendrait vider la file")
		}
		return PlacementWorker, nil
	case "":
		// Défaut de l'instance : la production délègue, le développement construit
		// sur place. Écrit ici et nulle part ailleurs.
		if env.Production {
			return DecidePlacement(string(PlacementWorker), env)
		}
		return PlacementLocal, nil
	default:
		return PlacementOff, fmt.Errorf("réglage de lieu de construction inconnu %q (attendu local, worker ou off)", setting)
	}
}

// ValidPlacementSetting dit si une valeur est acceptable comme réglage (le vide
// l'est : c'est « défaut de l'instance »). Sert la validation du PATCH settings.
func ValidPlacementSetting(setting string) bool {
	switch Placement(setting) {
	case "", PlacementLocal, PlacementWorker, PlacementOff:
		return true
	default:
		return false
	}
}

// LogPlacement journalise une décision dégradée, une seule ligne par appel.
// Passer par ce helper évite que chaque appelant réinvente le niveau de log : un
// réglage qui ne s'applique pas est un WARN, jamais un silence (règle n°3).
func LogPlacement(where string, p Placement, err error) {
	if err == nil {
		return
	}
	slog.Warn("rejeu 2D : lieu de construction dégradé",
		"appelant", where, "placement", string(p), "raison", err.Error())
}
