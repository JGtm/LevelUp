/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : L'ARBORESCENCE DU REJEU RESTE CELLE QUE LE README DÉCRIT.
 *
 * POURQUOI (2026-09-06, lot v2 D.11). Ce dossier a vécu à plat pendant 370 fichiers, et le
 * rangement par responsabilité ne tient que si la règle de classement est VÉRIFIÉE : une
 * arborescence documentée mais non gardée se remélange en trois lots, et le README devient une
 * doc inversée de plus.
 *
 * CE QU'IL VÉRIFIE, et c'est le critère du README (règle n° 5) : un module qui PEINT — une de
 * ses fonctions reçoit un `CanvasRenderingContext2D` — vit dans `layers/`. C'est le seul des
 * huit critères qui se lise dans le code plutôt que dans une intention, donc le seul qu'un test
 * puisse tenir. Les sept autres restent affaire de revue.
 *
 * CE QU'IL NE PRÉTEND PAS : il ne dit rien d'un fichier de `model/` qui aurait dû aller dans
 * `hooks/`. Il bloque la dérive la plus probable et la plus coûteuse — un calque neuf écrit à
 * la racine ou glissé dans `model/`, où personne ne le cherchera.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, nomDe, sourcesDeLaFeature } from './featureFiles'

/** La signature d'un module qui peint. */
const PEINT = /CanvasRenderingContext2D/

/**
 * Les trois qui peignent SANS être des calques, et chacun pour une raison nommée :
 *  - `ReplayCanvas.tsx` MONTE la toile et compose la scène : il n'est aucun calque ;
 *  - `overlayPaint.ts` repeint les panneaux React dans la toile EXPORTÉE (domaine `export/`,
 *    qui l'emporte sur le critère du trait, cf. l'ordre du README) ;
 *  - `recordingContext.ts` est le DOUBLE de test : il imite le contexte, il ne peint rien.
 */
const HORS_CALQUES = new Set(['ReplayCanvas.tsx', 'overlayPaint.ts', 'recordingContext.ts'])

describe('garde-rail : le classement des fichiers du rejeu', () => {
  it('tout module qui peint vit dans layers/', () => {
    const egares = sourcesDeLaFeature()
      .filter((f) => !HORS_CALQUES.has(nomDe(f)))
      .filter((f) => PEINT.test(lire(f)))
      .map(cheminCourt)
      .filter((c) => !c.startsWith('layers/'))
    expect(
      egares,
      `ces modules peignent et vivent hors de layers/ : [${egares.join(', ')}]. ` +
        `Voir features/match-replay/README.md — un module qui reçoit un contexte de dessin est un calque.`,
    ).toEqual([])
  })

  it('et layers/ en contient bien — sans quoi ce garde ne garderait rien', () => {
    const peintres = sourcesDeLaFeature()
      .map(cheminCourt)
      .filter((c) => c.startsWith('layers/'))
    // 62 modules au 2026-09-06. Le nombre n'est pas figé : c'est la présence qui l'est.
    expect(peintres.length).toBeGreaterThanOrEqual(40)
  })

  it('les huit dossiers de responsabilité existent tous', () => {
    const dossiers = new Set(
      sourcesDeLaFeature()
        .map(cheminCourt)
        .filter((c) => c.includes('/'))
        .map((c) => c.split('/')[0]),
    )
    for (const attendu of ['layers', 'ui', 'model', 'hooks', 'sound', 'export', 'settings', 'i18n']) {
      expect(dossiers.has(attendu), `le dossier ${attendu}/ a disparu du rejeu`).toBe(true)
    }
  })
})
