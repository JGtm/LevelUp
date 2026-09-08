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
	// `noBridge` le 2026-09-08 (lot E2) : `coverage.bombCarries.noBridge` compte les portages de
	// bombe qu AUCUN pont ne nommait. Le lien direct corps -> joueur le ramene de 2 a 0 sur
	// `c75f33b8`, et le gate lisait cette disparition comme une perte — le meme defaut que
	// `noTrack` la veille, sur un compteur dont le NOM dit qu il est un echec.
	"noBridge": true,
	// 2026-09-08 (lot E2-bis) : `coverage.flagCarries.ambiguousReturns` / `ambiguousSlot` et
	// `coverage.vehicles.shotsNoRide` — des ECHECS que le lien direct fait baisser (3 -> 2, 1 -> 0,
	// 2768 -> 2265 sur `084a804d`) et que le gate lisait en perte.
	"ambiguousReturns": true,
	"ambiguousSlot":    true,
	"shotsNoRide":      true,
	// `bombStats.coverage.periodsNoBridge` (E2-bis) : periodes de portage de bombe sans pont, 2 -> 0.
	"periodsNoBridge": true,
}

// marqueurParXUID : segment des cles ventilees par joueur (`.../par-xuid/<xuid>`,
// `tracks/vies-par-xuid/<xuid>`, `.../duree-totale/par-xuid/<xuid>`).
const marqueurParXUID = "par-xuid/"

// groupeParXUID rend la cle de GROUPE d'une mesure ventilee par joueur (tout ce qui precede
// l'identifiant), et false si la mesure n'est pas ventilee.
func groupeParXUID(k string) (string, bool) {
	i := strings.LastIndex(k, marqueurParXUID)
	if i < 0 {
		return "", false
	}
	return k[:i+len(marqueurParXUID)], true
}

// groupesConserves : les groupes par joueur dont la SOMME sur tous les joueurs est la meme des
// deux cotes. Une baisse chez un joueur y est une REATTRIBUTION — ce que le lien direct
// corps -> joueur (lot E2, 2026-09-08) fait a raison quand il rend a l'un ce que le pont par
// morts attribuait a l'autre (ramassages : 114, 402, 332... identiques ; trajets en vehicule :
// 74 et 28 995 frames conserves) — pas une perte. Une somme qui baisse reste une perte a
// instruire ; une somme qui monte, un gain.
func groupesConserves(a, b map[string]Mesure) map[string]bool {
	sa, sb := map[string]float64{}, map[string]float64{}
	presents := map[string]bool{}
	for k, m := range a {
		if g, ok := groupeParXUID(k); ok && m.EstNum {
			sa[g] += m.Num
			presents[g] = true
		}
	}
	for k, m := range b {
		if g, ok := groupeParXUID(k); ok && m.EstNum {
			sb[g] += m.Num
			presents[g] = true
		}
	}
	out := map[string]bool{}
	for g := range presents {
		// Somme conservee OU en hausse : la part attribuee peut monter (des ramassages qui
		// n'avaient pas d'auteur en trouvent un) pendant qu'un joueur en perd au profit d'un
		// autre — c'est encore une reattribution. Seule une somme qui BAISSE est une perte.
		if proches(sa[g], sb[g]) || sb[g] > sa[g] {
			out[g] = true
		}
	}
	return out
}

// estReattribution dit si la mesure `k` appartient a un groupe par joueur conserve.
func estReattribution(k string, conserves map[string]bool) bool {
	g, ok := groupeParXUID(k)
	return ok && conserves[g]
}

// prefixesMethode : les compteurs `coverage.bridge.namedBy*` disent PAR QUELLE VOIE une vie a ete
// nommee (fil des morts, vie voisine, fermeture, exclusion...). Ils se deplacent entre eux quand
// une voie plus sure prend le pas sur une voie de repli (P2-bis, 2026-09-08 : `namedByNextLife`
// 13 -> 8 sur `d9781168` parce que l'exclusion temporelle nomme d'abord) : ni gain ni perte, un
// CHANGEMENT. La richesse, elle, se lit sur `livesNamed` / `unnamedLives`.
//
// `coverage.bridge.closedBy*` (2026-09-08, lot E2) EST LA MEME FAMILLE, et pour la meme raison :
// `closedByShot` et `closedByRespawn` disent par quelle preuve une FERMETURE a comble le pont —
// une DEDUCTION, qui ne s'applique par construction qu'aux slots que la lecture n'a pas nommes.
// Quand le lien direct corps -> joueur nomme 100 % des corps (mesure du lot : 5 films sur 5), les
// fermetures n'ont plus rien a fermer et ces deux compteurs tombent a ZERO. Ce n'est pas une
// richesse perdue — c'est une voie de repli devenue inutile, et `livesNamed` / `unnamedLives` le
// disent a leur place.
//
// `coverage.flagCarries.homeBy*` / `assignedBy*` (2026-09-08, lot E2-bis) : par quelle voie un
// retour ou une attribution de drapeau a ete tranche (objet, marqueur, jeu...) — meme famille.
var prefixesMethode = []string{
	"coverage.bridge.namedBy", "coverage.bridge.closedBy",
	"coverage.flagCarries.homeBy", "coverage.flagCarries.assignedBy",
}

// estCompteurDeMethode dit si la mesure `k` est un compteur de voie de nommage.
func estCompteurDeMethode(k string) bool {
	_, chemin := decouper(k)
	for _, p := range prefixesMethode {
		if strings.HasPrefix(chemin, p) {
			return true
		}
	}
	return false
}

// estCompteurDEchec dit si la mesure `k` (cle d'empreinte `axe/chemin`, ex.
// `couverture/coverage.objectives.unpublished`, cf. `cle()`) est un compteur d'echec de la
// couverture. L'axe est ignore : c'est le chemin aplati qui porte le sens.
func estCompteurDEchec(k string) bool {
	_, chemin := decouper(k)
	// La couverture vit en tete du document (`coverage.*`) ou dans un calque qui porte la
	// sienne (`bombStats.coverage.*`) : c'est le SEGMENT `coverage` qui compte, pas sa position.
	if !strings.HasPrefix(chemin, prefixeCouverture) && !strings.Contains(chemin, "."+prefixeCouverture) {
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
