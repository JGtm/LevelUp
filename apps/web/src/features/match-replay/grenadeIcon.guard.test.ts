/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, même patron que `replaySoundAssets.guard.test.ts`) : la liste
 * de stems de `grenadeIcon.ts` et le dossier `static/grenades-assets/halo_infinite/` sont la
 * MÊME liste, rejouée ici — index compris.
 *
 * POURQUOI. Le client ne sonde jamais le serveur pour savoir si une image existe : il croit
 * la liste. Un stem sans fichier donnerait un cadre vide en production ; un fichier sans
 * stem, une image morte que la fiche n'affiche jamais parce qu'elle retombe en silence sur
 * le masque de HUD de l'artefact. Les deux dérives cassent CE test, pas l'écran.
 *
 * LES DEUX ENCRES SONT VÉRIFIÉES SÉPARÉMENT : livrer `frag_light` sans `frag_dark` ne se
 * verrait que dans un seul thème — c'est-à-dire, en pratique, chez l'autre moitié des
 * lecteurs.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { GRENADE_ICON_STEMS, grenadeIconInk, grenadeIconOf } from './grenadeIcon'

const REPO_ROOT = resolve(__dirname, '..', '..', '..', '..', '..')
const ASSETS_DIR = resolve(REPO_ROOT, 'static', 'grenades-assets', 'halo_infinite')

const shipped = new Set(
  readdirSync(ASSETS_DIR)
    .filter((f) => f.endsWith('.png'))
    .map((f) => f.slice(0, -'.png'.length)),
)

describe('garde-rail : stems de vignettes de grenade = dossier d assets', () => {
  it('chaque stem a ses DEUX encres livrées', () => {
    const missing = GRENADE_ICON_STEMS.flatMap((s) =>
      (['light', 'dark'] as const).map((ink) => `${s}_${ink}`),
    ).filter((f) => !shipped.has(f))
    expect(missing).toEqual([])
  })

  it('chaque image livrée est servie par un stem (0 asset mort)', () => {
    const orphans = [...shipped].filter((f) => {
      const stem = f.replace(/_(light|dark)$/, '')
      return !(GRENADE_ICON_STEMS as readonly string[]).includes(stem)
    })
    expect(orphans).toEqual([])
  })

  it('l index d assets déclare exactement ces types', () => {
    const index = JSON.parse(readFileSync(resolve(ASSETS_DIR, 'index.json'), 'utf8')) as {
      grenades: Record<string, unknown>
    }
    expect(Object.keys(index.grenades).sort()).toEqual([...GRENADE_ICON_STEMS].sort())
  })
})

describe('grenadeIconOf — quelle vignette, et dans quel ordre', () => {
  it("l'encre suit le FOND : thème clair -> dessin sombre, et l'inverse", () => {
    expect(grenadeIconInk('light')).toBe('dark')
    expect(grenadeIconInk('dark')).toBe('light')
    // Un thème inconnu ne doit pas rendre l'écran illisible : il retombe sur le défaut sombre.
    expect(grenadeIconInk('')).toBe('light')
  })

  it("l'image versionnée prime la vignette cuite dans l'artefact", () => {
    const ref = grenadeIconOf(
      { en: 'Frag', fr: 'Fragmentation', img: '/static/weapons-assets/halo_infinite/hud/Frag.png', tinted: true },
      'halo_infinite',
      'dark',
    )
    expect(ref).toEqual({
      url: '/static/grenades-assets/halo_infinite/frag_light.png',
      tinted: false,
    })
  })

  it('un type hors catalogue garde la vignette du document, jamais celle d un voisin', () => {
    const ref = grenadeIconOf(
      { en: 'Unobtainium', img: '/static/weapons-assets/halo_infinite/hud/Frag.png', tinted: true },
      'halo_infinite',
      'dark',
    )
    expect(ref).toEqual({
      url: '/static/weapons-assets/halo_infinite/hud/Frag.png',
      tinted: true,
    })
  })

  it('ni image versionnée ni vignette : rien — l appelant gardera le libellé', () => {
    expect(grenadeIconOf({ en: 'Unobtainium' }, 'halo_infinite', 'dark')).toBeNull()
    expect(grenadeIconOf(undefined, 'halo_infinite', 'dark')).toBeNull()
  })
})
