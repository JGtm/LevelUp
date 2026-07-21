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
import type { FragDistribution, FragClassEntry } from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

// ── Géométrie du sunburst (reprise fidèle de la maquette validée) ────────────────
const W = 440
const H = 300
const CX = 220
const CY = 142
const R0 = 44 // rayon interne de l'anneau CLASSE
const R1 = 76 // frontière classe / rôle
const R2 = 104 // rayon externe de l'anneau RÔLE
const CALLOUT_Y_TOP = CY - 96
const CALLOUT_Y_BOT = CY + 96
const KNEE_DX = 58
const DIM_OPACITY = 0.22

function polar(r: number, angleDeg: number): [number, number] {
  const t = ((angleDeg - 90) * Math.PI) / 180
  return [CX + r * Math.cos(t), CY + r * Math.sin(t)]
}

/** Chemin SVG d'un arc annulaire entre deux rayons et deux angles (degrés). */
function arcPath(rin: number, rout: number, a0: number, a1: number): string {
  const large = a1 - a0 > 180 ? 1 : 0
  const [x0, y0] = polar(rout, a0)
  const [x1, y1] = polar(rout, a1)
  const [x2, y2] = polar(rin, a1)
  const [x3, y3] = polar(rin, a0)
  return `M${x0} ${y0} A${rout} ${rout} 0 ${large} 1 ${x1} ${y1} L${x2} ${y2} A${rin} ${rin} 0 ${large} 0 ${x3} ${y3} Z`
}

// ── Modèle de rendu (builder PUR, injecté colors + labels → testable sans DOM) ────

export interface FragSunburstColors {
  classColor: (className: string) => string
  roleColor: (className: string, index: number, count: number) => string
  leafColor: (className: string) => string
}

export interface FragSunburstLabels {
  classLabel: (className: string) => string
  roleLabel: (role: string) => string
  formatValue: (n: number) => string
  formatShare: (n: number) => string
}

export interface SunArc {
  key: string
  d: string
  fill: string
  classKey: string
  kind: 'class' | 'role' | 'leaf'
  tipColor: string
  tipTitle: string
  tipSub: string
}

export interface SunCallout {
  key: string
  points: string
  color: string
  label: string
  valueLabel: string
  tx: number
  ly: number
  anchor: 'start' | 'end'
}

export interface SunLegendRow {
  classKey: string
  color: string
  label: string
  valueLabel: string
}

export interface SunModel {
  arcs: SunArc[]
  callouts: SunCallout[]
  legend: SunLegendRow[]
}

interface RoleArcSeed {
  label: string
  value: number
  color: string
  mid: number
  classKey: string
}

function clamp(v: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, v))
}

/** Construit les arcs (classe + rôle/feuille) et collecte les rôles à étiqueter. */
function buildArcs(
  classes: FragClassEntry[],
  total: number,
  colors: FragSunburstColors,
  labels: FragSunburstLabels,
): { arcs: SunArc[]; roleSeeds: RoleArcSeed[] } {
  const arcs: SunArc[] = []
  const roleSeeds: RoleArcSeed[] = []
  let cur = 0
  for (const c of classes) {
    const span = (c.kills / total) * 360
    const a0 = cur
    const a1 = cur + span
    cur = a1
    const classColor = colors.classColor(c.class)
    const share = `${labels.formatValue(c.kills)} · ${labels.formatShare(c.kills)}`
    arcs.push({
      key: `c-${c.class}`,
      d: arcPath(R0, R1, a0, a1),
      fill: classColor,
      classKey: c.class,
      kind: 'class',
      tipColor: classColor,
      tipTitle: labels.classLabel(c.class),
      tipSub: share,
    })
    const roles = c.roles ?? []
    if (roles.length > 0) {
      let rc = a0
      roles.forEach((r, i) => {
        const rs = (r.kills / total) * 360
        const ra0 = rc
        const ra1 = rc + rs
        rc = ra1
        const col = colors.roleColor(c.class, i, roles.length)
        arcs.push({
          key: `r-${c.class}-${r.role}`,
          d: arcPath(R1, R2, ra0, ra1),
          fill: col,
          classKey: c.class,
          kind: 'role',
          tipColor: col,
          tipTitle: `${labels.classLabel(c.class)} · ${labels.roleLabel(r.role)}`,
          tipSub: `${labels.formatValue(r.kills)} · ${labels.formatShare(r.kills)}`,
        })
        roleSeeds.push({ label: labels.roleLabel(r.role), value: r.kills, color: col, mid: (ra0 + ra1) / 2, classKey: c.class })
      })
    } else {
      arcs.push({
        key: `l-${c.class}`,
        d: arcPath(R1, R2, a0, a1),
        fill: colors.leafColor(c.class),
        classKey: c.class,
        kind: 'leaf',
        tipColor: classColor,
        tipTitle: labels.classLabel(c.class),
        tipSub: share,
      })
    }
  }
  return { arcs, roleSeeds }
}

/** Étale les rôles d'un côté (gauche/droite) en lignes de rappel espacées en Y. */
function buildCalloutsForSide(seeds: RoleArcSeed[], right: boolean, labels: FragSunburstLabels): SunCallout[] {
  const points = seeds
    .map((s) => {
      const [ex, ey] = polar(R2, s.mid)
      const [elbowX, elbowY] = polar(R2 + 8, s.mid)
      return { ...s, ex, ey, elbowX, elbowY }
    })
    .sort((a, b) => a.ey - b.ey)
  const tx = right ? W - 6 : 6
  const knee = right ? tx - KNEE_DX : tx + KNEE_DX
  const anchor: 'start' | 'end' = right ? 'end' : 'start'
  return points.map((p, k) => {
    const ly =
      points.length === 1
        ? clamp(p.ey, CALLOUT_Y_TOP, CALLOUT_Y_BOT)
        : CALLOUT_Y_TOP + (k * (CALLOUT_Y_BOT - CALLOUT_Y_TOP)) / (points.length - 1)
    const endX = right ? tx - 2 : tx + 2
    return {
      key: `${p.classKey}-${p.label}`,
      points: `${p.ex},${p.ey} ${p.elbowX},${p.elbowY} ${knee},${ly} ${endX},${ly}`,
      color: p.color,
      label: p.label,
      valueLabel: `${labels.formatValue(p.value)} · ${labels.formatShare(p.value)}`,
      tx,
      ly,
      anchor,
    }
  })
}

/**
 * Builder PUR du modèle de rendu SVG — exporté pour tester la géométrie sans DOM.
 * `total` doit être > 0 (le composant garde ce cas en amont).
 */
export function buildSunburstModel(
  classes: FragClassEntry[],
  total: number,
  colors: FragSunburstColors,
  labels: FragSunburstLabels,
): SunModel {
  if (total <= 0 || classes.length === 0) return { arcs: [], callouts: [], legend: [] }
  const { arcs, roleSeeds } = buildArcs(classes, total, colors, labels)
  // Côté = position HORIZONTALE (X = cos) du point de l'arc externe vs centre :
  // moitié droite du cercle → label à droite, moitié gauche → à gauche. Utiliser
  // sin (composante Y) répartirait par haut/bas et ferait traverser les lignes.
  const isRight = (mid: number): boolean => Math.cos(((mid - 90) * Math.PI) / 180) >= 0
  const rightSeeds = roleSeeds.filter((s) => isRight(s.mid))
  const leftSeeds = roleSeeds.filter((s) => !isRight(s.mid))
  const callouts = [
    ...buildCalloutsForSide(rightSeeds, true, labels),
    ...buildCalloutsForSide(leftSeeds, false, labels),
  ]
  const legend: SunLegendRow[] = classes.map((c) => ({
    classKey: c.class,
    color: colors.classColor(c.class),
    label: labels.classLabel(c.class),
    valueLabel: labels.formatValue(c.kills),
  }))
  return { arcs, callouts, legend }
}

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
}

export function FragSunburst({ distribution, title, externalHoveredClass = null, onClassHover }: FragSunburstProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const paletteVersion = useColorPaletteVersion()
  const numLoc = intlLocale(appLocale)
  const labelsBase = useSunburstLabels()
  const total = distribution?.total_kills ?? 0
  const classes = distribution?.classes ?? []
  const [hovered, setHovered] = useState<string | null>(null)
  const [tip, setTip] = useState<TipState | null>(null)

  // Part = valeur / total, formatée en pourcentage localisé.
  const labels: FragSunburstLabels = useMemo(
    () => ({
      ...labelsBase,
      formatShare: (n: number) =>
        `${(total > 0 ? (n / total) * 100 : 0).toLocaleString(numLoc, { maximumFractionDigits: 1 })} %`,
    }),
    // labelsBase dérive de appLocale ; on l'exclut (référence neuve à chaque rendu).
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [total, appLocale, numLoc],
  )

  const model = useMemo(
    () => buildSunburstModel(classes, total, SUNBURST_COLORS, labels),
    // couleurs réactives à la palette : paletteVersion force le recalcul.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [classes, total, appLocale, paletteVersion],
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

  return (
    <div className="relative rounded-lg border border-border bg-card" data-testid="frag-sunburst">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{cardTitle}</div>
      <div className="p-3">
        <div className="relative flex items-center justify-center">
          <svg
            viewBox={`0 0 ${W} ${H}`}
            role="img"
            aria-label={cardTitle}
            className="h-auto w-full"
            style={{ overflow: 'visible' }}
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
          </svg>
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
            <span className="text-[10px] uppercase tracking-wider text-muted-foreground">{centerLabel}</span>
            <span className="text-2xl font-bold leading-none">{total.toLocaleString(numLoc)}</span>
          </div>
        </div>
        <div className="mt-2 flex flex-wrap gap-x-3 gap-y-1" data-testid="frag-legend">
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
