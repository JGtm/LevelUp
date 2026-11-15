package replaybuild

// cadence.go — LE PAS DE CUISSON DE LEVELUP, ET LE CRITERE DE FRAICHEUR QUI LE LIT.
//
// Un rejeu est A JOUR quand son schema est le schema courant ET que son pas (`frameIntervalMs` du
// document) est le pas de cuisson courant. Le numero de schema seul ne suffit pas : un meme schema
// peut etre cuit a 100 ms puis a 33 ms, et les deux rejeux se ressemblent par leur numero.
//
// UNE SEULE SOURCE, [CookFrameIntervalMS] : c'est la valeur que `buildReplayOptions` passe au
// decodeur a chaque cuisson ET celle que [Digest.Verdict] exige. Changer la cadence de LevelUp,
// c'est changer ce seul point ; `TestCuissonEtCritereLisentLeMemePas` echoue si l'un des deux se
// met a lire autre chose, et `TestAucuneLectureDuPasDuDecodeurHorsDeCadenceGo` si un autre fichier
// relit le defaut du decodeur.
//
// UN DOCUMENT SANS PAS SE LIT A 100 ms ([HistoricalFrameIntervalMS]) : c'est la grille de tous les
// rejeux d'avant le champ, et la seule que leur axe de temps peut avoir.

import (
	"sync/atomic"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// HistoricalFrameIntervalMS : le pas d'un artefact qui ne declare pas `frameIntervalMs`. Il ne bouge
// jamais, et il n'est pas le pas de cuisson : c'est un fait de l'histoire des fichiers. La valeur est
// celle du decodeur, qui lit un document sans pas a cette grille ; LevelUp n'en garde pas de copie.
const HistoricalFrameIntervalMS = replay.HistoricalFrameIntervalMS

// cookFrameIntervalOverride : un pas de cuisson impose par un test (0 = aucun). Jamais pose en production.
var cookFrameIntervalOverride atomic.Int64

// CookFrameIntervalMS rend le pas de cuisson courant de LevelUp, en millisecondes.
func CookFrameIntervalMS() int {
	if v := cookFrameIntervalOverride.Load(); v > 0 {
		return int(v)
	}
	return replay.DefaultFrameIntervalMS
}

// ForceCookFrameIntervalMS impose un pas de cuisson et rend la fonction qui le retire. RESERVE AUX
// TESTS : ils comparent un rejeu a une cadence courante qu'ils choisissent.
func ForceCookFrameIntervalMS(ms int) (restore func()) {
	cookFrameIntervalOverride.Store(int64(ms))
	return func() { cookFrameIntervalOverride.Store(0) }
}

// FrameIntervalOrHistorical rend le pas que porte un document (0 = absent) ou, absent, le pas
// historique.
func FrameIntervalOrHistorical(frameIntervalMS int) int {
	if frameIntervalMS > 0 {
		return frameIntervalMS
	}
	return HistoricalFrameIntervalMS
}

// CadenceUpToDate dit si un document au pas donné (0 = absent) est au pas de cuisson courant.
func CadenceUpToDate(frameIntervalMS int) bool {
	return FrameIntervalOrHistorical(frameIntervalMS) == CookFrameIntervalMS()
}

// SameCadence dit si deux pas (0 = absent, lu a 100 ms) sont la meme grille : la marque de fraicheur d'un
// derive projete d'un artefact (resume d'usage, raster tactique) face a l'artefact dont il vient.
func SameCadence(a, b int) bool {
	return FrameIntervalOrHistorical(a) == FrameIntervalOrHistorical(b)
}

// cadenceCourante : le pas de cet artefact est-il le pas de cuisson courant ?
func (d Digest) cadenceCourante() bool { return CadenceUpToDate(d.FrameIntervalMS) }

// frameInterval : le pas que cette cuisson passe au decodeur. Un reglage explicite (outil de recherche
// `replay-build --interval`) gagne ; sans lui, c'est le pas de cuisson courant, jamais le defaut du decodeur.
func (b *Builder) frameInterval() int {
	if b.interval > 0 {
		return b.interval
	}
	return CookFrameIntervalMS()
}
