// Package filmbacklog tient, par titre, l'arriéré de films que les deux étapes film du post-sync
// n'ont pas encore traités : la source des frags (killcollector) et les artefacts de rejeu
// (replayartifacts). Chaque étape publie ce qui lui reste à la fin de sa passe ; l'ordonnanceur et
// l'écran d'état de l'historique le lisent par le port port.FilmBacklogReader (Reader).
//
// L'arriéré est celui du TITRE, pas d'un joueur : les deux étapes lisent un registre de matchs
// partagé par tous les joueurs du titre, et une seule passe de la source des frags tourne à la fois
// par titre. La valeur publiée est celle de la dernière passe qui a lu l'arriéré ; tant qu'aucune
// passe n'a tourné (titre sans l'étape, processus neuf), elle vaut zéro.
//
// Ce que l'arriéré compte : les matchs que la passe a laissés derrière elle ET qu'une passe suivante
// peut encore traiter (non traités faute de temps, reportés à la fin d'une partie ou d'un autre
// décodage). Il ne compte pas un match que la passe a essayé et dont elle n'a pas pu venir à bout
// (échec, film refusé), ni un match dont la carte n'est pas résolue : une cadence courte qui les
// reprendrait sans fin ne les ferait pas avancer.
package filmbacklog

import (
	"sync"

	"levelup/go-api/internal/port"
)

// Etape désigne l'étape film qui publie.
type Etape int

const (
	// Frags : la source des frags (étape 1.57, killcollector).
	Frags Etape = iota
	// Rejeux : les artefacts de rejeu (étape 1.58, replayartifacts).
	Rejeux
)

var (
	mu       sync.Mutex
	parTitre = map[string]port.FilmBacklog{}
)

// Publier range l'arriéré d'une étape pour un titre ; une valeur négative vaut zéro.
func Publier(titre string, etape Etape, restant int) {
	restant = max(restant, 0)
	mu.Lock()
	defer mu.Unlock()
	b := parTitre[titre]
	switch etape {
	case Frags:
		b.Frags = restant
	case Rejeux:
		b.Rejeux = restant
	}
	parTitre[titre] = b
}

// Lire rend l'arriéré publié pour un titre.
func Lire(titre string) port.FilmBacklog {
	mu.Lock()
	defer mu.Unlock()
	return parTitre[titre]
}

// Reset oublie tout l'arriéré publié. Pour les tests : l'état est celui du processus.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	parTitre = map[string]port.FilmBacklog{}
}

// Reader lit l'arriéré publié : il satisfait port.FilmBacklogReader.
type Reader struct{}

var _ port.FilmBacklogReader = Reader{}

// FilmBacklog rend l'arriéré publié pour le titre.
func (Reader) FilmBacklog(titleSlug string) port.FilmBacklog { return Lire(titleSlug) }

// ReaderFor rend le lecteur d'arriéré que l'ordonnanceur et l'écran d'état de l'historique consultent, selon la
// capacité FilmCatchUp du profil de déploiement : le [Reader] publié quand elle est vraie ; rien (nil) quand elle
// est fausse, car la cadence de rattrapage et l'état affiché ne suivent alors que l'historique, jamais l'arriéré
// de films. Les étapes film publient dans les deux cas.
func ReaderFor(filmCatchUp bool) port.FilmBacklogReader {
	if !filmCatchUp {
		return nil
	}
	return Reader{}
}
