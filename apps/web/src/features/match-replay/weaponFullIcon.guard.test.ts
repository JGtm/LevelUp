/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (même patron que `assistMarkIcon.guard.test.ts`) : l'échange de stem
 * `contour-XX` -> `silhouette-XX` de `weaponFullIcon.ts` repose sur un ALIGNEMENT
 * d'atlas — même index = même arme, mêmes tags `weap`. Ce test rejoue cette hypothèse
 * contre `static/weapons-assets/halo_infinite/jeu/index.json` ET contre le disque :
 * un atlas régénéré désaligné casserait CE test, pas l'écran (une fiche montrerait
 * l'icône d'une AUTRE arme, le pire des replis).
 */
import { describe, expect, it } from 'vitest'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { weaponFullIcon } from './weaponFullIcon'

const REPO_ROOT = resolve(__dirname, '..', '..', '..', '..', '..')
const JEU_DIR = resolve(REPO_ROOT, 'static', 'weapons-assets', 'halo_infinite', 'jeu')

interface SpriteEntry {
  style: string
  file: string
  tags_weap?: string[]
}

const index = JSON.parse(readFileSync(resolve(JEU_DIR, 'index.json'), 'utf8')) as SpriteEntry[]

/** L'index numérique d'un fichier d'atlas (`contour-07.png` -> 7). */
const numOf = (file: string) => Number.parseInt(file.replace(/\D/g, ''), 10)

describe('garde-rail : atlas contour et silhouette alignés (même index = même arme)', () => {
  const contours = index.filter((e) => e.style === 'contour')
  const silhouettes = new Map(index.filter((e) => e.style === 'silhouette').map((e) => [numOf(e.file), e]))

  it('chaque contour a sa silhouette de même index, aux MÊMES tags weap', () => {
    expect(contours.length).toBeGreaterThan(0)
    for (const c of contours) {
      const s = silhouettes.get(numOf(c.file))
      expect(s, `silhouette manquante pour ${c.file}`).toBeTruthy()
      expect(s?.tags_weap ?? [], `tags désalignés sur l'index ${numOf(c.file)}`).toEqual(
        c.tags_weap ?? [],
      )
    }
  })

  it('chaque silhouette référencée existe sur disque', () => {
    for (const c of contours) {
      const stem = c.file.replace('contour-', 'silhouette-')
      expect(existsSync(resolve(JEU_DIR, stem)), `${stem} absent du disque`).toBe(true)
    }
  })
})

describe('weaponFullIcon — quel échange, et pour qui', () => {
  it('échange le contour contre la silhouette, à retourner (le sens du kill feed)', () => {
    expect(weaponFullIcon('/static/weapons-assets/halo_infinite/jeu/contour-07.png')).toEqual({
      url: '/static/weapons-assets/halo_infinite/jeu/silhouette-07.png',
      mirrored: true,
    })
  })

  it('une vignette hors atlas garde son URL et son sens — jamais un dessin fini retourné', () => {
    const fini = '/static/weapons-assets/halo_infinite/oddball_skull.png'
    expect(weaponFullIcon(fini)).toEqual({ url: fini, mirrored: false })
  })
})
