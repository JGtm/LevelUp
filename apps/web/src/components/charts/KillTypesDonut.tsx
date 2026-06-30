/**
 * KillTypesDonut — donut SVG « répartition des frags » à lignes de rappel.
 *
 * Donut catégoriel (parts mutuellement exclusives) rendu en SVG pur (pas
 * d'ECharts) avec labels déportés par lignes de rappel réparties par côté.
 * Partagé entre l'Explorer (profil cible) et la Synthesis.
 *
 * Couleurs : les tokens sont fournis par l'appelant via `DonutSlice.token`.
 * Utiliser des indices chart-series DISTINCTS (ex. 1/6/7/8) plutôt que 2-5 (un
 * dégradé séquentiel illisible en catégoriel et non color-blind friendly).
 */
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

export interface DonutSlice {
  label: string
  count: number
  token: SemanticToken
}

function fmtPctRatio(value: number, locale: string): string {
  return `${(value * 100).toLocaleString(locale, { maximumFractionDigits: 1 })}%`
}

function fmtInt(value: number, locale: string): string {
  return value.toLocaleString(locale, { maximumFractionDigits: 0 })
}

// Géométrie du donut. Repère angulaire : 0 = midi, sens horaire.
// rOuter agrandi (~38 % de la largeur) pour un donut nettement plus lisible ;
// la largeur est élargie en conséquence pour préserver la marge des labels.
const DONUT = { w: 320, h: 184, cx: 160, cy: 92, rOuter: 60, stroke: 20, yTop: 18, yBot: 166, labelMargin: 86 }

interface Leader {
  slice: DonutSlice
  startFrac: number
  dashLen: number
  edgeX: number
  edgeY: number
  elbowX: number
  elbowY: number
  kneeX: number
  textX: number
  labelY: number
  anchor: 'start' | 'end'
}

/** computeLeaders calcule arcs + lignes de rappel (labels répartis par côté). */
function computeLeaders(slices: DonutSlice[], total: number, circ: number): Leader[] {
  let acc = 0
  const raw = slices.map((slice) => {
    const startFrac = acc
    const frac = slice.count / total
    acc += frac
    const midTheta = (startFrac + frac / 2) * 2 * Math.PI
    const sinT = Math.sin(midTheta)
    const cosT = Math.cos(midTheta)
    const right = sinT >= 0
    return {
      slice, startFrac, dashLen: frac * circ, right, sinT, cosT,
      edgeX: DONUT.cx + DONUT.rOuter * sinT,
      edgeY: DONUT.cy - DONUT.rOuter * cosT,
      elbowX: DONUT.cx + (DONUT.rOuter + 9) * sinT,
      elbowY: DONUT.cy - (DONUT.rOuter + 9) * cosT,
    }
  })
  const out: Leader[] = []
  for (const right of [true, false]) {
    const side = raw.filter((r) => r.right === right).sort((a, b) => a.elbowY - b.elbowY)
    side.forEach((r, k) => {
      const labelY = side.length === 1
        ? Math.min(Math.max(r.elbowY, DONUT.yTop), DONUT.yBot)
        : DONUT.yTop + (k * (DONUT.yBot - DONUT.yTop)) / (side.length - 1)
      const textX = right ? DONUT.w - DONUT.labelMargin : DONUT.labelMargin
      out.push({
        slice: r.slice, startFrac: r.startFrac, dashLen: r.dashLen,
        edgeX: r.edgeX, edgeY: r.edgeY, elbowX: r.elbowX, elbowY: r.elbowY,
        kneeX: right ? textX - 6 : textX + 6, textX, labelY,
        anchor: right ? 'start' : 'end',
      })
    })
  }
  return out
}

/** KillTypesDonut rend le donut SVG. Renvoie null si total == 0. */
export function KillTypesDonut({ slices, locale }: { slices: DonutSlice[]; locale: string }) {
  const total = slices.reduce((acc, s) => acc + s.count, 0)
  if (total === 0) return null
  const innerR = DONUT.rOuter - DONUT.stroke / 2
  const circ = 2 * Math.PI * innerR
  const leaders = computeLeaders(slices, total, circ)

  return (
    <svg width="100%" viewBox={`0 0 ${DONUT.w} ${DONUT.h}`} className="max-w-[700px]">
      <circle cx={DONUT.cx} cy={DONUT.cy} r={innerR} fill="none" stroke={tokenCssVar('perf-tier-5')} strokeWidth={DONUT.stroke} opacity="0.15" />
      {leaders.map((l, i) => (
        <circle
          key={`arc-${i}`}
          cx={DONUT.cx} cy={DONUT.cy} r={innerR} fill="none"
          stroke={tokenCssVar(l.slice.token)} strokeWidth={DONUT.stroke}
          strokeDasharray={`${l.dashLen} ${circ - l.dashLen}`}
          strokeDashoffset={-l.startFrac * circ}
          transform={`rotate(-90 ${DONUT.cx} ${DONUT.cy})`}
        />
      ))}
      <text x={DONUT.cx} y={DONUT.cy} textAnchor="middle" dominantBaseline="central" className="fill-foreground text-base font-semibold">
        {fmtInt(total, locale)}
      </text>
      {leaders.map((l, i) => (
        <g key={`lead-${i}`}>
          <polyline
            points={`${l.edgeX},${l.edgeY} ${l.elbowX},${l.elbowY} ${l.kneeX},${l.labelY}`}
            fill="none" stroke={tokenCssVar(l.slice.token)} strokeWidth="1" opacity="0.55"
          />
          <text x={l.textX} y={l.labelY - 1} textAnchor={l.anchor} className="fill-foreground" opacity="0.8" style={{ fontSize: '9px' }}>
            {l.slice.label}
          </text>
          <text x={l.textX} y={l.labelY + 10} textAnchor={l.anchor} style={{ fill: tokenCssVar(l.slice.token), fontSize: '10px', fontWeight: 600 }}>
            {`${fmtInt(l.slice.count, locale)} · ${fmtPctRatio(l.slice.count / total, locale)}`}
          </text>
        </g>
      ))}
    </svg>
  )
}
