/// <reference types="node" />
// @vitest-environment node
/**
 * livesPosition.guard.test.ts — LE GARDE-RAIL de l'unique écriture du bloc de position.
 *
 * POURQUOI IL EXISTE, ET IL A UNE DATE. Le 2026-09-01, le bloc « index des vies par joueur +
 * fenêtre après-mort + relecture » comptait QUATRE copies de hook (déflagration, drapeau,
 * crâne, couronne VIP) et deux copies pures (killFx, objectivesLayer) ; le calque du porteur
 * de bombe en aurait été la cinquième. Tout a été centralisé dans `livesPosition.ts` — et une
 * factorisation sans garde-rail re-diverge (règle n° 6 du dépôt : leçon du prédicat bot,
 * 8 → 36 copies après centralisation).
 *
 * L'ALLOWLIST EST PASSÉE DE TROIS ENTRÉES À DEUX LE 2026-09-05. `killFx.ts` y figurait au titre
 * de « copie jumelle autorisée » : il DÉFINISSAIT `posOfPlayerAt` et `KILLPOS_WINDOW_MS`, donc
 * `livesPosition.ts` l'importait, donc lui ne pouvait pas importer `livesPosition.ts` (cycle) —
 * et il réécrivait l'index chez lui. La primitive a été rapatriée dans le module canonique : le
 * cycle a disparu, `killFx.ts` importe désormais `buildLivesByXuid`/`deathWindowFrames` comme
 * tout le monde, et son exception avec. Une allowlist qui rétrécit est le seul mouvement qu'on
 * accepte sans justification datée.
 *
 * IL COUVRE AUSSI L'INDEX PAR SLOT DEPUIS LE 2026-09-06 (registre, K1). Quatre calques —
 * l'éclair de bouche, le « ! » du tireur, le câble de grappin, le dash de propulseur — le
 * réécrivaient BYTE POUR BYTE, avec la relecture « la vie qui couvre l'image » juste après :
 * quatre copies d'une règle dont l'erreur (prendre la première vie venue) place un geste
 * ailleurs sur la carte, sans qu'aucun type ne s'en aperçoive. `buildLivesBySlot` et
 * `lifeOfSlotAt` sont désormais dans la canonique, et ce garde interdit la cinquième copie.
 *
 * CE QU'IL DÉTECTE : toute nouvelle écriture de `livesByXuid`, ou de l'index par slot
 * (`new Map<number, ReplayTrackReady[]>`), hors de `livesPosition.ts` (la canonique). Un appelant qui a besoin de la relecture importe `useCarrierPosAt` /
 * `buildCarrierPosAt` (carrierPosition.ts — la porte des lecteurs de production, véhicule
 * compris) ou `buildPlayerPosAt` (le bipède seul), jamais la formule.
 *
 * CE QU'IL NE PRÉTEND PAS : une réécriture qui renommerait sa carte passerait — aucun test
 * grep ne remplace une revue. Il bloque la copie la plus probable, celle qui part du code
 * existant.
 */

import { describe, expect, it } from 'vitest'

import { cheminCourt, fichierNomme, lire, nomDe, sourcesDeLaFeature } from '../test/featureFiles'

const AUTORISES = new Set(['livesPosition.ts', 'livesPosition.guard.test.ts'])

describe('livesPosition — unique écriture du bloc de position', () => {
  it("aucune nouvelle copie de l'index livesByXuid", () => {
    const fautifs: string[] = []
    for (const f of sourcesDeLaFeature()) {
      if (AUTORISES.has(nomDe(f))) continue
      if (lire(f).includes('livesByXuid')) fautifs.push(cheminCourt(f))
    }
    expect(
      fautifs,
      `le bloc de position est RÉÉCRIT hors de livesPosition.ts : [${fautifs.join(', ')}]. ` +
        `Importer useCarrierPosAt / buildCarrierPosAt (carrierPosition.ts) ou buildPlayerPosAt ` +
        `(le bipède seul), jamais la formule.`,
    ).toEqual([])
  })

  it("aucune nouvelle copie de l'index des vies PAR SLOT", () => {
    const fautifs: string[] = []
    for (const f of sourcesDeLaFeature()) {
      if (AUTORISES.has(nomDe(f))) continue
      if (/new Map<number, ReplayTrackReady\[\]>/.test(lire(f))) fautifs.push(cheminCourt(f))
    }
    expect(
      fautifs,
      `l'index des vies par slot est RÉÉCRIT hors de livesPosition.ts : [${fautifs.join(', ')}]. ` +
        `Importer buildLivesBySlot, et lifeOfSlotAt pour la vie qui couvre l'image.`,
    ).toEqual([])
  })

  it('et la canonique porte bien les deux index — sans quoi ce garde ne garderait rien', () => {
    const src = lire(fichierNomme('livesPosition.ts'))
    expect(src).toMatch(/export function buildLivesByXuid\(/)
    expect(src).toMatch(/export function buildLivesBySlot\(/)
    expect(src).toMatch(/export function lifeOfSlotAt\(/)
  })
})
