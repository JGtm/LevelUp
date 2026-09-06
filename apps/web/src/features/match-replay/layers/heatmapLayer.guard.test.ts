/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : LA RAMPE DE LA CARTE DE CHALEUR VIENT DU HELPER, jamais d'une couleur écrite.
 *
 * POURQUOI CE GARDE. Une carte de chaleur est faite ENTIÈREMENT de couleur : c'est le fichier
 * du dépôt où la tentation d'écrire un dégradé « juste pour voir » est la plus forte, et une
 * couleur en dur y survivrait au thème clair, au thème sombre et aux palettes d'accessibilité
 * sans que rien ne le signale. La rampe vient donc de `heatmapRampTokens` (source unique,
 * garde-rail `heatmapColors.guard.test.ts`), résolue en hex par l'APPELANT (canvas) ou en
 * variable CSS (légende) — jamais écrite ici.
 *
 * CE QUE CE FICHIER NE FAIT PLUS (2026-09-06, lot v2 D.12, constat M6 de l'audit v7.5) : il
 * recopiait le contrôle « ni hex ni classe Tailwind » du lint canonique
 * (`tools/lint-no-hardcoded-colors.mjs`), avec sa propre expression régulière et sa propre
 * liste de deux fichiers. Ce lint est désormais joué en CI (`npm run lint:colors`, job
 * `frontend`) sur les 207 fichiers de la feature, seuil 0 — mesuré le 2026-09-06 : un hex
 * planté dans `heatmapLayer.ts` comme dans `ReplayHeatmapLegend.tsx` le fait rougir. Trois
 * expressions régulières divergentes valaient moins qu'une seule qui couvre tout.
 */
import { describe, expect, it } from 'vitest'
import { fichierNomme, lire } from '../test/featureFiles'

describe('garde-rail : la carte de chaleur ne nomme aucune couleur', () => {
  it('la rampe passe bien par le helper centralisé', () => {
    const legende = lire(fichierNomme('ReplayHeatmapLegend.tsx'))
    expect(legende).toContain('heatmapRampTokens')
  })
})
