package analysis

import "fmt"

// team_score_display.go — CE QU'ON AFFICHE COMME SCORE D'UN MATCH : des points, ou des
// MANCHES. Source UNIQUE de la règle, title-agnostic, sans I/O (ADR 0032).
//
// LE PROBLÈME QU'ELLE TRANCHE. Sur un mode qui se décide aux manches, le score rendu par
// l'API (`CoreStats.Score`) est un CUMUL DE POINTS sur toutes les manches : il ne dit pas
// qui a gagné, et il peut même donner l'avantage à l'équipe qui a perdu (Oddball). Le
// compte de manches, lui, tranche.
//
// DEUX CONDITIONS CUMULATIVES POUR AFFICHER LES MANCHES. Dès qu'elles tiennent, on affiche
// le nombre de manches remportées par chaque équipe, QUEL QUE SOIT LE RÉSULTAT — victoire,
// défaite, égalité de manches, match abandonné :
//
//  1. La VARIANTE est déclarée dans `regulation.toml [rounds_decide]` — une table MESURÉE,
//     jamais une heuristique. La règle « plus d'une manche donc on affiche les manches »
//     est RÉFUTÉE par la mesure : le CTF d'arène se joue en deux MI-TEMPS
//     (`rounds_total = 2`) alors que son score est le total de captures. La règle y aurait
//     affiché « 0 - 1 » à la place de « 2 - 3 ».
//  2. Le match a RÉELLEMENT plusieurs manches (`rounds_total >= 2`). Une variante déclarée
//     dont un match particulier n'a qu'une manche garde les points : le compte de manches
//     n'y ajouterait rien.
//
// Une égalité de manches n'est PAS une condition de repli : « 1 - 1 » est ce que le match a
// joué, même quand une manche nulle ou un abandon laisse un vainqueur par ailleurs. Le
// verdict (victoire/défaite) voyage séparément (`outcome`), il n'est jamais déduit de ce
// libellé.
//
// TOUT ÉCHEC DE CONDITION REVIENT AUX POINTS : une donnée manquante (colonne NULL, ligne
// antérieure au backfill, titre dont l'API ne publie pas les manches comme Halo 5) ne
// dégrade rien, elle ne gagne simplement pas la lecture en manches. Le comptage de manches
// exige aussi les deux camps connus et non négatifs.
//
// LES POINTS NE SONT JAMAIS PERDUS. Même quand on affiche les manches, `Points` reste
// renseigné : la vue match les montre en second plan, et un appelant qui n'en veut pas les
// ignore.

// ScoreKind dit CE QUE portent `Mine` et `Theirs` : des points de mode, ou des manches.
type ScoreKind string

const (
	// ScoreKindPoints : le score du mode tel que l'API le rend. Cas par défaut.
	ScoreKindPoints ScoreKind = "points"
	// ScoreKindRounds : des manches gagnées — le résultat, pas le cumul de points.
	ScoreKindRounds ScoreKind = "rounds"
)

// TeamScoreInput porte tout ce qu'il faut pour trancher, du POINT DE VUE DU JOUEUR de la
// page : « Mine » est son camp, « Enemy » l'autre. La permutation camp 0 / camp 1 est faite
// par l'appelant, qui seul sait dans quelle équipe joue le joueur affiché.
//
// Les pointeurs nil disent « inconnu », jamais zéro : une colonne NULL et un vrai zéro sont
// deux affirmations différentes (« on ne sait pas » contre « zéro manche gagnée »).
type TeamScoreInput struct {
	MyPoints, EnemyPoints       *int
	MyRoundsWon, EnemyRoundsWon *int
	// RoundsTotal est le nombre de manches JOUÉES (max des deux camps, cf.
	// sync.ExtractTeamRoundsByID).
	RoundsTotal *int
	// RoundsDecide : la variante est déclarée dans `regulation.toml [rounds_decide]`.
	RoundsDecide bool
}

// TeamScoreDisplay est la lecture à afficher.
type TeamScoreDisplay struct {
	Kind ScoreKind
	// Mine / Theirs : les deux nombres à écrire, dans l'unité de Kind.
	Mine, Theirs int
	// Points : le score de l'API, TOUJOURS renseigné quand il est connu — y compris sur
	// une lecture en manches, où il devient l'information secondaire. Nil = inconnu.
	Points *[2]int
}

// ReadTeamScore rend la lecture à afficher, et false quand il n'y a RIEN à afficher —
// aucun score connu et aucune manche exploitable. Un appelant qui reçoit false n'écrit pas
// « 0 - 0 » : il n'écrit rien (c'est déjà la doctrine des libellés de score actuels).
func ReadTeamScore(in TeamScoreInput) (TeamScoreDisplay, bool) {
	points, pointsKnown := readPoints(in)
	if roundsWin(in) {
		d := TeamScoreDisplay{Kind: ScoreKindRounds, Mine: *in.MyRoundsWon, Theirs: *in.EnemyRoundsWon}
		if pointsKnown {
			p := points
			d.Points = &p
		}
		return d, true
	}
	if !pointsKnown {
		return TeamScoreDisplay{}, false
	}
	p := points
	return TeamScoreDisplay{Kind: ScoreKindPoints, Mine: p[0], Theirs: p[1], Points: &p}, true
}

// roundsWin applique les deux conditions cumulatives de l'en-tête, plus la lisibilité du compte
// (deux camps connus, non négatifs). Aucune comparaison des deux nombres : l'égalité s'affiche.
func roundsWin(in TeamScoreInput) bool {
	if !in.RoundsDecide || in.RoundsTotal == nil || *in.RoundsTotal < 2 {
		return false
	}
	if in.MyRoundsWon == nil || in.EnemyRoundsWon == nil {
		return false
	}
	return *in.MyRoundsWon >= 0 && *in.EnemyRoundsWon >= 0
}

// readPoints rend le couple de points et s'il est exploitable.
//
// Un score NÉGATIF est refusé, comme le faisaient déjà les cinq constructeurs de libellé
// remplacés par ce module : il ne peut pas être un score de mode, et l'afficher donnerait
// du crédit à une donnée abîmée.
func readPoints(in TeamScoreInput) ([2]int, bool) {
	if in.MyPoints == nil || in.EnemyPoints == nil {
		return [2]int{}, false
	}
	if *in.MyPoints < 0 || *in.EnemyPoints < 0 {
		return [2]int{}, false
	}
	return [2]int{*in.MyPoints, *in.EnemyPoints}, true
}

// TeamScoreLabel rend le libellé « X - Y » d'un match, ou "" quand il n'y a rien à
// afficher. C'est la SOURCE UNIQUE du libellé de score d'équipe.
//
// POURQUOI ELLE EXISTE. Avant le 2026-08-29, cinq endroits fabriquaient ce libellé —
// en-tête de vue match, historique (qui alimente aussi l'Explorateur et la carrière), page
// coéquipiers, et les deux chemins de l'accueil — avec DEUX formats différents (`%d-%d` et
// `%d - %d`) et deux sources de données. Poser la règle « manches plutôt que points » sur
// cinq copies l'aurait fait diverger cinq fois ; l'audit l'avait relevé comme la dette n°2
// du chantier (règle 6 du CLAUDE.md, ≤ 2 copies).
//
// LE FORMAT EST UNIFIÉ sur « X - Y » (espaces autour du tiret) : c'était déjà celui de trois
// des cinq appelants, et c'est le plus lisible sur un grand nombre.
//
// Le libellé ne dit PAS s'il parle de points ou de manches : cette information voyage
// séparément (`ScoreKind` au contrat d'API) pour rester localisable côté client. Un libellé
// qui porterait « manches » en dur serait un mot anglais ou français figé côté serveur.
func TeamScoreLabel(in TeamScoreInput) string {
	d, ok := ReadTeamScore(in)
	if !ok {
		return ""
	}
	return FormatTeamScoreLabel(d)
}

// FormatTeamScoreLabel écrit une lecture déjà tranchée. Utile à l'appelant qui a besoin de
// la lecture ET du libellé sans repasser par la règle.
func FormatTeamScoreLabel(d TeamScoreDisplay) string {
	return fmt.Sprintf("%d - %d", d.Mine, d.Theirs)
}
