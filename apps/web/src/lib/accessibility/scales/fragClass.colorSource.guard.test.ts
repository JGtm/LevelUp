/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n6, règle « ≤ 2 copies » → helper + garde-rail ; P7.2 du
 * PLAN_FRAG_DISTRIBUTION_V2, révisé « gamme Antagonistes ») : la couleur d'une CLASSE
 * de frags a une SOURCE UNIQUE — les helpers fragClassColor / fragRoleColor /
 * fragLeafColor (fragClass.ts), qui résolvent le mapping classe→token de la gamme
 * Antagonistes (FRAG_CLASS_TOKENS) via la palette active.
 *
 * Aucun composant/feature ne doit :
 *   (a) importer/référencer le MAPPING brut `FRAG_CLASS_TOKENS` / `fragClassToken`
 *       (ce serait un second point de vérité classe→couleur qui re-diverge — ex. la
 *       collision mêlée=grenade de l'ancien donut — et court-circuite la palette) ;
 *   (b) recopier en dur les tokens Antagonistes de frags dans un mapping local sur
 *       les clés de classe (shoulder/sidearm/heavy/melee/grenade/spartan_ability).
 *
 * Le mapping vit UNIQUEMENT dans lib/accessibility/scales/fragClass.ts (hors portée
 * scannée ici). Ce test remplace l'ancien scan « hex Okabe FIXES » (les hex de classe
 * n'existent plus : la couleur vient de la palette, plus jamais d'un littéral).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const CLASS_KEYS = ['shoulder', 'sidearm', 'heavy', 'melee', 'grenade', 'spartan_ability']

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

describe('garde-rail couleur des classes de frags (source unique fragClassColor)', () => {
  it('aucun mapping classe→couleur en dur sous features/** ou components/**', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const roots = [join(srcRoot, 'features'), join(srcRoot, 'components')]
    const offenders: string[] = []
    for (const root of roots) {
      for (const file of walk(root)) {
        const txt = readFileSync(file, 'utf8')
        // (a) import/référence du mapping brut → doit passer par fragClassColor & co.
        const usesRawMap = /FRAG_CLASS_TOKENS|fragClassToken/.test(txt)
        // (b) littéral objet mappant ≥3 clés de classe vers une valeur (couleur/token) locale.
        // SEUIL 3, PAS 2 (2026-08-16) : « melee » et « grenade » sont du vocabulaire Halo
        // ordinaire, pas exclusif aux classes de frags — un filtre de sons par catégorie
        // (match-replay/replaySound.ts, SoundCategoryFilter) les emploie tous les deux comme
        // clés booléennes, sans rapport avec une couleur. À 2, ce mapping-là déclenchait un
        // faux positif ; une VRAIE réimplémentation du mapping classe→couleur vise à couvrir
        // le référentiel, pas 2 clés au hasard — 3 garde le signal, perd le bruit.
        const classKeyLiterals = CLASS_KEYS.filter((k) => new RegExp(`['"\`]?${k}['"\`]?\\s*:`).test(txt))
        const reimplementedMap = classKeyLiterals.length >= 3
        if (usesRawMap || reimplementedMap) offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `Couleur de classe de frags à router via fragClassColor/fragRoleColor/fragLeafColor (lib/accessibility/scales/fragClass) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
