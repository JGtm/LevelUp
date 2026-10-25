/**
 * shotImpactFx.ts — L'EXPLOSION D'UN PROJECTILE DE TIR (mortier, roquettes, bombe...).
 *
 * LE LIEN VIENT DU SERVEUR : `Projectile.shot` est l'index, dans `shots`, du tir dont le projectile
 * est né (appariement par coïncidence de frame et de position, aucune ambiguïté publiée). La
 * détonation, elle, est celle que le film écrit (`detoT/detoX/detoY`, `detoLue` vrai).
 *
 * CE QUI DÉCIDE D'UNE EXPLOSION EST UN FAIT DÉCLARÉ PAR ARME, `impact`, publié par le titre sur le
 * libellé de l'arme à pied (`WeaponLabel.impact`) ou sur l'entrée du registre de véhicule
 * (`VehicleWeapon.impact`). JAMAIS la forme de rendu `fx` : le mortier du Wraith est `plasma` comme
 * le pistolet plasma, qui n'explose pas, et le canon du Scorpion est `explosive` sans projectile
 * publié. Garde-rail : shotImpactFx.test.ts.
 *
 * Dégradations propres, sans erreur : un projectile sans `shot` (artefact antérieur au schéma 104,
 * grenade, balle sans tir publié), un tir hors bornes, une arme sans `impact`, une détonation
 * non lue (`detoLue` faux : repli sur la dernière position, pas une explosion lue) ne posent rien.
 *
 * Les entrées rejoignent celles des grenades (`GrenadeRestFx`) : MÊME calque, MÊME timeline, MÊME
 * germe, MÊME interrupteur de couche. Pas de React, pas de canvas : logique pure.
 */
import { fxTintOf, type FxTint } from '../layers/fxInk'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { buildGrenadeRestFx, explosionSeedOf, type GrenadeRestFx } from './grenadeFx'
import { vehicleWeaponOf } from './vehicleWeaponRegistry'
import { impactSoundStem } from '../sound/impactSound'

/** Nature de la déflagration d'un projectile de tir dont l'arme ne déclare pas de teinte. */
const DEFAULT_IMPACT_TINT: FxTint = 'blast'

/** Ce que le document dit de l'explosion d'une arme : sa nature, et le stem de son son (absent = muette). */
interface ImpactReading {
  tint: FxTint
  impactStem?: string
}

/**
 * impactOf lit, pour la clé d'arme `w` d'un tir, le fait « son projectile explose ». Le registre
 * des armes à pied d'abord, celui des armes de véhicule ensuite — le même ordre que le style des
 * tirs (`buildShotFx`). `null` = pas d'explosion.
 */
function impactOf(doc: ReplayDocumentReady, w: string | undefined): ImpactReading | null {
  if (!w) return null
  const label = doc.weaponLabels?.[w]
  if (label) {
    if (!label.impact) return null
    return { tint: tintOrDefault(label.tint), impactStem: impactSoundStem(label.key || w) }
  }
  const vehicle = vehicleWeaponOf(doc, w)
  if (!vehicle?.impact) return null
  return { tint: tintOrDefault(vehicle.tint), impactStem: vehicle.impactSound || undefined }
}

function tintOrDefault(raw: string | undefined): FxTint {
  const tint = fxTintOf(raw)
  return tint === 'neutral' ? DEFAULT_IMPACT_TINT : tint
}

/**
 * buildShotImpactFx relie chaque projectile à l'explosion de SON tir, quand l'arme du tir déclare
 * `impact` et que la détonation a été LUE dans le film.
 */
export function buildShotImpactFx(doc: ReplayDocumentReady): GrenadeRestFx[] {
  const out: GrenadeRestFx[] = []
  for (const pr of doc.projectiles) {
    if (pr.shot === undefined || pr.shot === null || !pr.detoLue) continue
    const shot = doc.shots[pr.shot]
    if (!shot) continue
    const impact = impactOf(doc, shot.w)
    if (!impact) continue
    out.push({
      frame: pr.detoT,
      x: pr.detoX,
      y: pr.detoY,
      rank: -1,
      rest: !!pr.rest,
      seed: explosionSeedOf(pr.detoT),
      tint: impact.tint,
      impactStem: impact.impactStem,
    })
  }
  return out
}

/** Toutes les fins de vol explosives du film : celles des grenades, puis celles des tirs. L'écran
 *  et le son lisent CETTE liste, jamais deux datations parallèles. */
export function buildRestFx(doc: ReplayDocumentReady): GrenadeRestFx[] {
  return [...buildGrenadeRestFx(doc), ...buildShotImpactFx(doc)]
}
