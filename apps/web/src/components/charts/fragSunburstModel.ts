/**
 * Modèle de rendu du FragSunburst (builder PUR + géométrie) — extrait de
 * FragSunburst.tsx pour que le module de composant n'exporte que des composants
 * (react-refresh/only-export-components) et testable sans DOM.
 */
import type { FragClassEntry } from '@/lib/api/types'

// ── Géométrie du sunburst (reprise fidèle de la maquette validée) ────────────────
export const W = 440
export const H = 300
export const CX = 220
export const CY = 142
const R0 = 44 // rayon interne de l'anneau CLASSE
const R1 = 76 // frontière classe / rôle
const R2 = 104 // rayon externe de l'anneau RÔLE
const CALLOUT_Y_TOP = CY - 96
const CALLOUT_Y_BOT = CY + 96
const KNEE_DX = 58
export const DIM_OPACITY = 0.22

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

/**
 * Étale les rôles (niveau 2) d'un côté en lignes de rappel — UNE étiquette par rôle.
 * L'étiquette est placée AU PLUS PRÈS de la hauteur de SON arc (`ey`), poussée vers le bas
 * seulement pour éviter le chevauchement avec la précédente (points triés par ey croissant).
 * Le côté gauche/droite est décidé en amont par la position HORIZONTALE de l'arc (cf.
 * buildSunburstModel) → traits courts, aucune traversée du donut. La répartition uniforme
 * de la maquette étalait les étiquettes sur toute la hauteur et éloignait le label de son arc
 * quand les rôles se regroupaient d'un côté (classe dominante).
 */
function buildCalloutsForSide(
  seeds: RoleArcSeed[],
  right: boolean,
  labels: FragSunburstLabels,
): SunCallout[] {
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
  // Écart vertical mini entre deux étiquettes = hauteur d'une étiquette à DEUX lignes (nom
  // au-dessus + « valeur · % » en dessous, ~22-26 px), sinon les CHIFFRES d'une étiquette
  // chevauchent le nom de la suivante. Quand TROP de rôles d'un côté pour tenir dans
  // [TOP, BOT] à cet écart (ex. Explorer d'une cible : beaucoup de rôles), on RÉDUIT l'écart
  // (réparti régulièrement) au lieu d'empiler en bas — l'ancien clamp à BOT empilait les
  // étiquettes en surnombre → chevauchement.
  const MIN_GAP = 26
  const n = points.length
  const available = CALLOUT_Y_BOT - CALLOUT_Y_TOP
  // Écart effectif : garanti ≤ available/(n-1) → toutes les étiquettes tiennent dans [TOP, BOT].
  const gap = n > 1 ? Math.min(MIN_GAP, available / (n - 1)) : MIN_GAP
  // Passe 1 (top-down) : chaque étiquette au plus près de sa position naturelle (ey), écart
  // mini garanti ; peut déborder sous BOT.
  const lys: number[] = []
  let prev = -Infinity
  for (const p of points) {
    prev = Math.max(p.ey, CALLOUT_Y_TOP, prev + gap)
    lys.push(prev)
  }
  // Passe 2 : si le bloc déborde en bas, recentrage bottom-up (jamais d'empilement — le span
  // (n-1)*gap ≤ available garantit lys[0] ≥ TOP après recalage).
  if (n > 0 && lys[n - 1] > CALLOUT_Y_BOT) {
    lys[n - 1] = CALLOUT_Y_BOT
    for (let i = n - 2; i >= 0; i--) lys[i] = Math.min(lys[i], lys[i + 1] - gap)
  }
  const endX = right ? tx - 2 : tx + 2
  return points.map((p, i) => {
    const ly = lys[i]
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
  // Côté = position HORIZONTALE (cos) de l'arc : chaque étiquette va du côté où son arc EST
  // réellement (droite si x >= centre, gauche sinon) → le trait ne traverse JAMAIS le donut.
  // La maquette décidait par la composante Y (sin, haut/bas) ; ça marche quand les rôles
  // sont répartis en haut ET en bas, mais dès qu'une classe DOMINE (ex. arme de poing 65 %),
  // tous les rôles étiquetés se retrouvent dans la moitié haute → tous à gauche → traits qui
  // traversent (bug observé). cos règle ça quelle que soit la répartition.
  const isRight = (mid: number): boolean => Math.cos(((mid - 90) * Math.PI) / 180) >= 0
  const rightSeeds = roleSeeds.filter((s) => isRight(s.mid))
  const leftSeeds = roleSeeds.filter((s) => !isRight(s.mid))
  const callouts = [
    ...buildCalloutsForSide(rightSeeds, true, labels),
    ...buildCalloutsForSide(leftSeeds, false, labels),
  ]
  // I5 (V7.1) : chaque entrée de légende affiche le POURCENTAGE du total (« Libellé — NN % »),
  // pas le seul décompte brut — cohérent avec le tooltip des arcs (déjà valeur · %). Le tiret
  // cadratin est un séparateur visuel (pas un libellé à traduire) : aucune clé i18n requise.
  const legend: SunLegendRow[] = classes.map((c) => ({
    classKey: c.class,
    color: colors.classColor(c.class),
    label: labels.classLabel(c.class),
    valueLabel: `— ${labels.formatShare(c.kills)}`,
  }))
  return { arcs, callouts, legend }
}
