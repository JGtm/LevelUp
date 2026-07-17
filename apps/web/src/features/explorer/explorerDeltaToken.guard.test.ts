/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail anti-divergence `deltaToken` (CLAUDE.md n°6 « ≤ 2 copies d'un même
 * pattern : à la 3e, centraliser dans un helper ET ajouter un garde-rail »).
 *
 * Le mapping « signe d'un delta → token de couleur d'issue » (outcome-win /
 * outcome-loss / outcome-draw) a atteint 3 usages avec la tuile Classement (V3) :
 * il est centralisé dans `ExplorerBriefing.logic.ts` (export `deltaToken`) et
 * consommé par le Strip, Modules et Tiles via import. Ce garde-rail interdit de
 * RÉINLINER le ternaire dans un composant du briefing — une factorisation sans
 * garde-rail re-diverge (leçon CLAUDE.md).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Signature distinctive du corps de deltaToken (assez spécifique pour éviter les
// faux positifs — `accent="outcome-draw"` seul ne matche pas).
const INLINE_DELTA_TOKEN = /\?\s*'outcome-win'\s*:\s*s\s*<\s*0\s*\?\s*'outcome-loss'/

/** Composants du briefing Explorer (hors tests, hors le helper canonique logic.ts). */
function briefingComponents(): string[] {
  const featDir = resolve(process.cwd(), 'src', 'features', 'explorer')
  const out: string[] = []
  for (const entry of readdirSync(featDir, { withFileTypes: true })) {
    if (!entry.isFile()) continue
    if (!/\.tsx?$/.test(entry.name) || /\.test\.tsx?$/.test(entry.name)) continue
    if (entry.name === 'ExplorerBriefing.logic.ts') continue // helper canonique
    out.push(join(featDir, entry.name))
  }
  return out
}

describe('garde-rail deltaToken (anti-divergence, CLAUDE.md §6)', () => {
  it('aucun composant du briefing ne ré-inline le ternaire deltaToken', () => {
    const offenders = briefingComponents().filter((f) =>
      INLINE_DELTA_TOKEN.test(readFileSync(f, 'utf8')),
    )
    expect(
      offenders,
      `Ré-inline deltaToken interdit (importer depuis ExplorerBriefing.logic) : ${offenders
        .map((f) => f.split(/[\\/]/).pop())
        .join(', ')}`,
    ).toEqual([])
  })
})
