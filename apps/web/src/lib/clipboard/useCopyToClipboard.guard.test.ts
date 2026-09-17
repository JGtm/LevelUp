/**
 * useCopyToClipboard.guard.test.ts — ratchet anti-recopie du bouton « copier ».
 *
 * Le 2026-09-17 (revue adversariale des lots feat/v75, item 2.2), quatre boutons
 * portaient le même corps recopié : état `copied`, `setTimeout`,
 * `navigator.clipboard.writeText`, `catch` muet — avec deux durées différentes
 * et, pour l'un d'eux, aucun `catch` du tout. Règle CLAUDE.md n°6 : à la 3e
 * copie, helper + garde-rail. Le helper est `useCopyToClipboard.ts` ; ce test
 * est le garde-rail qui empêche la dette de re-croître.
 *
 * INTERDITS hors du module `lib/clipboard/` :
 *  - `setCopied(` — l'état de coche ne se tient plus à la main ;
 *  - `clipboard.writeText(` — l'écriture passe par `copy()` du hook.
 *
 * ALLOWLIST : les appels au presse-papier qui ne portent PAS de coche
 * transitoire (retour par toast, ou état persistant sans minuteur) sont hors du
 * périmètre du hook. Ils sont listés par fichier, avec leur raison. N'agrandir
 * cette liste qu'avec une justification datée — un nouveau bouton « copier +
 * coche » utilise le hook.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

const SRC = join(process.cwd(), 'src')
/** Le module qui a le droit de tenir la mécanique (chemin relatif à src/). */
const HOOK_DIR = 'lib/clipboard'

const FORBIDDEN: Array<{ label: string; re: RegExp }> = [
  { label: 'setCopied(', re: /setCopied\(/ },
  { label: 'clipboard.writeText(', re: /clipboard\s*\??\.?\s*\n?\s*\.?writeText\(/ },
]

/** `fichier` → raison de l'exemption (chemins relatifs à src/, séparateurs `/`). */
const ALLOWED = new Map<string, string>([
  [
    'features/feedback-drawer/FeedbackDrawer.tsx',
    'retour par toast (succès ET échec visibles), pas de coche transitoire — 2026-09-17',
  ],
  [
    'features/admin/sections/InvitesSection.tsx',
    'retour par toast sur le lien d\'invitation, pas de coche transitoire — 2026-09-17',
  ],
  [
    'features/groups/GroupsPage.tsx',
    'retour par toast sur le lien de groupe, pas de coche transitoire — 2026-09-17',
  ],
  [
    'features/admin/titles/TitleDetailCards.tsx',
    'état persistant idle/done/error du brouillon TOML, sans minuteur — 2026-09-17',
  ],
])

function walk(dir: string, out: string[]): string[] {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) walk(p, out)
    else if (/\.(ts|tsx)$/.test(name)) out.push(p)
  }
  return out
}

/** Fichiers fautifs, hors module du hook et hors allowlist : `fichier|motif`. */
function findCopies(scanHook: boolean): string[] {
  const found = new Set<string>()
  for (const file of walk(SRC, [])) {
    const rel = relative(SRC, file).replace(/\\/g, '/')
    const inHook = rel.startsWith(`${HOOK_DIR}/`)
    if (inHook !== scanHook) continue
    const body = readFileSync(file, 'utf8')
    for (const { label, re } of FORBIDDEN) {
      if (re.test(body)) found.add(`${rel}|${label}`)
    }
  }
  return [...found].sort()
}

describe('bouton « copier » — une seule mécanique (lib/clipboard)', () => {
  it('aucun état de coche ni écriture presse-papier tenus à la main hors du hook', () => {
    const offenders = findCopies(false).filter((k) => !ALLOWED.has(k.split('|')[0]))
    expect(
      offenders,
      'utiliser useCopyToClipboard() (lib/clipboard), ou justifier le fichier dans ALLOWED',
    ).toEqual([])
  })

  it('témoin positif : le balayage voit bien la mécanique DANS le hook', () => {
    // Sans ce témoin, un motif cassé (renommage, regex qui ne matche plus) rendrait
    // le test vert pour de mauvaises raisons.
    const inHook = findCopies(true)
    expect(inHook).toContain(`${HOOK_DIR}/useCopyToClipboard.ts|setCopied(`)
    expect(inHook).toContain(`${HOOK_DIR}/useCopyToClipboard.ts|clipboard.writeText(`)
  })

  it('l’allowlist ne garde aucune entrée morte', () => {
    const seen = new Set(findCopies(false).map((k) => k.split('|')[0]))
    const stale = [...ALLOWED.keys()].filter((f) => !seen.has(f))
    expect(stale, 'retirer de ALLOWED les fichiers qui n’appellent plus le presse-papier').toEqual([])
  })
})
