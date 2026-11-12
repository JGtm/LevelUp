/**
 * projectileOwners.ts — À QUI APPARTIENT CHAQUE PROJECTILE DU FILM, pour Seul et Masquer.
 *
 * Le panneau de joueur retire du dessin les projectiles et les grenades du joueur concerné. Un
 * projectile ne porte pas son auteur : il le tient de ce qui le lie. Une GRENADE écrit son auteur
 * (`Grenade.slot`, à l'image du lancer `Grenade.t`) ; un projectile de TIR le tient de son tir
 * (`Projectile.shot` -> `Shot.slot`, à l'image du tir). La grenade prime quand les deux le lient.
 *
 * L'IMAGE COMPTE : le slot de bipède est réattribué à chaque réapparition, et `shownAtSlot` désigne
 * des VIES (un slot ET l'image qu'elle couvre). Le vol et l'explosion restent donc ceux du joueur
 * qui a lancé, même s'il est mort ou a changé de vie depuis. Un projectile sans lien (artefact
 * ancien, tir ambigu) n'a pas de propriétaire : `null`, rien n'en est deviné.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** La vie qui a lancé : son slot et l'image du lancer (ou du tir). */
export interface ProjectileOwner {
  slot: number
  frame: number
}

/** Un propriétaire par projectile, dans l'ordre de `doc.projectiles` ; `null` = auteur inconnu. */
export function projectileOwners(
  doc: Pick<ReplayDocumentReady, 'projectiles' | 'grenades' | 'shots'>,
): Array<ProjectileOwner | null> {
  const owners: Array<ProjectileOwner | null> = doc.projectiles.map((pr) => {
    if (pr.shot === undefined || pr.shot === null) return null
    const shot = doc.shots[pr.shot]
    return shot ? { slot: shot.slot, frame: shot.t } : null
  })
  for (const g of doc.grenades) {
    if (g.proj === undefined || g.proj === null || g.proj < 0 || g.proj >= owners.length) continue
    owners[g.proj] = { slot: g.slot, frame: g.t }
  }
  return owners
}
