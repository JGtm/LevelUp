/**
 * ReplayPlayerCard.stances.test.tsx — LA FICHE N'AFFICHE AUCUN ÉTAT DE MOUVEMENT.
 *
 * DÉCISION UTILISATEUR DU 2026-09-23 (Q16 du plan des retours du rejeu, lot L1.1) : les états
 * de mouvement (accroupi, glissade, escalade, sprint, saut dérivé) sont RETIRÉS de la fiche du
 * joueur. Ils y étaient arrivés le 21/09 (schémas 65 à 68) sur une décision du pilote, jamais
 * demandée. Le document garde `stances[]` (aucune montée de schéma) : c'est la FICHE qui se tait.
 *
 * Le test pose un document qui porte les cinq genres, chacun couvrant l'image lue, et vérifie
 * qu'aucun de leurs libellés (FR comme EN) n'apparaît sur la fiche vivante.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { ReplayTeams } from './ReplayTeams'
import { testReplayDoc } from '../test/testDoc'

/** Les libellés qu'affichait la fiche avant le retrait, FR puis EN. */
const LIBELLES = [
  'Accroupi', 'Glissade', 'Escalade', 'Saut (dérivé)', 'Sprint',
  'Crouched', 'Slide', 'Clamber', 'Jump (derived)',
]

const GENRES = ['crouch', 'slide', 'clamber', 'jumpDerived', 'sprint'] as const

function documentAvecEtats() {
  return testReplayDoc({
    roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
    tracks: [{ slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] }],
    stances: GENRES.map((kind) => ({ kind, slot: 512, t0: 0, t1: 100 })),
  })
}

describe('ReplayPlayerCard — aucun état de mouvement (décision du 2026-09-23)', () => {
  for (const locale of ['fr', 'en'] as const) {
    it(`la fiche vivante ne nomme aucun état, même sur un document qui en porte (${locale})`, () => {
      const doc = documentAvecEtats()
      // Le document porte bien les cinq genres : c'est la fiche qui se tait, pas la donnée.
      expect(doc.stances).toHaveLength(GENRES.length)
      const { container } = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale={locale} />)
      const texte = container.textContent ?? ''
      // La fiche est bien rendue (le nom est là) : l'absence ne vient pas d'une fiche vide.
      expect(texte).toContain('Alpha')
      for (const libelle of LIBELLES) {
        expect(texte, `libellé d'état « ${libelle} » affiché`).not.toMatch(new RegExp(libelle.replace(/[()]/g, '\\$&'), 'i'))
      }
    })
  }
})
