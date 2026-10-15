/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : l'encre d'une écriture posée sur un aplat de segment est BLANCHE à
 * contour sombre fin, partout (2026-10-15), et se décide en UN point : les jetons `--segment-ink`,
 * `lib/accessibility/segmentInk.ts`, `SegmentLabel` (pistes HTML) et `barLabel` (ECharts).
 *
 * Contexte (2026-10-15) : une quinzaine de pistes écrivaient `text-white` ou `tc.text` sur leur
 * aplat — blanc trop pâle sur ambre ou vert, texte décalé de 2 à 3 px sous le centre de la
 * barre. Trois interdits, sur `features/` et `components/` (hors tests) :
 *   1. `text-white` / `text-black` : l'encre d'un segment ne s'écrit pas en classe ;
 *   2. `data-fit-label` : l'étiquette d'un segment est `SegmentLabel`, seul à porter le contrat ;
 *   3. `position: 'inside'` : l'étiquette ECharts d'un segment est `barLabel`.
 * Une 4e règle (15/10) : une étiquette de segment est toujours centrée sur sa barre — aucune ligne
 * de « repli » (`data-testid="...repli..."`) posée au-dessus d'une barre ; ce qui ne tient pas se
 * lit dans l'infobulle du segment.
 * Une ligne justifiée se marque `barLabel-allow: <raison>` (2) ou `color-allow: <raison>` (1) sur
 * elle ou la précédente ; toute entrée de `TEXT_ON_FIXED_BACKGROUND` porte sa raison datée.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { basename, join, resolve } from 'node:path'

/** Fichiers où `text-white` pose sur un fond FIXE qui n'est pas un segment (justification datée). */
const TEXT_ON_FIXED_BACKGROUND = new Set([
  // 2026-10-15 : titre posé sur le dégradé sombre fixe du bandeau d'une saison.
  'SeasonPassPage.tsx',
  // 2026-10-15 : titre posé sur l'image d'adversaire assombrie.
  'MatchNemesisCards.tsx',
])
const LABEL_OWNERS = new Set(['SegmentLabel.tsx', 'segmentLabelFit.ts'])
const INK_OWNERS = new Set(['barLabel.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walk(full))
    else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) out.push(full)
  }
  return out
}

function offenders(test: (line: string, prev: string) => boolean, owners: ReadonlySet<string>): string[] {
  const root = resolve(process.cwd(), 'src')
  const found: string[] = []
  for (const dir of ['features', 'components']) {
    for (const file of walk(join(root, dir))) {
      if (owners.has(basename(file))) continue
      const lines = readFileSync(file, 'utf8').split('\n')
      lines.forEach((line, i) => {
        if (test(line, lines[i - 1] ?? '')) found.push(`${file.replace(root, 'src')}:${i + 1}`)
      })
    }
  }
  return found
}

describe('garde-rail encre des segments (segmentInk source unique)', () => {
  it('aucun text-white / text-black hors fond fixe justifié', () => {
    const bad = offenders((l, p) => /\btext-(white|black)\b/.test(l) && !/color-allow:/.test(l + p), TEXT_ON_FIXED_BACKGROUND)
    expect(bad, `Encre d'un segment : SegmentLabel (HTML) ou barLabel (ECharts) — ${bad.join(', ')}`).toEqual([])
  })

  it('aucune étiquette de segment HTML hors SegmentLabel', () => {
    const bad = offenders((l) => l.includes('data-fit-label') && !l.trimStart().startsWith('*') && !l.trimStart().startsWith('//'), LABEL_OWNERS)
    expect(bad, `Étiquette de segment : utiliser SegmentLabel — ${bad.join(', ')}`).toEqual([])
  })

  it("aucune étiquette ECharts « inside » à couleur fixe hors barLabel", () => {
    const bad = offenders((l, p) => /position:\s*'inside'/.test(l) && !/barLabel-allow:/.test(l + p), INK_OWNERS)
    expect(bad, `Étiquette inside : utiliser barLabel(fill) — ${bad.join(', ')}`).toEqual([])
  })

  it('aucune ligne de repli au-dessus d’une barre (étiquette toujours centrée sur sa barre)', () => {
    const bad = offenders((l) => /data-testid={?[`'"][^`'"]*-repli/.test(l), new Set())
    expect(bad, `Valeur qui ne tient pas : infobulle du segment, jamais au-dessus — ${bad.join(', ')}`).toEqual([])
  })
})
