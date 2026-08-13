/**
 * grenadeFx.ts — CE QUE LE LIEN LANCER -> PROJECTILE PERMET DE DIRE, et rien de plus.
 *
 * Le film ne porte AUCUN événement de détonation. Ce qui est certifié, c'est la FIN DE VOL
 * (`projectile-at-rest-state`, 78/79 sur le film de référence) : l'objet ne bouge plus.
 * L'effet posé là est donc un effet de « DERNIÈRE POSITION CONNUE » — jamais un « impact » :
 * pour une grenade à fragmentation la réplication cesse ~1,4 s après le lancer alors que la
 * mèche court jusqu'à ~3 s.
 *
 * PAR TYPE (décision du plan, lot 2 item 2.3) : Shock/Dynamo (rang 2) laisse une nappe
 * ÉLECTRIQUE persistante ~2-3 s — c'est ce que l'arme fait dans le jeu, et la géométrie
 * brisée est déjà la signature de la famille `shock`. Les autres types reçoivent un halo
 * discret, sans affirmation de détonation.
 *
 * Pas de React, pas de canvas : logique pure, testée (grenadeFx.test.ts).
 */
import type { ReplayDocumentReady } from './replayNormalize'

/** Rang du type Shock/Dynamo dans GrenadeLabels — l'ordre des rangs est LA donnée
 *  (établi par deux chaînes indépendantes, cf. replay_labels.toml). */
export const DYNAMO_RANK = 2

/** Rémanences au point de repos, en temps réel : la nappe électrique persiste (~2,5 s,
 *  décision produit « ~2-3 s »), le halo suit la convention des lancers (1,4 s). */
export const DYNAMO_REST_HOLD_MS = 2_500
export const GRENADE_REST_HOLD_MS = 1_400

/** Un effet de fin de vol, précalculé en coordonnées monde. */
export interface GrenadeRestFx {
  /** Frame où le vol s'arrête (t0 + dernier pas du projectile lié). */
  frame: number
  x: number
  y: number
  /** Rang du type de grenade (index dans grenadeLabels). */
  rank: number
  /** Germe stable pour la géométrie brisée : revenir en arrière redonne la même forme. */
  seed: number
}

/**
 * buildGrenadeRestFx relie chaque lancer à la fin de vol de SON projectile.
 *
 * SEULES les fins de vol CERTIFIÉES (`rest`) produisent un effet : l'arrêt de la
 * réplication n'est pas une preuve d'arrêt du projectile, y poser un effet affirmerait un
 * point que le film ne donne pas. Un lien hors bornes (artefact d'une autre version) est
 * ignoré — jamais un effet posé au hasard.
 */
export function buildGrenadeRestFx(doc: ReplayDocumentReady): GrenadeRestFx[] {
  const out: GrenadeRestFx[] = []
  for (const g of doc.grenades) {
    if (g.proj === undefined || g.proj === null) continue
    const pr = doc.projectiles[g.proj]
    if (!pr || !pr.rest || pr.p.length === 0) continue
    const last = pr.p[pr.p.length - 1]
    const frame = (pr.t0 ?? 0) + last[0]
    out.push({
      frame,
      x: last[1],
      y: last[2],
      rank: g.rank ?? 0,
      seed: (frame * 2654435761) % 100003,
    })
  }
  return out
}
