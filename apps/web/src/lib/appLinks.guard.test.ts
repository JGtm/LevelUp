// @vitest-environment node
/**
 * Garde-rail (règle n°6 CLAUDE.md) — les identités externes du projet (dépôt
 * GitHub, mécénat) n'existent qu'à UN endroit : `lib/appLinks.ts`.
 *
 * Contexte : le slug `JGtm/LevelUp` était écrit en dur dans
 * `features/feedback-drawer/buildIssueUrl.ts`. Le pied de page (2026-08-31) en
 * aurait été la 2e copie, et les liens de don une 3e famille de littéraux
 * dispersés. On centralise dans `appLinks.ts` et ce test interdit la
 * réapparition des littéraux ailleurs — sans quoi la factorisation re-diverge
 * (renommage du dépôt, changement d'identifiant PayPal appliqué à moitié).
 *
 * Ce que le test NE couvre PAS :
 *   - les URL GitHub d'autres dépôts (documentation, dépendances) — seuls les
 *     littéraux propres au projet sont interdits ;
 *   - les fichiers `*.test.ts(x)`, où épingler l'URL RÉELLE attendue est le
 *     but même de l'assertion (même convention que
 *     `lib/i18n/no-field-label-dictionary.test.ts`). Les faire pointer vers la
 *     constante rendrait ces tests tautologiques.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, dirname, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
const WEB_SRC = join(HERE, '..')

/** Fichier canonique — seul autorisé à porter ces littéraux. */
const CANONICAL = 'lib/appLinks.ts'

/** Littéraux interdits hors du fichier canonique, avec le remplaçant attendu. */
const FORBIDDEN: ReadonlyArray<{ literal: string; useInstead: string }> = [
  { literal: 'JGtm/LevelUp', useInstead: 'GITHUB_REPO / GITHUB_URL' },
  { literal: 'github.com/sponsors/', useInstead: 'SPONSORS_URL' },
  { literal: 'paypal.me/', useInstead: 'PAYPAL_URL' },
  { literal: 'csinsight.eu', useInstead: 'csinsightUrl(locale)' },
  { literal: 'lvelup.info', useInstead: 'privacyContactEmail()' },
]

function collectSourceFiles(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue
      out.push(...collectSourceFiles(full))
      continue
    }
    if (!/\.(ts|tsx)$/.test(entry.name)) continue
    // Voir « Ce que le test NE couvre PAS » en tête de fichier.
    if (/\.test\.(ts|tsx)$/.test(entry.name)) continue
    out.push(full)
  }
  return out
}

function toPosix(p: string): string {
  return p.split('\\').join('/')
}

describe('appLinks — source unique des identités externes', () => {
  it('aucun littéral de dépôt / de don hors de lib/appLinks.ts', () => {
    const offenders: string[] = []
    for (const file of collectSourceFiles(WEB_SRC)) {
      const rel = toPosix(relative(WEB_SRC, file))
      if (rel === CANONICAL) continue
      const src = readFileSync(file, 'utf8')
      for (const { literal, useInstead } of FORBIDDEN) {
        if (src.includes(literal)) {
          offenders.push(`${rel} contient "${literal}" — importer ${useInstead} depuis @/lib/appLinks`)
        }
      }
    }
    expect(offenders).toEqual([])
  })

  it('le fichier canonique porte bien les identités attendues', async () => {
    const links = await import('./appLinks')
    expect(links.GITHUB_URL).toContain(links.GITHUB_REPO)
    expect(links.GITHUB_ISSUES_URL.startsWith(links.GITHUB_URL)).toBe(true)
    expect(links.SPONSORS_URL).toMatch(/^https:\/\/github\.com\/sponsors\/\S+$/)
    expect(links.PAYPAL_URL).toMatch(/^https:\/\/paypal\.me\/\S+$/)
    expect(links.GITHUB_PROFILE_URL).toMatch(/^https:\/\/github\.com\/\S+$/)
  })

  it('csinsightUrl suit la locale de l’app (le site sert /fr et /en)', async () => {
    const { csinsightUrl } = await import('./appLinks')
    expect(csinsightUrl('fr')).toBe('https://csinsight.eu/fr')
    expect(csinsightUrl('en')).toBe('https://csinsight.eu/en')
  })

  it('l’adresse de contact est une adresse de rôle, pas une boîte personnelle', async () => {
    const { privacyContactEmail } = await import('./appLinks')
    const email = privacyContactEmail()
    expect(email).toMatch(/^[a-z0-9._-]+@[a-z0-9.-]+\.[a-z]{2,}$/)
    // Adresse de rôle : si la partie locale redevient un nom de personne, la
    // propriété « remplaçable sans rien perdre » tombe.
    expect(email.split('@')[0]).toMatch(/^(contact|privacy|confidentialite|legal|dpo)\d*$/)
  })
})
