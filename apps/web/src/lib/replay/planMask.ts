/**
 * planMask — ROGNER UNE COUCHE DE ZONES NOMMÉES AU PLAN : l'alpha de l'image plan est le masque.
 *
 * L'image « plan » d'un fond de carte est transparente là où la carte n'existe pas (silhouette du sol,
 * murs, étages). Une zone nommée peut déborder de cette silhouette ; peinte telle quelle, elle colorie du
 * vide. Ce module est LE SEUL endroit qui sait en faire un masque, pour les deux supports de dessin :
 *
 *   - SVG (page Tactique) : `<PlanMaskDef id href cadre />` (`PlanMaskDef.tsx`) pose un `<mask>` dont l'`<image>`
 *     est l'image plan ; la couche de zones porte `mask={planMaskUrl(id)}`. Le masque est de type ALPHA
 *     (`mask-type: alpha`), pas de luminance : seule la transparence de l'image compte, pas sa teinte ;
 *   - canvas (calque des zones du rejeu 2D, cuit hors écran) : `rognerAuPlan(ctx, peindrePlan)` ouvre un
 *     `destination-in` le temps que l'appelant peigne l'image plan, au MÊME rectangle que le fond.
 *
 * Dans les deux cas, l'appelant décide s'il y a un plan : pas d'image plan (carte sans vue plan, image pas
 * encore chargée ou en échec) = il ne rogne pas, et les zones restent entières. Ce module ne devine rien.
 *
 * CADRE : le rectangle que l'image occupe dans l'espace de la couche, avant toute rotation de la carte
 * (la rotation est celle de l'élément ou du contexte, pas du masque : le masque tourne avec la couche).
 */

/** Le rectangle de l'image plan dans l'espace de coordonnées de la couche à rogner (unités de la couche). */
export interface CadreDuPlan {
  x: number
  y: number
  width: number
  height: number
}

/**
 * planMaskId rend un identifiant de masque valide dans `url(#...)` : `useId` de React rend `:r1:`, que
 * l'on remet au jeu de caractères sûr, et un préfixe évite toute collision avec un autre `id` de la page.
 */
export function planMaskId(base: string): string {
  return `plan-mask-${base.replace(/[^A-Za-z0-9_-]/g, '_')}`
}

/** planMaskUrl est la valeur de l'attribut `mask` d'une couche SVG rognée par le masque `id`. */
export function planMaskUrl(id: string): string {
  return `url(#${id})`
}

/**
 * rognerAuPlan garde, de ce qui est DÉJÀ peint sur `ctx`, ce que l'image plan couvre : `peindrePlan`
 * dessine l'image plan (au rectangle et à la rotation du fond) pendant que la composition est
 * `destination-in`. Le contexte retrouve son état d'avant (composition, transformation).
 */
export function rognerAuPlan(ctx: CanvasRenderingContext2D, peindrePlan: () => void): void {
  ctx.save()
  ctx.globalCompositeOperation = 'destination-in'
  peindrePlan()
  ctx.restore()
}
