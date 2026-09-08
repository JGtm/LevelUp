package replaydiff

// polarite.go — LES COMPTEURS D'ECHEC SE LISENT A L'ENVERS.
//
// `classerNombres` nomme « perte » toute mesure qui BAISSE : c'est juste pour tout ce qui
// mesure une richesse (vies nommees, actions rattachees, portages, durees). C'est faux pour les
// compteurs que la couverture publie pour dire ce que la cuisson N'A PAS su faire : une action
// d'objectif `unpublished`, une vie `unnamedLives`, un tir `noSlot`, un desaccord d'index. Pour
// eux, une baisse est le gain que le chantier cherche (plan v2 §1 : « compteur d'echec qui
// baisse = gain »), et une hausse est la perte a instruire.
//
// Constate le 2026-09-08 sur le lot P2 (registre des joueurs) : `coverage.objectives.unpublished`
// 35 -> 0 et `coverage.shots.noSlot` 35 -> 15 sortaient en PERTE, et le gate refusait exactement
// le progres qu'il devait garder. La liste est FERMEE et nommee : un compteur absent d'ici garde
// la lecture generique (plus = mieux), et un nouveau compteur d'echec s'ajoute ici avec sa date.
//
// `deathOffsetRunnerUp` n'y est PAS : c'est un nombre de voix, pas un echec — sa marge se lit
// contre `deathOffsetMatched`, et une lecture inversee en ferait une perte a chaque voix
// de plus.

import "strings"

// prefixeCouverture : seules les mesures de l'axe `coverage` sont candidates a l'inversion.
const prefixeCouverture = "coverage."

// compteursDEchec : dernier segment de cle -> compteur d'echec (une baisse est un gain).
// Ajouts dates : 2026-09-08 (lot P2) ; `noTrack` le 2026-09-08 (P2-bis, `flagCarries.noTrack`).
var compteursDEchec = map[string]bool{
	"unpublished":           true,
	"unnamedLives":          true,
	"unnamedLivesContested": true,
	"noSlot":                true,
	"noTrack":               true,
	"outOfWindow":           true,
	"ambiguous":             true,
	"closedRefused":         true,
	"closedContested":       true,
	"indexDisagreements":    true,
	"slotCollisions":        true,
}

// prefixeMethode : les compteurs `coverage.bridge.namedBy*` disent PAR QUELLE VOIE une vie a ete
// nommee (fil des morts, vie voisine, fermeture, exclusion...). Ils se deplacent entre eux quand
// une voie plus sure prend le pas sur une voie de repli (P2-bis, 2026-09-08 : `namedByNextLife`
// 13 -> 8 sur `d9781168` parce que l'exclusion temporelle nomme d'abord) : ni gain ni perte, un
// CHANGEMENT. La richesse, elle, se lit sur `livesNamed` / `unnamedLives`.
const prefixeMethode = "coverage.bridge.namedBy"

// estCompteurDeMethode dit si la mesure `k` est un compteur de voie de nommage.
func estCompteurDeMethode(k string) bool {
	_, chemin := decouper(k)
	return strings.HasPrefix(chemin, prefixeMethode)
}

// estCompteurDEchec dit si la mesure `k` (cle d'empreinte `axe/chemin`, ex.
// `couverture/coverage.objectives.unpublished`, cf. `cle()`) est un compteur d'echec de la
// couverture. L'axe est ignore : c'est le chemin aplati qui porte le sens.
func estCompteurDEchec(k string) bool {
	_, chemin := decouper(k)
	if !strings.HasPrefix(chemin, prefixeCouverture) {
		return false
	}
	i := strings.LastIndexByte(chemin, '.')
	return compteursDEchec[chemin[i+1:]]
}

// inverserSens retourne le sens d'un ecart pour un compteur d'echec : une baisse (perte
// generique) devient un gain, une hausse une perte ; un compteur qui apparait (> 0) est une
// perte, un compteur qui disparait est un gain. Un changement textuel reste un changement.
func inverserSens(sens string) string {
	switch sens {
	case SensPerte:
		return SensGain
	case SensGain:
		return SensPerte
	case SensApparu:
		return SensPerte
	case SensDisparu:
		return SensGain
	}
	return sens
}
