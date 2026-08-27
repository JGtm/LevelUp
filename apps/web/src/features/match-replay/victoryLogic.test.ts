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

import { readVictory } from './victoryLogic'

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
