/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n6, règle « <= 2 copies » -> helper + garde-rail ; P7.2 du
 * PLAN_FRAG_DISTRIBUTION_V2) : la couleur d'une CLASSE de frags a une SOURCE UNIQUE —
 * fragClassColor() (fragClass.ts), qui sert les hex fixes Okabe-Ito de
 * fragClassColors.ts. Aucun composant/feature ne doit mapper une classe vers une
 * couleur EN DUR (hex littéral) : ce serait un second point de vérité qui re-diverge
 * (ex. la collision mêlée=grenade de l'ancien donut) et court-circuite la garantie
 * CVD (indépendance de la palette active).
 *
 * Ce test échoue si l'un des 6 hex de combat des classes de frags apparaît en
 * LITTÉRAL sous src/features/** ou src/components/** — la consommation doit passer
 * par fragClassColor(class) / fragRoleColor(class, i, n). Les hex vivent UNIQUEMENT
 * dans lib/accessibility/scales/fragClassColors.ts (exception documentée), hors de la
 * portée scannée ici. Le neutre #888888 (résidu) n'est pas forbidé : c'est un gris
 * générique partagé, jamais une couleur de classe empruntable.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Les 6 hex FIXES des classes de COMBAT (fragClassColors.ts, teintes Okabe-Ito). Un
// littéral de l'un d'eux dans un chart/feature = mapping direct classe→couleur interdit.
const FRAG_CLASS_COMBAT_HEX = /#(?:0072B2|E69F00|56B4E9|D55E00|009E73|F0E442)/i

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail couleur des classes de frags (fragClassColor source unique)', () => {
  it('aucun hex de classe de frags en dur sous features/** ou components/**', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const roots = [join(srcRoot, 'features'), join(srcRoot, 'components')]
    const offenders: string[] = []
    for (const root of roots) {
      for (const file of walk(root)) {
        if (FRAG_CLASS_COMBAT_HEX.test(readFileSync(file, 'utf8'))) {
          offenders.push(file.replace(srcRoot, 'src'))
        }
      }
    }
    expect(
      offenders,
      `Couleur de classe de frags à router via fragClassColor(class) (lib/accessibility/scales/fragClass) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
