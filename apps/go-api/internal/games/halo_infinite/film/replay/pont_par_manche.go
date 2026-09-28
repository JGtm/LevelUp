package replay

// pont_par_manche.go — LE PONT SLOT D'ENTITE STATBORG -> XUID, PAR MANCHE, DE LA CUISSON ET DU SYNC.
//
// DEPLACE depuis `replaybuild/matchfacts.go` le 2026-09-28 (lot V1.4 du plan
// `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`), sans changement de regle : le collecteur de sync lit
// desormais les porteurs d'objectif par l'assembleur de production (`porteurs_au_sync.go`), et ce
// pont nomme les porteurs du drapeau et du crane. Deux producteurs du meme fait ne peuvent pas
// resoudre deux ponts : il vit ICI, ou les deux le trouvent, et `replaybuild` l'appelle.
//
// LES LIGNES DE MATCH N'Y ENTRENT QUE COMME DONNEES (`types.PlayerLine`, xuid et compteurs) : ce
// paquet ne lit toujours aucune base. Sans lignes, `CompletedByLines` rend le pont par morts
// inchange et les calques restent publiables hors ligne.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// PontParManche est LE pont slot d'entite -> xuid : resolu par manche via les instants de mort,
// COMPLETE par le triplet quand le film est mono-manche et que les lignes de match sont la. Coeur
// PUR, sans I/O — testable sans film.
//
// POURQUOI IL EST MEMORISE, ET POURQUOI IL EST PARESSEUX. Plusieurs calques le consomment (les
// ACTIONS d'objectif, le DRAPEAU VIVANT, le PORTEUR DU CRANE) et le resolvaient chacun de leur cote
// sur les MEMES enregistrements et le MEME fil des morts — autant de deroulages complets du
// compteur de morts par cuisson. Il est memorise pour n'en payer qu'un ; il est PARESSEUX pour n'en
// payer AUCUN sur les films qu'aucun calque ne lit (le deroulage sur un film d'une autre grammaire
// est ce qui montait a 19-22 Go avant la garde du 2026-08-18).
type PontParManche struct {
	recs   []types.StatRecord
	deaths []types.DeathInstant
	lines  []types.PlayerLine
	resolu bool
	id     objectives.RoundIdentity
}

// NouveauPontParManche prepare le pont, sans le resoudre. `lines` vide = pont par morts seul.
func NouveauPontParManche(recs []types.StatRecord, deaths []types.DeathInstant,
	lines []types.PlayerLine) *PontParManche {
	return &PontParManche{recs: recs, deaths: deaths, lines: lines}
}

// Identite rend le pont, en le resolvant au premier appel.
//
// QUATRE VOIES CHAINEES, DANS L'ORDRE DE LA FORCE DE PREUVE (lot P2, 2026-09-08 ; lot 6.7-B1,
// 2026-09-10) : les instants de mort, puis le triplet de la feuille (MONO-MANCHE seulement — le
// triplet apparie des totaux de match), puis l'ELIMINATION par manche, qui ne suppose rien du
// contenu et se controle sur le residu de la feuille, puis le RESIDU DE MANCHE (MULTI-MANCHE
// seulement), qui produit l'appariement que l'elimination se contentait de controler des que la
// manche laisse PLUSIEURS slots muets. Sans les deux dernieres, les ACTIONS d'objectif d'un
// joueur qui meurt moins de trois fois dans une manche restaient sans auteur alors que les
// COMPTEURS, eux, allaient etre completes par le meme mecanisme (`buildPlayerScores`) — deux
// lecteurs du meme pont n'auraient plus dit la meme chose du meme match.
func (p *PontParManche) Identite() objectives.RoundIdentity {
	if !p.resolu {
		p.id = objectives.ResolveRoundIdentity(p.recs, p.deaths).
			CompletedByLines(p.recs, p.lines).
			CompletedByElimination(p.recs, p.lines).
			CompletedByRoundResidue(p.recs, p.lines)
		p.resolu = true
	}
	return p.id
}
