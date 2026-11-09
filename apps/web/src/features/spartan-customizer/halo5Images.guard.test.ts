/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — les images Halo 5 du front sont des WebP (L23.3-e, `scripts/convert-halo5-images.mjs`) :
 * les masques du personnalisateur Spartan (emblèmes, plaques, vignettes) et le fond. Un PNG de
 * ces familles qui revient dans `public/titles/halo_5/` repèse l'installeur de bureau (41 Mo avant la
 * conversion), et une URL en `.png` vers elles serait un 404. `teams/` reste en PNG : son chemin
 * `/titles/{slug}/teams/{id}.png` est partagé avec Halo Infinite.
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { describe, expect, it } from 'vitest'

const racine = join(process.cwd(), 'public', 'titles', 'halo_5')
const sources = [
  'src/features/spartan-customizer/SpartanCustomizerModal.tsx',
  'src/features/explorer/ExplorerTargetIdentityBanner.tsx',
  'src/features/home/HomeSpartanIdentityBanner.tsx',
  'src/features/home/HomeHeroBanner.tsx',
]

function pngsSous(dossier: string): string[] {
  return readdirSync(dossier, { withFileTypes: true }).flatMap((e) =>
    e.isDirectory() ? pngsSous(join(dossier, e.name)) : e.name.endsWith('.png') ? [join(dossier, e.name)] : [],
  )
}

describe('images Halo 5 du front : WebP', () => {
  it('aucun PNG hors de teams/ sous public/titles/halo_5', () => {
    const restants = pngsSous(racine)
      .map((p) => relative(racine, p).split(sep).join('/'))
      .filter((p) => !p.startsWith('teams/'))
    expect(restants).toEqual([])
  })

  it('le code ne référence plus de PNG pour ces familles', () => {
    for (const s of sources) {
      const code = readFileSync(join(process.cwd(), s), 'utf8')
      expect(code, s).not.toMatch(/(spartan\/(emblems|nameplates)|\/emblems|\/nameplates|rawThumbDir|wallpaper_halo_5)[^'`"]*\.png/)
    }
  })

  it('les fichiers référencés existent', () => {
    expect(existsSync(join(racine, 'wallpaper_halo_5_guardians_01.webp'))).toBe(true)
    for (const dir of ['emblems', 'nameplates', 'emblems_thumb_raw', 'nameplates_thumb_raw']) {
      expect(existsSync(join(racine, 'spartan', dir, '001.webp')), dir).toBe(true)
    }
  })
})
