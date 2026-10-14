/**
 * tueursPaint.ts — la couche « autre joueur » du plan, peinte sur sa propre toile, par-dessus la
 * chaleur : les traits d'abord (sous les marqueurs), puis les marqueurs. Elle partage la projection
 * du plan (`projectionDuPlan`) : mêmes mètres, même cadrage, même zoom que la chaleur et les zones.
 *
 * COULEUR : tueurs et victimes sont des ADVERSAIRES, donc à la couleur d'équipe adverse de l'app (celle
 * que le joueur a choisie dans ses réglages d'accessibilité), cernés d'un liseré à l'encre de premier
 * plan du thème qui détache la forme de la carte comme de la chaleur. Le tueur et la victime se
 * distinguent par la FORME — losange, rond — et non par la couleur.
 *
 * Les tailles arrivent en px, déjà calculées pour l'échelle du plan (`dimensionsDeLaCouche`). Le
 * SURVOL ou la SÉLECTION met une contribution en avant (trait plus épais, marqueur plus grand,
 * opacité pleine) et estompe les autres.
 */
import { formeDuRole, type DimensionsDeLaCouche, type PointDeLaCouche, type RoleDuPoint } from './tueurs.logic'

/** Opacités du trait et du marqueur : au repos, estompés par une mise en avant ailleurs, mis en avant. */
const OPACITE_TRAIT = { repos: 0.4, estompe: 0.12, avant: 0.9 } as const
const OPACITE_POINT = { repos: 0.95, estompe: 0.3, avant: 1 } as const
/** Épaisseur du liseré (encre) autour des marqueurs, et de part et d'autre des traits. */
const LISERE_MARQUEUR_PX = 1.5
const LISERE_TRAIT_PX = 2
/** Le losange a la même emprise visuelle que le rond : sa demi-diagonale est un peu plus longue. */
const FACTEUR_LOSANGE = 1.3

export interface CouleursDeLaCouche {
  /** La couleur d'équipe adverse (marqueurs et traits). */
  adverse: string
  /** L'encre de premier plan du thème : liseré des marqueurs. */
  encre: string
  /** La couleur du fond du thème : liseré sous les traits. */
  fond: string
}

function tracerForme(ctx: CanvasRenderingContext2D, role: RoleDuPoint, x: number, y: number, r: number): void {
  ctx.beginPath()
  if (formeDuRole(role) === 'rond') {
    ctx.arc(x, y, r, 0, 2 * Math.PI)
    return
  }
  const d = r * FACTEUR_LOSANGE
  ctx.moveTo(x, y - d)
  ctx.lineTo(x + d, y)
  ctx.lineTo(x, y + d)
  ctx.lineTo(x - d, y)
  ctx.closePath()
}

/** peindreLaCouche peint les traits puis les marqueurs ; `enAvant` = l'index de contribution en relief. */
export function peindreLaCouche(
  ctx: CanvasRenderingContext2D,
  points: readonly PointDeLaCouche[],
  projeter: (p: { x: number; y: number }) => { x: number; y: number },
  couleurs: CouleursDeLaCouche,
  dim: DimensionsDeLaCouche,
  enAvant: number | null,
): void {
  const etat = (p: PointDeLaCouche) => (enAvant === null ? 'repos' : p.index === enAvant ? 'avant' : 'estompe')
  // La contribution mise en avant se peint en dernier : au-dessus des autres.
  const ordre = [...points].sort((a, b) => Number(a.index === enAvant) - Number(b.index === enAvant))
  ctx.save()
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  for (const p of ordre) {
    const e = etat(p)
    const a = projeter(p.de)
    const b = projeter(p.vers)
    const largeur = e === 'avant' ? dim.epaisseurTraitAvant : dim.epaisseurTrait
    ctx.globalAlpha = OPACITE_TRAIT[e]
    // Le liseré du fond sous le trait le détache de la chaleur et d'un fond clair comme sombre.
    ctx.strokeStyle = couleurs.fond
    ctx.lineWidth = largeur + LISERE_TRAIT_PX
    ctx.beginPath()
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
    ctx.stroke()
    ctx.strokeStyle = couleurs.adverse
    ctx.lineWidth = largeur
    ctx.beginPath()
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
    ctx.stroke()
  }
  for (const p of ordre) {
    const e = etat(p)
    const b = projeter(p.vers)
    ctx.globalAlpha = OPACITE_POINT[e]
    ctx.fillStyle = couleurs.adverse
    ctx.strokeStyle = couleurs.encre
    ctx.lineWidth = LISERE_MARQUEUR_PX
    tracerForme(ctx, p.role, b.x, b.y, e === 'avant' ? dim.rayonPointAvant : dim.rayonPoint)
    ctx.fill()
    ctx.stroke()
  }
  ctx.restore()
}
