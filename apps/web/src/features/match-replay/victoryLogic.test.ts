/**
 * Tests — readVictory (comment le match s'est terminé pour le joueur de la page).
 *
 * Ce que ces tests protègent : une lecture fausse ne casse rien à l'exécution, elle HABILLE
 * L'ÉCRAN DE FIN AVEC LA MAUVAISE ÉQUIPE, en plein cadre. Trois erreurs sont plausibles et
 * couvertes ici : confondre « mon équipe » et « l'équipe qui gagne » sur une défaite (elles
 * diffèrent, et c'est le cœur de l'amendement du 2026-08-26), inverser le camp adverse, et
 * afficher l'écran là où il n'a pas de sens (FFA, trois camps, abandon).
 */
import { describe, expect, it } from 'vitest'

import { finalScoreFromHeader, readVictory } from './victoryLogic'

/** Une ligne de scoreboard réduite à ce que la lecture regarde. */
function row(side: string | null, isMe = false) {
  return { team_side: side, is_me: isMe }
}

/** Le cas de référence : deux camps, le joueur de la page dans `t0`. */
const DEUX_CAMPS = [row('t0', true), row('t0'), row('t1'), row('t1')]

describe('readVictory — l’issue vient du code d’en-tête, pas du score', () => {
  it('victoire (code 2) : mon équipe habille l’écran, et c’est elle qui gagne', () => {
    expect(readVictory(DEUX_CAMPS, 2)).toEqual({
      outcome: 'win',
      mine: { teamID: 0, teamSide: 't0', ally: true },
      winner: { teamID: 0, teamSide: 't0', ally: true },
    })
  })

  it('défaite (code 3) : mon équipe habille TOUJOURS l’écran, le vainqueur est l’AUTRE', () => {
    expect(readVictory(DEUX_CAMPS, 3)).toEqual({
      outcome: 'loss',
      mine: { teamID: 0, teamSide: 't0', ally: true },
      winner: { teamID: 1, teamSide: 't1', ally: false },
    })
  })

  it('le camp adverse est nommé par son team_side réel, pas par « l’autre numéro »', () => {
    // Camps t2 et t5 : un calcul par complément (1 − index) sur les IDENTIFIANTS donnerait
    // n'importe quoi. La lecture indexe les CAMPS, pas les numéros d'équipe.
    const exotique = [row('t5', true), row('t2')]
    expect(readVictory(exotique, 3)).toEqual({
      outcome: 'loss',
      mine: { teamID: 5, teamSide: 't5', ally: true },
      winner: { teamID: 2, teamSide: 't2', ally: false },
    })
  })

  it('égalité (code 1) : aucune équipe rendue — la neutralité est dans la donnée', () => {
    expect(readVictory(DEUX_CAMPS, 1)).toEqual({ outcome: 'tie', mine: null, winner: null })
  })
})

describe('readVictory — les situations où aucun écran ne doit s’afficher', () => {
  it('FFA (aucun camp transmis) : null', () => {
    expect(readVictory([row(null, true), row(null), row(null)], 2)).toBeNull()
  })

  it('trois camps : null — un écran à deux camps ne peut pas dire un match à trois', () => {
    const troisCamps = [row('t0', true), row('t1'), row('t2')]
    expect(readVictory(troisCamps, 2)).toBeNull()
  })

  it('un seul camp identifié : null', () => {
    expect(readVictory([row('t0', true), row('t0')], 2)).toBeNull()
  })

  it('code d’issue absent : null (rien à annoncer, on n’invente pas de résultat)', () => {
    expect(readVictory(DEUX_CAMPS, undefined)).toBeNull()
    expect(readVictory(DEUX_CAMPS, null)).toBeNull()
  })

  it('abandon (code 4) : null — un match quitté ne se conclut pas', () => {
    expect(readVictory(DEUX_CAMPS, 4)).toBeNull()
  })

  it('code hors contrat (0, 99) : null', () => {
    expect(readVictory(DEUX_CAMPS, 0)).toBeNull()
    expect(readVictory(DEUX_CAMPS, 99)).toBeNull()
  })

  it('scoreboard sans ligne `is_me` : null — le pont vers mon camp manque', () => {
    expect(readVictory([row('t0'), row('t1')], 2)).toBeNull()
  })

  it('ligne `is_me` sans camp transmis : null', () => {
    const sansCamp = [row(null, true), row('t0'), row('t1')]
    expect(readVictory(sansCamp, 2)).toBeNull()
  })

  it('égalité en FFA : null aussi — le panneau annonce la fin d’un duel de camps', () => {
    expect(readVictory([row(null, true), row(null)], 1)).toBeNull()
  })
})

// ─── finalScoreFromHeader : le résultat vient de l'API sur un mode à manches ─

describe('finalScoreFromHeader', () => {
  it("rend le compte de manches quand l'en-tête le dit", () => {
    // Témoin 293a763e : 2 manches à 1, alors que le calque du film rendrait « 100 - 43 ».
    expect(
      finalScoreFromHeader({ score_kind: 'rounds', score_mine: 2, score_theirs: 1 }),
    ).toEqual({ ally: 2, enemy: 1 })
  })

  // Le critère est la PRÉSENCE des deux nombres, jamais leur nature. Une variante à manches
  // dont les camps finissent à égalité (témoin adb93fb7 : 1 partout + 1 nulle) retombe côté
  // serveur sur les points et publie score_kind = "points" ; filtrer sur « rounds » renverrait
  // alors l'écran de fin vers les points de la DERNIÈRE MANCHE, en contradiction avec la vue
  // match qui affiche le total.
  it("rend aussi le score quand il est en points (repli d'une égalité de manches)", () => {
    expect(
      finalScoreFromHeader({ score_kind: 'points', score_mine: 277, score_theirs: 234 }),
    ).toEqual({ ally: 277, enemy: 234 })
  })

  it('rend null quand les nombres manquent (ligne antérieure au backfill)', () => {
    expect(finalScoreFromHeader({ score_kind: 'rounds' })).toBeNull()
    expect(finalScoreFromHeader({ score_kind: 'points', score_mine: 50 })).toBeNull()
    expect(finalScoreFromHeader(undefined)).toBeNull()
  })

  it('accepte un zéro : 2 manches à 0 est une mesure, pas une absence', () => {
    expect(
      finalScoreFromHeader({ score_kind: 'rounds', score_mine: 2, score_theirs: 0 }),
    ).toEqual({ ally: 2, enemy: 0 })
  })
})
