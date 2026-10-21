/**
 * pionsSurFiches.test.ts — L'INVARIANT DU PION (utilisateur, 2026-10-21 : « on a autant de joueurs
 * sur le terrain que sur les fiches ») : à tout instant, les pions dessinés d'une équipe sont des
 * joueurs qui occupent une fiche, donc il n'y en a jamais plus que de fiches occupées.
 *
 * LE CRITÈRE DE DESSIN EST CELUI DU CANEVAS : une vie vivante à l'image, dont la couleur de slot
 * (`gatedByFiche`) n'est pas nulle — c'est ce que lit `drawTracksLayer`. Il est joué sur les huit
 * documents de contrat produits par Go (un par build du jeu). Le cas du bot relevé (`3d58eb37` :
 * présence bornée à 319, vies jusqu'à 3 777) est tenu à part, sur pièces synthétiques, par
 * `seatedIndex.test.ts` et `replayMarkers.seated.test.ts`.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import { isAliveAt } from '@/lib/replay/replayLogic'
import { normalizeReplayDocument } from '@/lib/replay/replayNormalize'
import { buildPlayers, buildSlotOwnership } from '@/lib/replay/rosterLogic'

import { gatedByFiche, seatedSlotResolver } from '../layers/useSlotIdentity'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { buildSeats, seatedIndexOf, seatOccupantAt } from './seatLogic'

/** Un pas d'échantillonnage : assez fin pour voir une relève, assez large pour 6 000 images. */
const PAS = 5

interface Bilan {
  /** Images où une équipe dessine plus de pions que de fiches occupées, avec la règle du pion. */
  depassements: number
  /** La même mesure SANS la règle : ce que le canevas dessinait avant. */
  depassementsSansRegle: number
  /** Pions dessinés dont le joueur n'occupe aucune fiche à l'image. */
  pionsSansFiche: number
}

function mesurer(raw: ReplayDocument): Bilan {
  const doc = normalizeReplayDocument(raw)
  const players = buildPlayers(doc, [])
  const ownership = buildSlotOwnership(players)
  const seats = buildSeats(players, doc)
  const seated = seatedIndexOf(seats)
  const gate = gatedByFiche(() => 'encre', seatedSlotResolver(ownership, seated))
  const bilan: Bilan = { depassements: 0, depassementsSansRegle: 0, pionsSansFiche: 0 }
  const tracks = players.flatMap((p) => p.lives.map((life) => ({ p, life })))
  for (let f = 0; f < doc.frameCount; f += PAS) {
    const fiches = new Map<number, number>()
    for (const s of seats) {
      if (seatOccupantAt(s, f).player !== null) fiches.set(s.team, (fiches.get(s.team) ?? 0) + 1)
    }
    const avec = new Map<number, number>()
    const sans = new Map<number, number>()
    for (const { p, life } of tracks) {
      if (p.team === undefined || !isAliveAt(life, f)) continue
      sans.set(p.team, (sans.get(p.team) ?? 0) + 1)
      if (gate(life.slot, f) === null) continue
      avec.set(p.team, (avec.get(p.team) ?? 0) + 1)
      if (!seated.isSeated(p.xuid, f)) bilan.pionsSansFiche++
    }
    if ([...avec].some(([t, n]) => n > (fiches.get(t) ?? 0))) bilan.depassements++
    if ([...sans].some(([t, n]) => n > (fiches.get(t) ?? 0))) bilan.depassementsSansRegle++
  }
  return bilan
}

describe('invariant du pion — pions dessinés <= fiches occupées, à chaque image', () => {
  for (const entry of goFixtureEntries()) {
    it(`document de contrat ${entry.film} (${entry.build})`, () => {
      const bilan = mesurer(loadGoFixture(entry).doc)
      expect(bilan.pionsSansFiche).toBe(0)
      expect(bilan.depassements).toBe(0)
    })
  }
})
