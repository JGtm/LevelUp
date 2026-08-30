/**
 * GARDE-RAIL — UNE SEULE ATTACHE hls.js DANS LE DÉPÔT.
 *
 * L'attache d'un flux HLS à un `<video>` a vécu en un seul endroit (la galerie) jusqu'à ce que
 * la lightbox du rejeu 2D en ait besoin (2026-08-28). Elle a été extraite dans
 * `lib/media/useHlsVideo.ts` plutôt que recopiée : deux copies auraient divergé sur le premier
 * quirk de navigateur corrigé d'un seul côté — et il y en a déjà un au dossier (Chrome répond
 * « maybe » à `canPlayType` pour du HLS sans savoir exposer les pistes audio, incident
 * 2026-06-14).
 *
 * SANS CE TEST, LA FACTORISATION SE DÉFERAIT. Un `import Hls from 'hls.js'` dans un troisième
 * composant compile, passe tous les autres tests, et personne ne voit la copie arriver — c'est
 * exactement la dette que la règle des deux copies du dépôt cherche à empêcher (leçon du
 * prédicat bot, passé de 8 à 36 copies après sa centralisation).
 *
 * CE QUE CE TEST N'INTERDIT PAS : utiliser le hook. Il n'interdit que d'importer la
 * bibliothèque en direct ailleurs que dans le helper.
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

/** Le SEUL fichier autorisé à importer hls.js, en chemin relatif à `src/`. */
const HELPER = 'lib/media/useHlsVideo.ts'

/** Les tests mockent la bibliothèque (`vi.mock('hls.js')`) : ce n'est pas une seconde attache. */
const IS_TEST = /\.(test|guard\.test)\.(ts|tsx)$/

function sourceFiles(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) {
      sourceFiles(full, out)
      continue
    }
    if (/\.(ts|tsx)$/.test(entry) && !IS_TEST.test(entry)) out.push(full)
  }
  return out
}

describe('hls.js — une seule attache dans le dépôt', () => {
  it(`n'est importé que par ${HELPER}`, () => {
    const srcRoot = resolve(__dirname, '../..')
    const coupables = sourceFiles(srcRoot)
      .filter((file) => /from ['"]hls\.js['"]/.test(readFileSync(file, 'utf8')))
      .map((file) => relative(srcRoot, file).replace(/\\/g, '/'))

    expect(coupables).toEqual([HELPER])
  })
})
