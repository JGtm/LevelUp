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
        // (b) littéral objet mappant ≥2 clés de classe vers une COULEUR/TOKEN locale.
        //
        // SEUIL 2, RÉTABLI LE 2026-08-16 (il était passé à 3 le matin même). La régression
        // que ce garde-rail existe pour attraper — la collision mêlée=grenade de l'ancien
        // donut, cf. l'en-tête de ce fichier — EST un mapping à DEUX clés. Pire : mesuré le
        // 2026-08-16, `shoulder`, `sidearm`, `heavy` et `spartan_ability` n'apparaissent
        // dans AUCUN fichier de features/ ni components/ ; seules `melee` et `grenade` y
        // vivent. À 3, cette branche ne pouvait donc plus se déclencher du tout.
        //
        // LE FAUX POSITIF DU 16/08 NE VENAIT PAS DU SEUIL mais de l'absence de test sur la
        // VALEUR : « melee » et « grenade » sont du vocabulaire Halo ordinaire — filtre de
        // sons par catégorie (match-replay/replaySound.ts) et libellés i18n FR/EN — et
        // `melee: true` comme `melee: 'Mêlée'` ne sont pas des couleurs. La docstring de ce
        // test disait déjà « vers une valeur (couleur/token) » ; le prédicat, lui, ne
        // regardait que la clé. On exige désormais que la valeur SOIT une couleur : hex,
        // `var(--…)`, `resolveToken(…)`/`tokenCssVar(…)`, ou un identifiant kebab-case (la
        // forme d'un SemanticToken : chart-series-8, perf-tier-2). Même technique de
        // discrimination que perf-tier.guard.test.ts, qui sépare une échelle d'une simple
        // liste de tokens par la présence d'une comparaison numérique.
        const COLOR_VALUE = String.raw`(?:['"\`](?:#[0-9a-fA-F]{3,8}|var\(--|[a-z]+(?:-[a-z0-9]+)+)|#[0-9a-fA-F]{3,8}|var\(--|resolveToken\(|tokenCssVar\()`
        const classKeyLiterals = CLASS_KEYS.filter((k) =>
          new RegExp(`['"\`]?${k}['"\`]?\\s*:\\s*${COLOR_VALUE}`).test(txt),
        )
        const reimplementedMap = classKeyLiterals.length >= 2
        if (usesRawMap || reimplementedMap) offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(
      offenders,
      `Couleur de classe de frags à router via fragClassColor/fragRoleColor/fragLeafColor (lib/accessibility/scales/fragClass) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
