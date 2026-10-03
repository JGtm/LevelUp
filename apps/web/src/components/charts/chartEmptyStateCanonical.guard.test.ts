/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (2026-09-22) : DANS `components/charts/`, UN ÉTAT VIDE SE DESSINE PAR
 * `EmptyStateNotice`, et AUCUNE string UI n'y est écrite en français en dur.
 *
 * Pourquoi : `ChartCardEmpty` rendait une ligne grise centrée, sans titre ni cadre, et son
 * message par défaut était le littéral FR `'Aucune donnée à afficher'` — donc une string UI
 * sans parité EN, au cœur d'un composant monté par toutes les pages. Sur une rangée où la
 * carte voisine portait le gabarit canonique (`components/ui/empty-state.tsx` : titre
 * `text-sm font-semibold text-foreground` + phrase `mt-1 text-sm text-muted-foreground` dans
 * un cadre `rounded-xl border-dashed border-border bg-muted/80`), la MÊME absence se lisait
 * de deux façons. Décision utilisateur du 2026-09-22 : aligner sur le canon.
 *
 * Jumeau de `features/_shared/usage/usageEmptyStateCanonical.guard.test.ts`, appliqué au
 * dossier des wrappers ECharts.
 *
 * Preuve de mordant (trois mutations, 2026-09-22), chacune sur UNE assertion distincte :
 *   - l'ancien `<div className="flex items-center justify-center text-sm
 *     text-muted-foreground" data-testid="chart-card-empty">{message}</div>` remis dans
 *     `ChartCard.tsx` fait rougir « aucun cadre ni ligne maison » ;
 *   - le même bloc SANS l'import d'`EmptyStateNotice` fait rougir « passe par
 *     EmptyStateNotice » ;
 *   - le défaut `emptyMessage = 'Aucune donnée à afficher'` remis en place fait rougir
 *     « aucune string FR en dur ».
 * L'état courant repasse les trois au vert. Mutations non committées — seul le garde-rail
 * l'est.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const CHARTS_ROOT = resolve(process.cwd(), 'src', 'components', 'charts')

/** Les sources du dossier, hors tests (un test PEUT citer un littéral FR : il l'assert). */
function chartSources(): string[] {
  return readdirSync(CHARTS_ROOT, { withFileTypes: true })
    .filter((e) => e.isFile() && /\.tsx?$/.test(e.name) && !/\.test\.tsx?$/.test(e.name))
    .map((e) => join(CHARTS_ROOT, e.name))
}

const SOURCES = chartSources()

/**
 * LES MARQUEURS D'UN ÉTAT VIDE DESSINÉ SUR PLACE — `EmptyStateNotice` est le seul autorisé
 * à les porter :
 *   - un cadre pointillé posé en bloc (`border-dashed` + un `rounded-*`) ;
 *   - une balise qui porte À LA FOIS sa typographie grise et un `data-testid` d'état vide.
 *
 * Le `data-testid` doit SE TERMINER par `empty` : `heatmap-empty-cell-legend` est une
 * LÉGENDE qui nomme la forme des cases sans mesure, pas un état vide de carte.
 */
const HOME_MADE_FRAME = /className="[^"]*\brounded-(?:lg|xl|md)\b[^"]*\bborder-dashed\b[^"]*"/
const HOME_MADE_LINE =
  /className="[^"]*\btext-muted-foreground\b[^"]*"[^>]*\bdata-testid="[a-z-]*empty"/

/** Un fichier compte comme « rendant un état vide » s'il en nomme un. */
const RENDERS_EMPTY = /data-testid="[a-z-]*empty"|EmptyStateNotice/

/** L'import du gabarit canonique. */
const IMPORTS_CANONICAL =
  /import\s*\{[^}]*\bEmptyStateNotice\b[^}]*\}\s*from\s*'@\/components\/ui\/empty-state'/

/**
 * STRINGS FR EN DUR. Un littéral (guillemets simples/doubles, ou texte JSX entre balises)
 * qui porte un mot français accentué ou un mot-outil français non ambigu.
 *
 * Retirer d'abord les commentaires (`/* ... *\/` et `//`) et les libellés de `data-*` /
 * `className` : un commentaire FR est la règle du dépôt, pas une string UI.
 *
 * Exemption unique et JUSTIFIÉE : un dictionnaire `Record<Locale, string>` porte par
 * construction le FR ET l'EN (parité garantie par le typage, CLAUDE.md n°1) — le fichier
 * qui en déclare un est hors de ce scan. Aujourd'hui : `Heatmap2DChart.tsx`
 * (`HEATMAP_EMPTY_CELL_TEXT`).
 */
const LOCALE_DICT = /Record<Locale,\s*(?:string|[A-Za-z]+)>/
const FR_WORD =
  /(?:[àâäéèêëîïôöùûüç]|\b(?:aucun|aucune|données|donnée|afficher|erreur|chargement|matchs?|joueur|équipe|indisponible|sélection|période|affichée?)\b)/i

function stripComments(src: string): string {
  return src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '')
}

/** Littéraux de chaîne susceptibles d'être affichés (on ignore imports et chemins). */
function uiLiterals(src: string): string[] {
  const out: string[] = []
  const re = /'([^'\\\n]{3,})'|"([^"\\\n]{3,})"/g
  let m: RegExpExecArray | null
  while ((m = re.exec(src)) !== null) {
    const value = m[1] ?? m[2] ?? ''
    // Chemins d'import, classes utilitaires, clés i18n, sélecteurs : pas du texte UI.
    if (/^[@./]/.test(value)) continue
    if (/^[a-z0-9-]+(\.[a-z0-9_]+)+$/i.test(value)) continue
    if (/^[a-z0-9 :\-[\]#.%/]+$/.test(value)) continue
    out.push(value)
  }
  return out
}

describe('garde-rail état vide canonique des cartes de graphe', () => {
  it('le périmètre est non vide (le garde-rail scanne bien quelque chose)', () => {
    expect(SOURCES.length).toBeGreaterThan(0)
  })

  it('aucun wrapper ne dessine son propre cadre ou sa propre ligne d\'état vide', () => {
    const offenders: string[] = []
    for (const f of SOURCES) {
      const text = readFileSync(f, 'utf8')
      if (HOME_MADE_FRAME.test(text) || HOME_MADE_LINE.test(text)) {
        offenders.push(f.replace(CHARTS_ROOT, 'src/components/charts'))
      }
    }
    expect(
      offenders,
      'état vide dessiné sur place — passer par `EmptyStateNotice` ' +
        `(@/components/ui/empty-state) : ${offenders.join(', ')}`,
    ).toEqual([])
  })

  it('tout fichier qui rend un état vide passe par EmptyStateNotice', () => {
    const offenders: string[] = []
    for (const f of SOURCES) {
      const text = readFileSync(f, 'utf8')
      if (!RENDERS_EMPTY.test(text)) continue
      if (IMPORTS_CANONICAL.test(text)) continue
      offenders.push(f.replace(CHARTS_ROOT, 'src/components/charts'))
    }
    expect(
      offenders,
      'état vide rendu hors du gabarit canonique — importer `EmptyStateNotice` : ' +
        offenders.join(', '),
    ).toEqual([])
  })

  it('aucune string FR en dur dans components/charts', () => {
    const offenders: string[] = []
    for (const f of SOURCES) {
      const raw = readFileSync(f, 'utf8')
      if (LOCALE_DICT.test(raw)) continue // parité FR/EN garantie par le typage
      const body = stripComments(raw)
      for (const literal of uiLiterals(body)) {
        if (FR_WORD.test(literal)) {
          offenders.push(`${f.replace(CHARTS_ROOT, 'src/components/charts')} : "${literal}"`)
        }
      }
    }
    expect(
      offenders,
      'string FR en dur — passer par un manifest i18n (`formatMessage` + ' +
        `lib/i18n/generated/*) ou un \`Record<Locale, T>\` : ${offenders.join(' | ')}`,
    ).toEqual([])
  })
})
