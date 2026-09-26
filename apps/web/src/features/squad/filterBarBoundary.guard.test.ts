/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (2026-09-20) — FRONTIÈRE ENTRE LA BARRE DE FILTRES ET LE LAYOUT
 * de la page Escouade.
 *
 * Défaut mesuré (utilisateur, 2026-09-20) : « dès que je touche à un filtre dans
 * la barre, c'est comme si la page rechargeait mais sans changer ce qui
 * s'affiche ». `SquadLayout`, PARENT de `<Outlet />` et du
 * `SquadContext.Provider`, portait l'état TRANSITOIRE de la barre (période et
 * cascade en attente, popover ouvert, requête de preview). Chaque case cochée
 * re-rendait donc tout l'arbre de la page jusqu'aux `ChartCard`, qui
 * reconstruisent alors leur option ECharts et REJOUENT leur animation d'entrée,
 * sans qu'aucune donnée n'ait bougé. Mesuré par
 * `SquadLayout.rerender.test.tsx` : 9 rendus du contenu au lieu de 2.
 *
 * Deux invariants, tous deux nécessaires (le second protège le cas où un rendu
 * du layout reste inévitable — arrivée de `resolvedContext`, refetch) :
 *
 *  1. L'état transitoire de la barre vit dans `SquadFilterBar` /
 *     `useSquadFilterBarState`, JAMAIS dans `SquadLayout`.
 *  2. La valeur de `SquadContext.Provider` est MÉMOÏSÉE — un objet littéral
 *     `value={{ … }}` est neuf à chaque rendu et re-rend tous les
 *     consommateurs, `React.memo` de l'Outlet compris (le contexte le traverse).
 *
 * Un test de rendu seul ne suffit pas : il mesure UN chemin. Ce grep interdit la
 * FORME qui a produit le défaut, où qu'elle réapparaisse dans `features/squad/`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const SQUAD_ROOT = resolve(process.cwd(), 'src', 'features', 'squad')

/**
 * Seuls modules autorisés à porter l'état transitoire de la barre. Allowlist
 * DATÉE (2026-09-20) et fermée par CONCEPTION : ce sont la barre et son hook
 * d'état, c'est-à-dire l'endroit où cet état DOIT vivre. Tout ajout ici exige
 * une nouvelle justification datée.
 */
const MODULES_DE_LA_BARRE = ['SquadFilterBar.tsx', 'useSquadFilterBarState.ts']
/** Le garde-rail lui-même : il NOMME ces motifs en prose, ce n'est pas un usage. */
const SOI_MEME = 'filterBarBoundary.guard.test.ts'
/**
 * Le test de rendu : il décrit le défaut en prose et monte SquadLayout, mais ne
 * porte aucun état transitoire.
 */
const TEST_DE_RENDU = 'SquadLayout.rerender.test.tsx'

/**
 * Motifs de l'état TRANSITOIRE de la barre. Les trois sont des formes propres à
 * cet état, pas des mots génériques : `useFiltersPreview` est la requête de
 * preview live, `setPending` le setter de la période/cascade en attente,
 * `activePopover` le popover ouvert.
 */
const MOTIFS_ETAT_TRANSITOIRE: { motif: RegExp; quoi: string }[] = [
  { motif: /useFiltersPreview\s*\(/, quoi: 'requête de preview live' },
  { motif: /setPending\b/, quoi: 'setter des filtres en attente' },
  { motif: /activePopover\b/, quoi: 'popover ouvert de la barre' },
]

function parcourir(dir: string): string[] {
  const out: string[] = []
  for (const entree of readdirSync(dir, { withFileTypes: true })) {
    const complet = join(dir, entree.name)
    if (entree.isDirectory()) {
      out.push(...parcourir(complet)) // features/squad/ a des sous-dossiers (charts/, components/, v2/)
    } else if (/\.(ts|tsx)$/.test(entree.name)) {
      out.push(complet)
    }
  }
  return out
}

const relatif = (f: string) => f.replace(SQUAD_ROOT, 'src/features/squad')

describe('garde-rail frontière barre de filtres / layout Escouade (2026-09-20)', () => {
  const fichiers = parcourir(SQUAD_ROOT)

  it.each(MOTIFS_ETAT_TRANSITOIRE)(
    'seuls SquadFilterBar et son hook portent $quoi',
    ({ motif, quoi }) => {
      const coupables = fichiers
        .filter((f) => {
          const nom = f.split(/[\\/]/).pop() ?? ''
          if (MODULES_DE_LA_BARRE.includes(nom) || nom === SOI_MEME || nom === TEST_DE_RENDU) {
            return false
          }
          return motif.test(readFileSync(f, 'utf8'))
        })
        .map(relatif)
      expect(
        coupables,
        `${quoi} : cet état est TRANSITOIRE (non commité) et doit vivre dans ` +
          `SquadFilterBar / useSquadFilterBarState, jamais dans un parent de <Outlet /> ` +
          `— sinon toucher un filtre re-rend toute la page et rejoue l'animation des ` +
          `graphes. Fichiers fautifs : ${coupables.join(', ')}`,
      ).toEqual([])
    },
  )

  it('la valeur de SquadContext.Provider est mémoïsée, jamais un objet littéral', () => {
    const coupables = fichiers
      .filter((f) => {
        const nom = f.split(/[\\/]/).pop() ?? ''
        if (nom === SOI_MEME) return false
        // `value={{` (avec ou sans espaces/sauts de ligne entre l'ouverture de la
        // balise et la prop) sur un SquadContext.Provider.
        return /SquadContext\.Provider[\s\S]{0,200}?value=\{\{/.test(readFileSync(f, 'utf8'))
      })
      .map(relatif)
    expect(
      coupables,
      `SquadContext.Provider : passer une valeur mémoïsée (useMemo). Un objet ` +
        `littéral est neuf à chaque rendu et re-rend TOUS les consommateurs — le ` +
        `contexte traverse le React.memo de <Outlet />. Fichiers fautifs : ${coupables.join(', ')}`,
    ).toEqual([])
  })
})
