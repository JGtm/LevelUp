/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) — UN SEUL FORMATEUR D'HORLOGE DANS LE REJEU 2D.
 *
 * POURQUOI CE GARDE (registre 2026-09-05, résidu de P0-6). Deux écritures du même instant
 * cohabitaient sur la page de rejeu : `replayLogic.formatClock`, qui TRONQUE à la seconde
 * (l'horloge du lecteur, le fil, le bandeau, les infobulles, l'export), et `formatClockMMSS`
 * de `lib/formatters`, qui ARRONDIT (les marques de la frise). Le même instant s'y lisait à
 * une seconde d'écart selon l'endroit — un défaut qui ne se voit qu'en comparant deux coins
 * de l'écran, donc jamais.
 *
 * L'ARBITRAGE : le rejeu garde le formateur QUI TRONQUE. C'est la convention d'un lecteur
 * (une position de lecture n'annonce pas une seconde qui n'est pas advenue), c'est ce que
 * lisent toutes les surfaces VISIBLES du rejeu, et c'est donc le choix qui ne déplace aucun
 * pixel. `formatClockMMSS` reste le formateur d'INSTANT du reste du dépôt (la vue match,
 * `_scoreCurve`), où l'arrondi est le bon choix pour une étiquette d'axe.
 *
 * L'EXCEPTION, UNE SEULE : `ReplayMediaLightbox`, qui n'affiche pas l'horloge du match mais
 * la DURÉE d'un clip vidéo et la position dans ce clip — une autre horloge, celle du média.
 */
import { describe, expect, it } from 'vitest'
import { resolve } from 'node:path'

import { cheminCourt, featureRoot, fichierNomme, fichiersSous, lire, nomDe, tousLesFichiers } from '../test/featureFiles'

/** La signature du défaut : le formateur d'arrondi importé dans la feature du rejeu. */
const FORMATEUR_ARRONDI = /\bformatClockMMSS\b/

/** Cf. l'en-tête : la visionneuse de médias date un CLIP, pas le match. */
const AUTORISES = new Set(['ReplayMediaLightbox.tsx', 'replayClockFormat.guard.test.ts'])

describe('garde-rail : une seule horloge visible dans le rejeu', () => {
  it('aucun fichier de features/match-replay/ n’appelle le formateur d’arrondi', () => {
    const fautifs = tousLesFichiers()
      .filter((f) => !AUTORISES.has(nomDe(f)) && FORMATEUR_ARRONDI.test(lire(f)))
      .map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('et le formateur du rejeu, lui, TRONQUE — sans quoi ce garde ne garderait rien', () => {
    const src = lire(fichierNomme('replayLogic.ts'))
    expect(src).toMatch(/export function formatClock\(/)
    expect(src).toMatch(/Math\.floor\(ms \/ 1000\)/)
  })

  it('la route du rejeu non plus', () => {
    const routes = resolve(featureRoot(), '..', '..', 'routes')
    const fautifs = fichiersSous(routes).filter((f) => FORMATEUR_ARRONDI.test(lire(f)))
    expect(fautifs).toEqual([])
  })
})
