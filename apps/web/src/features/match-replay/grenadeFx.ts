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

/** Rémanence du badge de lancer sur la FICHE (le `.gic` du POC) : celle des lancers. */
export const GRENADE_THROW_HOLD_MS = 1_400

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
/** Le lancer actif d'un joueur : son rang, et l'âge du geste (en frames). */
export interface ThrowReading {
  rank: number
  age: number
}

/**
 * grenadeThrowActive — le DERNIER lancer d'un joueur dans la fenêtre de rémanence.
 *
 * L'AUTEUR EST ÉCRIT DANS LE FILM (Grenade.i = index de joueur du film) : il n'est pas
 * deviné, contrairement au tireur d'un tir — c'est ce qui autorise un badge sur la FICHE.
 * La jointure passe par l'index de film du roster, jamais par un ordre supposé.
 */
export function grenadeThrowActive(
  doc: ReplayDocumentReady,
  filmIndex: number,
  frame: number,
  holdFrames: number,
): ThrowReading | null {
  let best: ThrowReading | null = null
  for (const g of doc.grenades) {
    if (g.i !== filmIndex) continue
    const age = frame - g.t
    if (age < 0 || age > holdFrames) continue
    if (!best || age < best.age) best = { rank: g.rank ?? 0, age }
  }
  return best
}

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
