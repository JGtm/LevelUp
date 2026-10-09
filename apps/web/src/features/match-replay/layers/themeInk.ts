/**
 * themeInk.ts — LA LECTURE D'UNE VARIABLE DU THÈME pour le canvas du rejeu, et L'EXPORT QUI SE
 * PEINT TOUJOURS DANS LE THÈME SOMBRE (décision D9 du plan « formats vidéo », 2026-09-16).
 *
 * # LE BESOIN
 *
 * Un clip exporté doit avoir la MÊME apparence quelle que soit la préférence de qui l'exporte :
 * le thème sombre, celui pour lequel la carte et ses encres ont été dessinées. Mais basculer
 * `data-theme` sur la page le temps de l'export ferait clignoter toute l'interface.
 *
 * # POURQUOI NE PAS SIMPLEMENT LIRE UN ÉLÉMENT PORTANT `data-theme="dark"`
 *
 * Les variables du thème sont déclarées sur `:root[data-theme='dark']` / `:root[data-theme='light']`
 * (cf. `styles/globals.css`) : ces sélecteurs ne visent QUE la racine. Un élément isolé qui porte
 * l'attribut ne les reçoit pas — il hérite des valeurs du thème ACTIF. Il faudrait dupliquer tout
 * le bloc sombre sous un autre sélecteur, c'est-à-dire tenir deux définitions du thème.
 *
 * # CE QUI EST FAIT
 *
 * Pendant un export (`isExportActive`), la valeur est lue DANS LES RÈGLES de la feuille de style,
 * avec la cascade que le navigateur appliquerait en thème sombre : le style en ligne de la racine
 * (palettes d'accessibilité, `applyPalette`) l'emporte, puis `:root[data-theme=dark]`, puis
 * `:root`. Une référence `var(--x)` se résout de la même façon. La table est construite une fois
 * par export (clé : la mise en page demandée), pas à chaque lecture.
 *
 * Hors export, rien ne change : `getComputedStyle` sur la racine, comme avant.
 */
import { exportLayoutRequested, isExportActive } from '../export/exportLayoutStore'

/** Le thème dans lequel l'export se peint, quel que soit celui de la page. */
export const EXPORT_THEME = 'dark'

interface ThemeTable {
  root: Map<string, string>
  themed: Map<string, string>
}

let cacheKey: object | null = null
let cache: ThemeTable | null = null

/** Les sélecteurs d'une règle, normalisés : sans guillemets ni espaces. */
function selectorsOf(rule: CSSStyleRule): string[] {
  return rule.selectorText.split(',').map((s) => s.replace(/["'\s]/g, ''))
}

/** Une règle `@media` dont la condition ne tient pas n'appliquerait rien : on ne la lit pas. */
function mediaMatches(rule: CSSRule): boolean {
  const media = (rule as CSSMediaRule).media
  if (!media || typeof window.matchMedia !== 'function') return true
  return window.matchMedia(media.mediaText).matches
}

function collect(rules: CSSRuleList, table: ThemeTable): void {
  for (const rule of Array.from(rules)) {
    // LES RÈGLES DE STYLE D'ABORD : une règle imbriquée porte AUSSI `cssRules`.
    if (!('selectorText' in rule)) {
      if ('cssRules' in rule && mediaMatches(rule)) collect((rule as CSSGroupingRule).cssRules, table)
      continue
    }
    const style = rule as CSSStyleRule
    const sels = selectorsOf(style)
    const target = sels.includes(`:root[data-theme=${EXPORT_THEME}]`)
      ? table.themed
      : sels.includes(':root')
        ? table.root
        : null
    if (!target) continue
    for (const name of Array.from(style.style)) {
      if (name.startsWith('--')) target.set(name, style.style.getPropertyValue(name).trim())
    }
  }
}

function themeTable(): ThemeTable {
  const key = exportLayoutRequested()
  if (cache && cacheKey === key) return cache
  const table: ThemeTable = { root: new Map(), themed: new Map() }
  for (const sheet of Array.from(document.styleSheets)) {
    try {
      collect(sheet.cssRules, table)
    } catch {
      // Feuille d'une autre origine : ses règles ne sont pas lisibles, et le thème du dépôt n'y
      // vit pas. On la saute.
    }
  }
  cacheKey = key
  cache = table
  return table
}

/** La valeur d'une variable dans le thème de l'export, `var()` résolus ; `''` si absente. */
function exportThemeValue(name: string, depth = 0): string {
  const inline = document.documentElement.style.getPropertyValue(name).trim()
  const t = themeTable()
  const raw = inline || t.themed.get(name) || t.root.get(name) || ''
  if (depth > 8) return raw
  return raw.replace(/var\((--[\w-]+)(?:\s*,\s*([^)]*))?\)/g, (_m, ref: string, fallback?: string) => {
    return exportThemeValue(ref, depth + 1) || (fallback ?? '').trim()
  })
}

/**
 * readThemeVar — la valeur d'une variable du thème pour le canvas : celle du thème SOMBRE pendant
 * un export, celle du thème actif sinon. Chaîne vide côté serveur ou si la variable manque.
 */
export function readThemeVar(name: string): string {
  if (typeof document === 'undefined') return ''
  if (isExportActive()) return exportThemeValue(name)
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}
