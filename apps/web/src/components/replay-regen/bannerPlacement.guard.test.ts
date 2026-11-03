// @vitest-environment node
/**
 * Garde-rail — le bandeau de régénération des rejeux (`ReplayRegenBanner`) n'est monté que par la page de rejeu.
 *
 * Décision du lot L22b : le bandeau parle sur la page de rejeu seulement ; les Paramètres portent la carte
 * (`ReplayRegenCard`, même composant d'affichage) et nulle autre page ne propose la régénération. Un second site
 * d'import ferait apparaître l'offre ailleurs sans que rien ne le dise. Les fichiers `*.test.ts(x)` sont exclus : ils
 * rendent le composant pour le tester.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, dirname, relative, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
const WEB_SRC = join(HERE, '..', '..')

/** Seuls fichiers autorisés à importer le bandeau : son propre dossier et la route de la page de rejeu. */
const ALLOWED = [
  'components/replay-regen/',
  'routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx',
]

function collect(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue
      out.push(...collect(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('ReplayRegenBanner — monté par la page de rejeu seulement', () => {
  it('aucun autre fichier ne l\'importe', () => {
    const offenders = collect(WEB_SRC)
      .map((f) => relative(WEB_SRC, f).split(sep).join('/'))
      .filter((rel) => !ALLOWED.some((a) => rel === a || (a.endsWith('/') && rel.startsWith(a))))
      .filter((rel) => /import[^;]*ReplayRegenBanner/.test(readFileSync(join(WEB_SRC, rel), 'utf8')))
    expect(offenders, `import du bandeau hors de la page de rejeu : ${offenders.join(', ')}`).toEqual([])
  })

  it('la page de rejeu le monte bien', () => {
    const route = join(WEB_SRC, ALLOWED[1])
    expect(readFileSync(route, 'utf8')).toMatch(/<ReplayRegenBanner\s*\/>/)
  })
})
