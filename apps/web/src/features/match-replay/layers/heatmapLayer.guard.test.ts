/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail COULEURS de la carte de chaleur (CLAUDE.md n° 12, skill color-tokens) : ni
 * valeur hexadécimale, ni classe Tailwind de palette dans les deux fichiers du calque.
 *
 * POURQUOI CELUI-LÀ ET PAS UNE RELECTURE. Une carte de chaleur est faite ENTIÈREMENT de
 * couleur : c'est le fichier du dépôt où la tentation d'écrire un dégradé « juste pour
 * voir » est la plus forte, et une couleur en dur y survivrait au thème clair, au thème
 * sombre et aux palettes d'accessibilité sans que rien ne le signale. La rampe vient donc
 * de `heatmapRampTokens` (source unique, garde-rail heatmapColors.guard.test.ts), résolue
 * en hex par l'APPELANT (canvas) ou en variable CSS (légende) — jamais écrite ici.
 */
import { describe, expect, it } from 'vitest'
import { fichierNomme, lire } from '../test/featureFiles'

/** Une valeur hexadécimale de couleur écrite en dur. */
const HEX = /#[0-9a-fA-F]{6}\b/

/** Une classe Tailwind de PALETTE (les classes sémantiques — bg-card, border-border… — sont
 *  au contraire la bonne façon de faire, elles ne sont pas visées). */
const TAILWIND_PALETTE =
  /\b(?:text|bg|border|from|via|to|fill|stroke|ring|decoration|outline|accent|caret|divide|placeholder)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-\d{2,3}\b/

const FICHIERS = ['heatmapLayer.ts', 'ReplayHeatmapLegend.tsx']

describe('garde-rail : la carte de chaleur ne nomme aucune couleur', () => {
  it.each(FICHIERS)('%s ne contient aucun hex ni classe Tailwind de palette', (nom) => {
    const source = lire(fichierNomme(nom))
    expect(HEX.test(source), `hex en dur dans ${nom}`).toBe(false)
    expect(TAILWIND_PALETTE.test(source), `classe de palette dans ${nom}`).toBe(false)
  })

  it('et la rampe passe bien par le helper centralisé — sans quoi ce garde ne garderait rien', () => {
    const legende = lire(fichierNomme('ReplayHeatmapLegend.tsx'))
    expect(legende).toContain('heatmapRampTokens')
  })
})
