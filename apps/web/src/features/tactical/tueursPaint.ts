/**
 * tueursPaint.ts — la couche « autre joueur » du plan, peinte sur sa propre toile, par-dessus la
 * chaleur : les traits d'abord (sous les points), puis les points. Elle partage la projection du
 * plan (`projectionDuPlan`) : mêmes mètres, même cadrage, même zoom que la chaleur et les zones.
 *
 * Le SURVOL met une contribution en avant (trait plus épais, point plus grand, opacité pleine) et
 * estompe les autres. Les couleurs arrivent résolues (jetons sémantiques, lus par l'appelant).
 */
import type { PointDeLaCouche, RoleDuPoint } from './tueurs.logic'

/** Opacités du trait et du point : au repos, estompés par un survol ailleurs, mis en avant. */
const OPACITE_TRAIT = { repos: 0.4, estompe: 0.12, avant: 1 } as const
const OPACITE_POINT = { repos: 0.9, estompe: 0.3, avant: 1 } as const
/** Épaisseurs du trait et rayons du point, en px. */
const EPAISSEUR = { repos: 1, avant: 2.5 } as const
const RAYON = { repos: 4, avant: 6.5 } as const

export interface CouleursDeLaCouche {
  /** La couleur de chaque rôle (jeton sémantique résolu). */
  roles: Readonly<Record<RoleDuPoint, string>>
  /** L'encre du contour des points (jeton `foreground`, lu sur la toile). */
  encre: string
}

/** peindreLaCouche peint les traits puis les points ; `survol` = l'index de contribution mis en avant. */
export function peindreLaCouche(
  ctx: CanvasRenderingContext2D,
  points: readonly PointDeLaCouche[],
  projeter: (p: { x: number; y: number }) => { x: number; y: number },
  couleurs: CouleursDeLaCouche,
  survol: number | null,
): void {
  const etat = (p: PointDeLaCouche) => (survol === null ? 'repos' : p.index === survol ? 'avant' : 'estompe')
  // La contribution mise en avant se peint en dernier : au-dessus des autres.
  const ordre = [...points].sort((a, b) => Number(a.index === survol) - Number(b.index === survol))
  ctx.save()
  ctx.lineCap = 'round'
  for (const p of ordre) {
    const e = etat(p)
    const a = projeter(p.de)
    const b = projeter(p.vers)
    ctx.globalAlpha = OPACITE_TRAIT[e]
    ctx.strokeStyle = couleurs.roles[p.role]
    ctx.lineWidth = EPAISSEUR[e === 'avant' ? 'avant' : 'repos']
    ctx.beginPath()
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
    ctx.stroke()
  }
  for (const p of ordre) {
    const e = etat(p)
    const b = projeter(p.vers)
    ctx.globalAlpha = OPACITE_POINT[e]
    ctx.fillStyle = couleurs.roles[p.role]
    ctx.strokeStyle = couleurs.encre
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.arc(b.x, b.y, RAYON[e === 'avant' ? 'avant' : 'repos'], 0, 2 * Math.PI)
    ctx.fill()
    ctx.stroke()
  }
  ctx.restore()
}
