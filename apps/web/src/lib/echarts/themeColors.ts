/**
 * themeColors — résolution runtime des couleurs sémantiques shadcn pour
 * ECharts. ECharts est rendu en `<canvas>` et n'accepte pas de CSS vars
 * dans ses options : on doit lire la valeur computée au moment de la
 * génération de l'option.
 *
 * À combiner avec `useThemeVersion()` (cf. `./useThemeVersion.ts`) pour
 * forcer le re-mémoization de l'option lors d'un toggle de thème.
 */

export interface EChartsThemeColors {
  /** Couleur des labels d'axe — var(--muted-foreground). */
  axisLabel: string
  /** Ligne d'axe — var(--border). */
  axisLine: string
  /** Gridlines (splitLine) — var(--border). */
  splitLine: string
  /** Bande paire splitArea radar (fond translucide léger). */
  splitAreaA: string
  /** Bande impaire splitArea radar (fond translucide un cran plus fort). */
  splitAreaB: string
  /** Texte (labels valeurs, tooltips) — var(--foreground). */
  text: string
  /** Fond tooltip — var(--popover). */
  tooltipBg: string
  /** Bordure tooltip — var(--border). */
  tooltipBorder: string
  /**
   * Surface de la carte qui porte le chart — var(--card). Opaque : sert à
   * détourer un tracé posé sur une aire colorée (cerne couleur de fond), là
   * où `CHART_BG` ('transparent') ne peut rien masquer.
   */
  surface: string
  /** true si le thème actif est sombre (classe `dark` sur <html>). */
  isDark: boolean
}

/**
 * Lit les CSS vars du thème actif et les retourne sous forme de chaînes
 * directement consommables par ECharts.
 *
 * Les vars sémantiques sont en `oklch(...)` et `color-mix(...)` est utilisé
 * pour les bandes splitArea (alpha contrôlé). Les fallbacks ne servent qu'à
 * un environnement de test sans CSS résolu (jsdom).
 */
export function getEChartsThemeColors(): EChartsThemeColors {
  if (typeof document === 'undefined') {
    return FALLBACK_COLORS
  }
  const cs = getComputedStyle(document.documentElement)
  const get = (name: string, fallback: string) => cs.getPropertyValue(name).trim() || fallback

  const muted = get('--muted', '#374151')
  const border = get('--border', '#374151')
  const popover = get('--popover', '#1f2937')

  return {
    axisLabel: get('--muted-foreground', '#9ca3af'),
    axisLine: border,
    splitLine: border,
    splitAreaA: `color-mix(in oklch, ${muted} 15%, transparent)`,
    splitAreaB: `color-mix(in oklch, ${muted} 30%, transparent)`,
    text: get('--foreground', '#f3f4f6'),
    tooltipBg: popover,
    tooltipBorder: border,
    surface: get('--card', '#171717'),
    isDark: document.documentElement.getAttribute('data-theme') === 'dark',
  }
}

const FALLBACK_COLORS: EChartsThemeColors = {
  axisLabel: '#9ca3af',
  axisLine: '#374151',
  splitLine: '#374151',
  splitAreaA: 'rgba(55,65,81,0.15)',
  splitAreaB: 'rgba(55,65,81,0.30)',
  text: '#f3f4f6',
  tooltipBg: '#1f2937',
  tooltipBorder: '#374151',
  surface: '#171717',
  isDark: true,
}
