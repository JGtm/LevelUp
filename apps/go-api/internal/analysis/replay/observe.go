package replay

// observe.go — L'OBSERVATEUR DE CONSTRUCTION : chaque etape de BuildFromFilm se montre.
//
// # POURQUOI IL VIT DANS LE CODE DE PRODUCTION
//
// Le harnais d'equivalence de la cuisson (`cmd/replay-equiv`, PLAN_CUISSON_PERF §3 D4a) prouve
// qu'un refacto du decodeur rend des sorties IDENTIQUES en hachant la sortie de CHAQUE balayage,
// pas seulement l'artefact final : c'est ce qui localise une divergence au balayage pres au lieu
// de dire « quelque chose a change ». La sequence des balayages ne vit qu'a UN endroit —
// BuildFromFilm — et une copie de cette sequence dans un harnais aurait derive au premier
// balayage ajoute. L'observateur est donc appele ICI, apres chaque etape, avec son nom et sa
// valeur.
//
// # CE QU'IL COUTE EN PRODUCTION
//
// `Options.Observe` nil ne coute rien — mais EN PRODUCTION IL N'EST JAMAIS NIL :
// `replaybuild.BuildBytes` passe systematiquement `b.observe` (methode liee, cf.
// `replaybuild/replaybuild.go`), et c'est CETTE methode qui teste si un observateur est branche
// (`b.observer != nil`). La cuisson paie donc, par etape, un appel de fonction et un test de
// pointeur nil — soit une trentaine d'appels vides par film. Le nil ne concerne que les
// appelants qui construisent `replay.Options` eux-memes (tests, `cmd/zone-attribution`).
//
// # CE QU'IL N'EST PAS
//
// Ni un journal, ni un point d'extension : il ne modifie aucune valeur et n'est appele qu'a
// des etapes NOMMEES dont la liste est fermee par un test (observe_test.go). Un balayage ajoute
// sans etape observee fait echouer ce test — c'est le but.

import (
	"log/slog"
	"strings"
	"time"
)

// Observer recoit le nom d'une etape de construction et la valeur qu'elle a produite. Les
// valeurs sont les sorties BRUTES des balayages (tranches, statistiques, tables) ; l'observateur
// ne doit ni les retenir par reference apres le retour, ni les modifier.
type Observer func(step string, v any)

// BuildFromFilmSteps est la liste FERMEE, DANS L'ORDRE, des etapes que BuildFromFilm rend a
// l'observateur. Elle est exportee pour que le harnais d'equivalence verifie qu'aucune etape ne
// manque a un fichier de digests, et gardee par observe_test.go : un balayage ajoute a
// BuildFromFilm sans etape ici fait echouer le test.
var BuildFromFilmSteps = []string{
	"positions", "fire", "loadouts",
	"heldWeaponChanges", "heldWeaponChanges.stats",
	"pickups", "pickups.stats",
	"inventory", "inventory.stats",
	"inventoryDeltas", "inventoryDeltas.stats",
	"abilityRanks", "abilityRanks.stats",
	"equipmentChanges", "equipmentChanges.stats",
	"camoStates", "camoStates.stats",
	"grappleReads", "grappleReads.stats",
	"zoomEvents",
	"placements", "placements.stats",
	"pads", "carrierMarks", "zoneReads", "bombReads",
	"grenades", "projectiles", "deaths", "playerIndices", "clockOrigin",
}

// stepClock date la FIN du balayage precedent. C'est tout ce qu'il faut pour chronometrer les
// balayages de BuildFromFilm sans y ajouter un seul site d'appel : l'observateur est deja appele
// juste apres chaque balayage, dans l'ordre, donc la duree d'un balayage est l'ecart entre deux
// appels consecutifs (PLAN_CUISSON_PERF §3 D5).
//
// POURQUOI UN POINTEUR DANS Options : `observe` est une methode SUR VALEUR — Options circule par
// copie entre les balayages, et un champ `time.Time` verrait chaque copie repartir de sa propre
// date. Le pointeur, lui, designe la meme horloge pour toute la construction.
type stepClock struct{ last time.Time }

// observe appelle l'observateur s'il y en a un. Methode sur valeur : Options circule par copie.
//
// LA MESURE PRECEDE L'OBSERVATEUR, et il le faut : un observateur qui hache la valeur (harnais
// d'equivalence) couterait des millisecondes qui ne sont pas celles du balayage.
//
// LES ETAPES `.stats` NE SONT NI JOURNALISEES NI COMPTEES : elles ne sont pas des balayages mais
// la seconde sortie du balayage precedent, emise dans la foulee. Les chronometrer mesurerait
// zero, et surtout avancer l'horloge sur elles volerait au balayage SUIVANT le temps ecoule
// entre les deux appels.
func (o Options) observe(step string, v any) {
	if o.clock != nil && !strings.HasSuffix(step, ".stats") {
		slog.Debug("replay: balayage", "step", step, "duration", time.Since(o.clock.last))
		o.clock.last = time.Now()
	}
	if o.Observe != nil {
		o.Observe(step, v)
	}
}
