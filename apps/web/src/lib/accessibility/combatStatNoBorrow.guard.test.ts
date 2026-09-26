/**
 * combatStatNoBorrow.guard.test.ts — ratchet anti-emprunt pour les stats de combat.
 *
 * Depuis le 2026-09-17 (`.ai/V7.5/chantiers/PLAN_COULEURS_STATS_COMBAT_2026-09-17.md`), une couleur qui
 * dit « frag / mort / assistance / sens d'assistance » passe par la famille dédiée
 * (`stat-kills`, `stat-deaths`, `stat-assists`, `assist-received`, `assist-given`).
 * Avant, chaque page empruntait un rôle voisin et les assistances avaient cinq teintes.
 *
 * DÉTECTION (volontairement simple, ligne par ligne) : une ligne de code de
 * `features/` ou `components/` qui nomme une stat (kill, frag, death, assist) ET porte en
 * littéral un jeton d'un AUTRE rôle. Les familles qui encodent autre chose qu'une stat
 * restent permises sans justification : équipe (`team-*`), joueur d'escouade
 * (`squad-player-*`), classe d'arme (`frag-*`).
 *
 * Les usages CONSERVÉS (la couleur dit l'équipe, le joueur, l'issue, la qualité ou le
 * récit, pas la stat) sont listés ci-dessous par fichier + jeton, avec leur raison. Une
 * nouvelle ligne fautive fait échouer le test : utiliser la famille, ou justifier ici.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { ALL_TOKENS } from './semantic-tokens'

const SRC = join(process.cwd(), 'src')
const SCANNED_DIRS = ['features', 'components']

const PERMITTED_FAMILIES = /^(stat-|assist-received$|assist-given$|team-|squad-player-|frag-)/
const BORROWED = ALL_TOKENS.filter((t) => !PERMITTED_FAMILIES.test(t))
const BORROWED_RE = new RegExp(`['"\`](${BORROWED.join('|')})['"\`]`, 'g')
// « fragment » n'est pas un frag.
const STAT_RE = /kill|frag(?!ment)|death|assist/i

/** `fichier|jeton` → raison. Fichier relatif à src/, séparateurs `/`. */
const KEPT = new Map<string, string>([
  // SOUS-TYPES de frag : la couleur distingue un type de frag d'un autre, pas « frag ».
  ['components/ui/match-card.tsx|perf-tier-3', 'frags parfaits de la tuile (accent de sous-type)'],
  ['features/explorer/ExplorerTargetSampleStats.tsx|outcome-win', 'tuile frags parfaits (accent de sous-type)'],
  ['features/synthesis/SynthesisPage.tsx|perf-tier-3', 'carte frags parfaits (accent de sous-type)'],
  ['features/synthesis/SynthesisPage.tsx|perf-tier-2', 'carte tirs à la tête (accent de sous-type)'],
  ['features/synthesis/SynthesisPage.tsx|chart-series-2', 'carte assassinats (accent de sous-type)'],
  ['features/synthesis/SynthesisPage.tsx|chart-series-3', 'carte frappes au sol (accent de sous-type)'],
  ['features/synthesis/SynthesisPage.tsx|chart-series-4', "carte coups d'épaule (accent de sous-type)"],
])

function walk(dir: string, out: string[]): string[] {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) walk(p, out)
    else if (/\.(ts|tsx)$/.test(name) && !/\.test\.|\.stories\./.test(name)) out.push(p)
  }
  return out
}

function findBorrows(): string[] {
  const found = new Set<string>()
  for (const d of SCANNED_DIRS) {
    for (const file of walk(join(SRC, d), [])) {
      const rel = relative(SRC, file).replace(/\\/g, '/')
      readFileSync(file, 'utf8')
        .split('\n')
        .forEach((line) => {
          const t = line.trim()
          if (t.startsWith('//') || t.startsWith('*') || t.startsWith('/*') || t.startsWith('{/*')) return
          if (!STAT_RE.test(line)) return
          for (const m of line.matchAll(BORROWED_RE)) found.add(`${rel}|${m[1]}`)
        })
    }
  }
  return [...found].sort()
}

describe('stats de combat — aucun jeton emprunté', () => {
  const found = findBorrows()

  it("aucune nouvelle couleur empruntée pour un frag, une mort ou une assistance", () => {
    const unexplained = found.filter((k) => !KEPT.has(k))
    expect(
      unexplained,
      'utiliser stat-kills / stat-deaths / stat-assists / assist-received / assist-given, ou justifier dans KEPT',
    ).toEqual([])
  })

  it('la liste des usages conservés ne garde aucune entrée morte', () => {
    const stale = [...KEPT.keys()].filter((k) => !found.includes(k))
    expect(stale, 'retirer de KEPT les entrées qui ne correspondent plus au code').toEqual([])
  })
})
