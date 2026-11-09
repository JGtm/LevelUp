// @vitest-environment node
/**
 * Garde-rail CI — Roboto est auto-hébergée (apps/web/public/fonts/) : aucun appel vers Google
 * Fonts ne doit subsister dans le HTML servi (`index.html` source et `dist/index.html` construit).
 * Les fichiers de police et la licence SIL OFL 1.1 doivent exister.
 */
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, it, expect } from 'vitest'

const webRoot = fileURLToPath(new URL('../../', import.meta.url))
const GOOGLE_FONTS_HOSTS = ['fonts.googleapis', 'fonts.gstatic']

describe('polices — Roboto auto-hébergée, aucun appel Google Fonts', () => {
  for (const rel of ['index.html', 'dist/index.html']) {
    it(`${rel} ne référence pas Google Fonts (si présent)`, () => {
      const path = webRoot + rel
      if (!existsSync(path)) return
      const html = readFileSync(path, 'utf8')
      for (const host of GOOGLE_FONTS_HOSTS) {
        expect(html.includes(host), `${rel} contient ${host}`).toBe(false)
      }
    })
  }

  it('les fichiers Roboto et la licence OFL existent dans public/fonts/', () => {
    for (const rel of ['public/fonts/roboto-latin.woff2', 'public/fonts/roboto-latin-ext.woff2', 'public/fonts/OFL.txt']) {
      expect(existsSync(webRoot + rel), `${rel} absent`).toBe(true)
    }
  })
})
