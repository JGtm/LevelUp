/**
 * coequipiersPaint.ts — la couche des coéquipiers à la mort, peinte sur sa propre toile, par-dessus la
 * chaleur et la couche des tueurs : le cercle du rayon du radar, les traînes, le trait du frag d'un
 * coéquipier sur le tueur, puis les marques. Elle partage la projection du plan (mêmes mètres, même
 * cadrage, même zoom).
 *
 * TOUT SE MESURE EN MÈTRES MONDE, converti par l'échelle du plan : le cercle est un vrai rayon de
 * 18 m ; les rayons des marques et les épaisseurs viennent de `dimensionsDeLaCouche` (modèle de
 * taille du rejeu 2D, `screenLengthPx`) — jamais des pixels fixes.
 *
 * COULEURS : jetons d'ÉQUIPE de l'app (`team-ally` pour les coéquipiers, `team-enemy` pour le tueur),
 * lus par l'appelant dans la palette active ; le mort est à l'encre du thème, sur un disque de la
 * couleur du fond. Chaque marque est cernée d'un liseré de la couleur du fond pour se détacher de la
 * carte comme de la chaleur. Les rôles se distinguent par la FORME (tueur : losange ; mort : rond à
 * cercle épais ; coéquipier : rond plein dans le rayon, rond creux au-delà), jamais par la couleur seule.
 */
import type { MarqueDeScene, SceneDuPlan } from './mort.logic'
import type { DimensionsDeLaCouche } from './tueurs.logic'

export interface CouleursDesCoequipiers {
  /** Encre de premier plan du thème (cercle, mort). */
  encre: string
  /** Couleur du fond du thème (liseré, disque du mort). */
  fond: string
  /** Jeton d'équipe alliée (coéquipiers, frag du coéquipier). */
  allie: string
  /** Jeton d'équipe adverse (tueur de la séquence). */
  adverse: string
}

/** Opacités : le nuage se lit en masse, les marques d'une mort sont pleines. */
const OPACITE = { nuage: 0.55, traine: 0.65, trait: 0.95, cercle: 0.55, remplissageCercle: 0.07 } as const
const LISERE_PX = 1.5
const TRAIT_CERCLE_PX = 1.5
const TIRETS_PX: readonly [number, number] = [5, 4]
const FACTEUR_MORT = 1.25
const ANNEAU_MORT_PX = 2.5
const FACTEUR_NUAGE = 0.6
const FACTEUR_LOSANGE = 1.3
const ANNEAU_CREUX_PX = 2

type Projeter = (p: { x: number; y: number }) => { x: number; y: number }

function disque(ctx: CanvasRenderingContext2D, x: number, y: number, r: number): void {
  ctx.beginPath()
  ctx.arc(x, y, r, 0, 2 * Math.PI)
}

function losange(ctx: CanvasRenderingContext2D, x: number, y: number, r: number): void {
  const d = r * FACTEUR_LOSANGE
  ctx.beginPath()
  ctx.moveTo(x, y - d)
  ctx.lineTo(x + d, y)
  ctx.lineTo(x, y + d)
  ctx.lineTo(x - d, y)
  ctx.closePath()
}

function peindreUneMarque(
  ctx: CanvasRenderingContext2D,
  m: MarqueDeScene,
  s: { x: number; y: number },
  c: CouleursDesCoequipiers,
  dim: DimensionsDeLaCouche,
): void {
  ctx.lineWidth = LISERE_PX
  ctx.strokeStyle = c.fond
  switch (m.role) {
    case 'nuage':
      ctx.globalAlpha = OPACITE.nuage
      ctx.fillStyle = c.allie
      disque(ctx, s.x, s.y, dim.rayonPoint * FACTEUR_NUAGE)
      ctx.fill()
      return
    case 'mort':
      ctx.globalAlpha = 1
      ctx.fillStyle = c.fond
      ctx.strokeStyle = c.encre
      ctx.lineWidth = ANNEAU_MORT_PX
      disque(ctx, s.x, s.y, dim.rayonPoint * FACTEUR_MORT)
      ctx.fill()
      ctx.stroke()
      return
    case 'tueur':
      ctx.globalAlpha = 1
      ctx.fillStyle = c.adverse
      losange(ctx, s.x, s.y, dim.rayonPoint)
      ctx.fill()
      ctx.stroke()
      return
    case 'coequipier':
      ctx.globalAlpha = 1
      disque(ctx, s.x, s.y, dim.rayonPoint)
      if (m.plein) {
        ctx.fillStyle = c.allie
        ctx.fill()
        ctx.stroke()
        return
      }
      // Creux : un anneau de la couleur alliée sur un liseré du fond, centre vide.
      ctx.strokeStyle = c.fond
      ctx.lineWidth = ANNEAU_CREUX_PX + 2 * LISERE_PX
      ctx.stroke()
      ctx.strokeStyle = c.allie
      ctx.lineWidth = ANNEAU_CREUX_PX
      ctx.stroke()
  }
}

/** peindreLaScene — le cercle, les traînes, les traits, puis les marques (la plus importante en dernier). */
export function peindreLaScene(
  ctx: CanvasRenderingContext2D,
  scene: SceneDuPlan,
  projeter: Projeter,
  scalePxParM: number,
  couleurs: CouleursDesCoequipiers,
  dim: DimensionsDeLaCouche,
): void {
  ctx.save()
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  if (scene.cercle) {
    const o = projeter(scene.cercle.centre)
    disque(ctx, o.x, o.y, scene.cercle.rayonM * scalePxParM)
    ctx.globalAlpha = OPACITE.remplissageCercle
    ctx.fillStyle = couleurs.allie
    ctx.fill()
    ctx.globalAlpha = OPACITE.cercle
    ctx.setLineDash([...TIRETS_PX])
    ctx.strokeStyle = couleurs.encre
    ctx.lineWidth = TRAIT_CERCLE_PX
    ctx.stroke()
    ctx.setLineDash([])
  }
  for (const traine of scene.traines) {
    ctx.globalAlpha = OPACITE.traine
    ctx.strokeStyle = traine.role === 'tueur' ? couleurs.adverse : traine.role === 'mort' ? couleurs.encre : couleurs.allie
    ctx.lineWidth = dim.epaisseurTrait
    for (const segment of traine.segments) {
      ctx.beginPath()
      segment.forEach((p, i) => {
        const s = projeter(p)
        if (i === 0) ctx.moveTo(s.x, s.y)
        else ctx.lineTo(s.x, s.y)
      })
      ctx.stroke()
    }
  }
  for (const trait of scene.traits) {
    const a = projeter(trait.de)
    const b = projeter(trait.vers)
    ctx.globalAlpha = OPACITE.trait
    ctx.strokeStyle = couleurs.allie
    ctx.lineWidth = dim.epaisseurTraitAvant
    ctx.beginPath()
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
    ctx.stroke()
  }
  // Le mort se peint en dernier : au-dessus de ce qui l'entoure.
  const ordre = [...scene.marques].sort((a, b) => Number(a.role === 'mort') - Number(b.role === 'mort'))
  for (const m of ordre) peindreUneMarque(ctx, m, projeter(m.pt), couleurs, dim)
  ctx.restore()
}
