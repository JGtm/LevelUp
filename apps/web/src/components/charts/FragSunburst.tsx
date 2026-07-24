/**
 * FragSunburst — carte hiérarchique « Répartition des frags » v2, rendue en SVG
 * (2 anneaux). Anneau INTERNE = classe (axe manipulation : Épaule/Poing/Lourde/
 * Mêlée/Grenade/Capacités spartanes), anneau EXTERNE = rôle (fonction de combat).
 *
 * Forme validée (maquette Option A) : sunburst DÉSENCOMBRÉ —
 *   - les RÔLES (niveau 2) sont étiquetés par des LIGNES DE RAPPEL réparties
 *     gauche/droite (point sur l'arc externe → coude → genou → texte au bord),
 *     triées par Y et espacées verticalement ; texte = nom du rôle + « valeur · % » ;
 *   - les CLASSES sont listées en LÉGENDE sous le SVG (pastille + nom + valeur) —
 *     AUCUN texte sur les arcs de classe ;
 *   - les classes FEUILLES (poing/grenade/résidu) : pas de libellé externe, teinte
 *     éclaircie de la classe sur l'anneau externe, présentes seulement en légende ;
 *   - centre = total ; survol d'un arc = tooltip (classe · rôle + valeur + %) +
 *     estompage des autres classes.
 *
 * Couleurs : gamme « Antagonistes » réactive à la palette (fragClassColor via token) ;
 * rôles = teintes éclaircies (fragRoleColor). Double encodage couleur + label +
 * position. Composant PARTAGÉ (Synthesis/Match view/Timeseries/Sessions) — cf.
 * .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P1.4. Rend null si total 0.
 */
import { useMemo, useState, type MouseEvent as ReactMouseEvent } from 'react'

import { fragClassColor, fragRoleColor, fragLeafColor } from '@/lib/accessibility/scales'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { FragDistribution } from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  buildSunburstModel,
  W,
  H,
  CX,
  CY,
  DIM_OPACITY,
  type FragSunburstColors,
  type FragSunburstLabels,
  type SunArc,
} from './fragSunburstModel'

// La géométrie, les types et le builder pur buildSunburstModel sont extraits dans
// ./fragSunburstModel (react-refresh : ce module n'exporte que des composants).

// ── Câblage React (i18n + couleurs réactives à la palette) ───────────────────────

/** Formatters i18n hors part (formatShare dépend du total → composé côté composant). */
type FragSunburstBaseLabels = Omit<FragSunburstLabels, 'formatShare'>

function useSunburstLabels(): FragSunburstBaseLabels {
  const appLocale = useAppShellStore((s) => s.locale)
  const numLoc = intlLocale(appLocale)
  return {
    classLabel: (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale),
    roleLabel: (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, appLocale),
    formatValue: (n: number) => n.toLocaleString(numLoc),
  }
}

const SUNBURST_COLORS: FragSunburstColors = {
  classColor: fragClassColor,
  roleColor: fragRoleColor,
  leafColor: fragLeafColor,
}

interface TipState {
  x: number
  y: number
  color: string
  title: string
  sub: string
}

export interface FragSunburstProps {
  distribution?: FragDistribution | null
  title?: string
  /**
   * Survol LIÉ (optionnel) : classe survolée pilotée par un composant frère
   * (ex. `FragWeaponBreakdown` via `MatchFragCard`). Quand renseignée, les autres
   * classes du sunburst sont estompées même si le survol vient de l'extérieur.
   * Non fournie → le composant reste autonome (son propre survol interne pilote).
   */
  externalHoveredClass?: string | null
  /** Remonté au parent au survol d'un arc/légende (classe parente ou null en sortie). */
  onClassHover?: (classKey: string | null) => void
  /** Classe(s) utilitaire(s) fusionnée(s) sur la carte racine (ex. `lg:col-span-2`). */
  className?: string
  /** Masque le libellé « Frags » au centre (opt-in Match view) → compteur seul, centré dans l'anneau. */
  hideCenterLabel?: boolean
  /** Largeur max (px) du SVG, centré (opt-in Match view) — borne la hauteur `h-auto`. */
  maxWidthPx?: number
  /** Position de la légende des classes : 'bottom' (défaut, sous l'anneau), 'left' (colonne
   *  le long de la bordure gauche, opt-in Match view), ou 'none' (légende NON rendue en
   *  interne — à placer soi-même via <FragClassLegend>, ex. encart Explorer : légende
   *  centrée en bas du bloc entier). */
  legendSide?: 'bottom' | 'left' | 'none'
  /** Mode « nu » (opt-in) : rend le SVG + légende SANS la carte racine (bordure + barre de
   *  titre) — pour intégrer le sunburst dans un bloc parent (ex. encart Explorer) sans
   *  bloc-dans-bloc. Défaut false = carte complète (comportement des autres surfaces). */
  bare?: boolean
  /**
   * Hauteur fixe (px, opt-in) du contenu SVG — remplace le calage naturel par LARGEUR
   * (aspect ratio 440:300 du viewBox, `h-auto`). À utiliser quand la carte doit avoir la
   * MÊME hauteur qu'une carte voisine dont la hauteur est elle-même fixe (ex. session-detail
   * « Répartition des frags » / « Outils de destruction », I5 V7.1) — sans ça, le sunburst
   * se met à l'échelle de sa largeur de colonne (2/3 vs 1/3, ou pleine largeur en pile) et sa
   * hauteur rendue diverge de celle du graphe voisin. Le SVG garde son ratio (`preserveAspectRatio`
   * par défaut = `xMidYMid meet`) à l'intérieur de la boîte fixée, centré, sans déformation.
   */
  heightPx?: number
}

export function FragSunburst({
  distribution,
  title,
  externalHoveredClass = null,
  onClassHover,
  className = '',
  hideCenterLabel = false,
  maxWidthPx,
  legendSide = 'bottom',
  bare = false,
  heightPx,
}: FragSunburstProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const paletteVersion = useColorPaletteVersion()
  const numLoc = intlLocale(appLocale)
  const labelsBase = useSunburstLabels()
  const total = distribution?.total_kills ?? 0
  const classes = distribution?.classes ?? []
  // Dénominateur des ANGLES d'arc ET du ratio %. En données SAINES, Σ classes == total
  // (unattributed = résidu) → arcTotal == total → rendu INCHANGÉ. Si une anomalie backend
  // fait Σ classes > total (double-comptage : sur H5 un kill mêlée/assassinat est attribué à
  // l'arme tenue dans les weapon kills ET recompté dans le compteur natif), on borne par Σ :
  // les arcs remplissent EXACTEMENT 360° sans déborder ni se chevaucher — sinon les derniers
  // arcs recouvrent les premiers et deux rôles se retrouvent au même angle (étiquettes qui
  // semblent partir de la même source). Le centre garde le vrai total (me.Kills).
  const arcTotal = Math.max(total, classes.reduce((s, c) => s + c.kills, 0)) || 1
  const [hovered, setHovered] = useState<string | null>(null)
  const [tip, setTip] = useState<TipState | null>(null)

  // Part = valeur / total, formatée en pourcentage localisé.
  const labels: FragSunburstLabels = useMemo(
    () => ({
      ...labelsBase,
      formatShare: (n: number) =>
        `${((n / arcTotal) * 100).toLocaleString(numLoc, { maximumFractionDigits: 1 })} %`,
    }),
    // labelsBase dérive de appLocale ; on l'exclut (référence neuve à chaque rendu).
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [arcTotal, appLocale, numLoc],
  )

  const model = useMemo(
    () => buildSunburstModel(classes, arcTotal, SUNBURST_COLORS, labels),
    // couleurs réactives à la palette : paletteVersion force le recalcul.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [classes, arcTotal, appLocale, paletteVersion],
  )

  if (total <= 0 || classes.length === 0) return null

  const tc = getEChartsThemeColors()
  const cardTitle = title ?? formatMessage(fragsManifest, 'frags.charts.sunburst_title', appLocale)
  const centerLabel = formatMessage(fragsManifest, 'frags.charts.center_total_label', appLocale)

  // Classe active = survol interne (arc/légende) OU survol externe (composant frère lié).
  const activeClass = hovered ?? externalHoveredClass
  const showTip = (e: ReactMouseEvent, arc: SunArc) => {
    setHovered(arc.classKey)
    onClassHover?.(arc.classKey)
    setTip({ x: e.clientX, y: e.clientY, color: arc.tipColor, title: arc.tipTitle, sub: arc.tipSub })
  }
  const clearTip = () => {
    setHovered(null)
    onClassHover?.(null)
    setTip(null)
  }
  const arcOpacity = (classKey: string) => (activeClass && classKey !== activeClass ? DIM_OPACITY : 1)

  // Légende des classes (pastille + nom + valeur). Position pilotée par legendSide : 'bottom'
  // = ligne sous l'anneau (défaut, autres surfaces) ; 'left' = colonne verticale le long de la
  // bordure gauche, à côté de l'anneau (opt-in Match view).
  // En mode 'bottom' AVEC hauteur fixe (heightPx — cards Sessions, drawer compare) :
  // hauteur de légende RÉSERVÉE constante (2 lignes), pour que deux cards côte à
  // côte (session courante vs comparée) gardent la même hauteur totale même si
  // leurs sessions n'ont pas le même nombre de classes de frags (parité drawer,
  // demande utilisateur 2026-07-24).
  const legendBlock = (
    <div
      className={
        legendSide === 'left'
          ? 'flex shrink-0 flex-col gap-1'
          : heightPx
            ? 'mt-2 flex min-h-12 flex-wrap content-start gap-x-3 gap-y-1'
            : 'mt-2 flex flex-wrap gap-x-3 gap-y-1'
      }
      data-testid="frag-legend"
    >
      {model.legend.map((row) => (
        <span
          key={row.classKey}
          className="inline-flex cursor-default items-center gap-1.5 rounded px-1 py-0.5 text-xs"
          style={{ opacity: activeClass && row.classKey !== activeClass ? DIM_OPACITY : 1 }}
          onMouseEnter={() => {
            setHovered(row.classKey)
            onClassHover?.(row.classKey)
          }}
          onMouseLeave={() => {
            setHovered(null)
            onClassHover?.(null)
          }}
        >
          <span className="inline-block h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: row.color }} />
          {row.label}
          <span className="text-muted-foreground">{row.valueLabel}</span>
        </span>
      ))}
    </div>
  )

  return (
    <div className={`relative ${bare ? '' : 'rounded-lg border border-border bg-card'} ${className}`} data-testid="frag-sunburst">
      {!bare && <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{cardTitle}</div>}
      <div className={legendSide === 'left' ? 'flex items-center gap-3 p-3' : 'p-3'}>
        {legendSide === 'left' && legendBlock}
        <div
          className={`relative flex items-center justify-center ${legendSide === 'left' ? 'min-w-0 flex-1' : ''}`}
          style={heightPx ? { height: heightPx } : undefined}
        >
          <svg
            viewBox={`0 0 ${W} ${H}`}
            role="img"
            aria-label={cardTitle}
            // heightPx fourni → hauteur FIXE (preserveAspectRatio par défaut centre/contient
            // sans déformer) ; sinon comportement historique (hauteur pilotée par la largeur).
            className={`w-full ${heightPx ? 'h-full' : 'h-auto'} ${maxWidthPx ? 'mx-auto block' : ''}`}
            style={{ overflow: 'visible', ...(maxWidthPx ? { maxWidth: maxWidthPx } : {}) }}
          >
            {model.arcs.map((a) => (
              <path
                key={a.key}
                d={a.d}
                fill={a.fill}
                style={{ stroke: 'var(--card)', strokeWidth: 2, opacity: arcOpacity(a.classKey), transition: 'opacity .12s', cursor: 'pointer' }}
                onMouseEnter={(e) => showTip(e, a)}
                onMouseMove={(e) => setTip((t) => (t ? { ...t, x: e.clientX, y: e.clientY } : t))}
                onMouseLeave={clearTip}
              >
                <title>{`${a.tipTitle} — ${a.tipSub}`}</title>
              </path>
            ))}
            {model.callouts.map((co) => (
              <g key={co.key} data-testid="frag-callout">
                <polyline points={co.points} fill="none" stroke={co.color} strokeWidth={1} opacity={0.6} />
                <text x={co.tx} y={co.ly - 2} textAnchor={co.anchor} fill={tc.text} style={{ fontSize: 10, opacity: 0.9 }}>
                  {co.label}
                </text>
                <text x={co.tx} y={co.ly + 10} textAnchor={co.anchor} fill={co.color} style={{ fontSize: 10, fontWeight: 600 }}>
                  {co.valueLabel}
                </text>
              </g>
            ))}
            {hideCenterLabel && (
              <text
                x={CX}
                y={CY}
                textAnchor="middle"
                dominantBaseline="central"
                fill={tc.text}
                style={{ fontSize: 26, fontWeight: 700 }}
              >
                {total.toLocaleString(numLoc)}
              </text>
            )}
          </svg>
          {!hideCenterLabel && (
            <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground">{centerLabel}</span>
              <span className="text-2xl font-bold leading-none">{total.toLocaleString(numLoc)}</span>
            </div>
          )}
        </div>
        {legendSide === 'bottom' && legendBlock}
      </div>
      {tip && (
        <div
          className="pointer-events-none fixed z-50 max-w-[240px] rounded-md border border-border bg-card px-2.5 py-1.5 text-xs shadow-lg"
          style={{
            left: typeof window !== 'undefined' && tip.x + 260 > window.innerWidth ? tip.x - 254 : tip.x + 14,
            top: tip.y + 14,
          }}
          role="tooltip"
        >
          <div className="mb-0.5 flex items-center gap-1.5 font-medium">
            <span className="inline-block h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: tip.color }} />
            {tip.title}
          </div>
          <div className="text-muted-foreground">{tip.sub}</div>
        </div>
      )}
    </div>
  )
}

// ── FragClassLegend : légende des classes EXTRACTIBLE ─────────────────────────
// Rend la légende (pastille + nom + valeur) AILLEURS que sous l'anneau — pour un
// <FragSunburst legendSide="none" /> dont on place la légende soi-même (ex. encart
// Explorer : sunburst + Top armes en rangée, légende centrée EN BAS du bloc entier).
// Survol LIÉ au sunburst via hoveredClass/onClassHover partagés. Recalcule le modèle
// de légende depuis la MÊME distribution (2e usage du setup classes/arcTotal — sous le
// plafond ≤2 copies ; buildSunburstModel reste la source unique du modèle).
export function FragClassLegend({
  distribution,
  hoveredClass = null,
  onClassHover,
  className = '',
}: {
  distribution?: FragDistribution | null
  hoveredClass?: string | null
  onClassHover?: (classKey: string | null) => void
  className?: string
}) {
  const appLocale = useAppShellStore((s) => s.locale)
  const paletteVersion = useColorPaletteVersion()
  const labelsBase = useSunburstLabels()
  const numLoc = intlLocale(appLocale)
  const classes = distribution?.classes ?? []
  const total = distribution?.total_kills ?? 0
  const arcTotal = Math.max(total, classes.reduce((s, c) => s + c.kills, 0)) || 1
  const labels: FragSunburstLabels = useMemo(
    () => ({
      ...labelsBase,
      formatShare: (n: number) => `${((n / arcTotal) * 100).toLocaleString(numLoc, { maximumFractionDigits: 1 })} %`,
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [arcTotal, appLocale, numLoc],
  )
  const legend = useMemo(
    () => buildSunburstModel(classes, arcTotal, SUNBURST_COLORS, labels).legend,
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [classes, arcTotal, appLocale, paletteVersion],
  )
  if (legend.length === 0) return null
  return (
    <div className={`flex flex-wrap justify-center gap-x-3 gap-y-1 ${className}`} data-testid="frag-legend">
      {legend.map((row) => (
        <span
          key={row.classKey}
          className="inline-flex cursor-default items-center gap-1.5 rounded px-1 py-0.5 text-xs"
          style={{ opacity: hoveredClass && row.classKey !== hoveredClass ? DIM_OPACITY : 1 }}
          onMouseEnter={() => onClassHover?.(row.classKey)}
          onMouseLeave={() => onClassHover?.(null)}
        >
          <span className="inline-block h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: row.color }} />
          {row.label}
          <span className="text-muted-foreground">{row.valueLabel}</span>
        </span>
      ))}
    </div>
  )
}
