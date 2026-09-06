/**
 * meleeStar.ts — L'ÉTOILE DU COUP DE MÊLÉE FATAL.
 *
 * D'OÙ ELLE VIENT. « Vu que le son est court et discret, on peut afficher un effet visuel
 * spécial ? » (bilan du 2026-08-18, D3). Le lot R2-V avait proposé un éclat en CROIX ; le
 * verdict du 18/08 après-midi tranche : « une ÉTOILE, forme dessin animé, pas une croix ».
 *
 * POURQUOI UNE ÉTOILE SE LIT, ET POURQUOI LA CROIX NON. La carte porte déjà deux croix — le
 * marqueur de fin de vie et les axes des effets de mort — et une troisième aurait ajouté un
 * sens à un signe qui en a déjà deux. L'étoile, elle, n'existe nulle part ailleurs sur ce
 * canvas : le trait de mort est un axe, l'explosion un disque, l'éclair de bouche une flamme,
 * la nappe Dynamo un réseau d'arcs. C'est la FORME qui dit « corps à corps », pas la couleur.
 *
 * CE QU'ELLE N'AFFIRME PAS : aucune direction. Elle est symétrique, et c'est voulu — le film
 * ne date aucun impact de mêlée, et le couple tueur/victime d'un corps à corps tombe presque
 * toujours sous le seuil de 1,5 px qui rend un axe non fiable (cf. drawKillFxLayer). Elle se
 * pose donc au LIEU DE LA MORT, jamais entre deux points.
 *
 * Pas de React, pas de token : la couleur arrive résolue de l'appelant (celle du tueur, comme
 * tout effet de mort).
 */

/** Durée de l'étoile, en ms — celle proposée sur la planche (R2-3) et validée. */
export const MELEE_STAR_MS = 400

/** Nombre de BRANCHES. Huit : quatre auraient refait une croix, seize un disque hérissé. */
const STAR_BRANCHES = 8

/** Rayons des sommets, en pixels d'écran : la pointe longue et le creux entre deux pointes. */
const STAR_OUTER_PX = 13
const STAR_INNER_PX = 5

/** Le noyau incandescent au centre, en pixels d'écran. */
const STAR_CORE_PX = 2.6

/** Épaisseur du contour de la pointe. */
const STAR_LINE_PX = 1.6

/**
 * meleeStarProgress — l'avancement de l'étoile dans [0, 1], ou null quand elle est finie.
 *
 * `reduced` fige l'étoile à son plein éclat (0,45) pendant toute sa durée : sous « mouvement
 * réduit » elle ne jaillit pas, elle est là puis n'est plus là.
 */
export function meleeStarProgress(ageMs: number, reduced: boolean): number | null {
  if (ageMs < 0 || ageMs > MELEE_STAR_MS) return null
  return reduced ? 0.45 : ageMs / MELEE_STAR_MS
}

/**
 * drawMeleeStar — l'étoile au point de la mort.
 *
 * DEUX HORLOGES, comme les autres effets du canvas : elle JAILLIT (le premier tiers l'ouvre à
 * sa taille pleine) puis s'ÉTEINT (les deux tiers restants la font pâlir). Une étoile qui
 * apparaîtrait à sa taille finale ne se lirait pas comme un coup porté.
 */
export function drawMeleeStar(
  ctx: CanvasRenderingContext2D,
  at: { x: number; y: number },
  progress: number,
  k: number,
  color: string,
): void {
  const ouverture = Math.min(1, progress / 0.33)
  const extinction = progress < 0.33 ? 1 : Math.max(0, (1 - progress) / 0.67)
  const outer = STAR_OUTER_PX * k * ouverture
  const inner = STAR_INNER_PX * k * ouverture
  ctx.save()
  ctx.strokeStyle = color
  ctx.fillStyle = color
  ctx.lineJoin = 'round'
  ctx.lineWidth = STAR_LINE_PX * k
  ctx.beginPath()
  for (let i = 0; i < STAR_BRANCHES * 2; i++) {
    const a = (i / (STAR_BRANCHES * 2)) * Math.PI * 2 - Math.PI / 2
    const r = i % 2 === 0 ? outer : inner
    const px = at.x + Math.cos(a) * r
    const py = at.y + Math.sin(a) * r
    if (i === 0) ctx.moveTo(px, py)
    else ctx.lineTo(px, py)
  }
  ctx.closePath()
  // Le corps de l'étoile est PLEIN mais discret, son contour franc : c'est le contour qui
  // porte la forme sur un fond de carte, le remplissage qui lui donne sa masse.
  ctx.globalAlpha = 0.35 * extinction
  ctx.fill()
  ctx.globalAlpha = 0.95 * extinction
  ctx.stroke()
  ctx.globalAlpha = Math.min(1, 1.1 * extinction * extinction)
  ctx.beginPath()
  ctx.arc(at.x, at.y, STAR_CORE_PX * k, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()
}
