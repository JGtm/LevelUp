package main

// cmd_decode_verrou.go — le verrou solo de decodage, pris par les commandes de `levelup` qui decodent un film DANS
// leur propre processus (ADR 0045 D-29 ; meme regle que l'etape 1.57 du post-sync et la file de regeneration des
// rejeux de l'application de bureau).
//
// Regle du projet : JAMAIS DEUX DECODAGES DE FILM A LA FOIS sur la machine (pic memoire cumule, `filmproc.AcquireSolo`).
// Une commande qui decode en processus (aujourd'hui `backfill-killsource`, hors ligne comme en ligne) prend le verrou
// EN REFUS IMMEDIAT avant d'ouvrir la base : un serveur qui cuit un rejeu, une file de regeneration ou une autre passe
// tient la machine, et la commande le dit et sort avec [CodeSortieVerrouTenu] au lieu de faire monter la memoire de
// deux decodages. Les passes d'enfant (`backfill-replay`) attendent leur tour a la place (`AcquireSoloWait`) : elles
// n'ont pas de cycle suivant. Garde-rail : cmd_decode_verrou_test.go.

import (
	"errors"
	"fmt"

	"levelup/go-api/internal/config"

	"github.com/JGtm/LevelUp/libs/film/filmproc"
)

// CodeSortieVerrouTenu : le code de sortie d'une commande refusee parce qu'un autre decodage de film tient la machine. Ce
// n'est PAS 2 : c'est le code de `flag.ExitOnError` (ligne de commande refusee), qu'un script doit pouvoir distinguer d'un verrou tenu.
const CodeSortieVerrouTenu = 3

// outilBackfillKillSource : le nom sous lequel `backfill-killsource` se declare dans le verrou (un operateur qui se
// voit refuser le verrou lit QUI decode).
const outilBackfillKillSource = "backfill-killsource"

// verrouDeDecodageTenu : un autre decodage tient la machine. Un refus attendu, pas une panne : sortirSur le traduit
// en [CodeSortieVerrouTenu] avec le detenteur nomme.
type verrouDeDecodageTenu struct {
	outil string
	err   error
}

func (e *verrouDeDecodageTenu) Error() string {
	return fmt.Sprintf("%s ne decode pas : %v", e.outil, e.err)
}

func (e *verrouDeDecodageTenu) Unwrap() error { return e.err }

// prendreLeVerrouDeDecodage prend le verrou solo de la machine (racine du cache film du depot) en refus immediat et
// rend sa liberation. Un refus d'occupation rend un [*verrouDeDecodageTenu] ; toute autre erreur (disque, racine vide)
// est rendue telle quelle : jamais de decodage sans le verrou.
func prendreLeVerrouDeDecodage(cfg *config.AppConfig, outil string) (func(), error) {
	lock, err := filmproc.AcquireSolo(cfg.PathResolver().CacheRootDir(), outil, "")
	if err == nil {
		return lock.Release, nil
	}
	if errors.Is(err, filmproc.ErrDecodeBusy) {
		return nil, &verrouDeDecodageTenu{outil: outil, err: err}
	}
	return nil, fmt.Errorf("%s : verrou de decodage indisponible : %w", outil, err)
}

// verrouDeLaPasse rend la liberation du verrou solo de `backfill-killsource` : il est pris quand la passe decode
// (hors `--dry-run` et `--credit-only`), sinon la liberation est sans effet.
func verrouDeLaPasse(cfg *config.AppConfig, o killsourceOptions) (func(), error) {
	if o.dryRun || o.creditOnly {
		return func() {}, nil
	}
	return prendreLeVerrouDeDecodage(cfg, outilBackfillKillSource)
}
