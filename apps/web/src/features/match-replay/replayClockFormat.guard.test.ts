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
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

/** La signature du défaut : le formateur d'arrondi importé dans la feature du rejeu. */
const FORMATEUR_ARRONDI = /\bformatClockMMSS\b/

/** Cf. l'en-tête : la visionneuse de médias date un CLIP, pas le match. */
const AUTORISES = new Set(['ReplayMediaLightbox.tsx', 'replayClockFormat.guard.test.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail : une seule horloge visible dans le rejeu', () => {
  it('aucun fichier de features/match-replay/ n’appelle le formateur d’arrondi', () => {
    const fautifs = walk(__dirname).filter((f) => {
      const base = f.split(/[\\/]/).pop() ?? ''
      if (AUTORISES.has(base)) return false
      return FORMATEUR_ARRONDI.test(readFileSync(f, 'utf8'))
    })
    expect(fautifs).toEqual([])
  })

  it('et le formateur du rejeu, lui, TRONQUE — sans quoi ce garde ne garderait rien', () => {
    const src = readFileSync(join(__dirname, 'replayLogic.ts'), 'utf8')
    expect(src).toMatch(/export function formatClock\(/)
    expect(src).toMatch(/Math\.floor\(ms \/ 1000\)/)
  })

  it('la route du rejeu non plus', () => {
    const routes = resolve(__dirname, '..', '..', 'routes')
    const fautifs = walk(routes).filter((f) => FORMATEUR_ARRONDI.test(readFileSync(f, 'utf8')))
    expect(fautifs).toEqual([])
  })
})
