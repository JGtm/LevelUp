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
import { EXPLOSION_MS } from './explosionFx'
import type { FxTint } from './fxInk'
import type { ReplayDocumentReady } from './replayNormalize'

/** Rang du type Shock/Dynamo dans GrenadeLabels — l'ordre des rangs est LA donnée
 *  (établi par deux chaînes indépendantes, cf. replay_labels.toml). */
export const DYNAMO_RANK = 2

/**
 * restKindOf — CE QUE FAIT UNE GRENADE AU BOUT DE SON VOL, par type.
 *
 * L'ORDRE DES RANGS EST LA DONNÉE (0 Frag, 1 Plasma, 2 Dynamo, 3 Spike), établi par deux
 * chaînes indépendantes et publié par le titre ; ce qui est CHOISI ici, c'est la mise en
 * scène, et elle suit ce que l'arme fait dans le jeu :
 *  - Frag et Plasma DÉTONENT : explosion particulaire (item 3.3) ;
 *  - Dynamo N'EXPLOSE PAS — elle pose une nappe électrique persistante (~2,5 s). Règle
 *    livrée au lot 2.3 et confirmée par l'utilisateur, inchangée ici ;
 *  - Spike : elle se PLANTE puis projette ses pointes. C'est aussi le seul type dont la fin
 *    de vol est certifiée `at-rest` (mesure : 15 vols liés sur 21, 0 pour les trois autres)
 *    — l'objet s'immobilise AVANT de se déclencher. Elle reçoit donc une explosion, teintée
 *    « cristal » plutôt que feu. **C'est le point que le plan laisse À TRANCHER À L'ŒIL**
 *    (item 3.3) : si l'utilisateur la préfère sans détonation, une ligne suffit à la rendre
 *    au halo.
 *
 * Un rang inconnu (type qu'un titre publierait sans qu'on le connaisse) garde le HALO
 * discret : jamais l'effet d'un type voisin.
 */
export type GrenadeRestKind = 'explosion' | 'nappe' | 'halo'

export function restKindOf(rank: number): GrenadeRestKind {
  if (rank === DYNAMO_RANK) return 'nappe'
  return rank === 0 || rank === 1 || rank === 3 ? 'explosion' : 'halo'
}

/**
 * explosionTintOf — la NATURE de la déflagration d'un type, dans le vocabulaire des teintes
 * (fxInk.ts) : une grenade à fragmentation est une déflagration chimique, une grenade plasma
 * une décharge d'énergie froide, une Spike une gerbe de cristal. Même logique que les tirs :
 * la couleur dit ce qui explose, jamais qui l'a lancée.
 */
export function explosionTintOf(rank: number): FxTint {
  if (rank === 1) return 'plasma_cool'
  if (rank === 3) return 'needle'
  return 'blast'
}

/**
 * Rémanences au point de repos, en temps réel : la nappe électrique persiste (~2,5 s,
 * décision produit « ~2-3 s »), et la fenêtre des trois types qui détonent vaut EXACTEMENT
 * la timeline de leur explosion.
 *
 * ELLE EST ALIGNÉE SUR `EXPLOSION_MS`, ET C'EST UN INVARIANT, pas une coïncidence : cette
 * fenêtre BORNE le dessin (`drawGrenadeRestLayer` sort dès `age > hold`). Restée à 1,4 s
 * pendant que la timeline passait à 2,4 s, elle aurait coupé chaque explosion à 58 % de sa
 * course — exactement le défaut que la planche du 16/08 signalait (« trop bref »), déplacé
 * d'un cran. Le test de `grenadeFx` rejoue l'égalité.
 */
export const DYNAMO_REST_HOLD_MS = 2_500
export const GRENADE_REST_HOLD_MS = EXPLOSION_MS

/** Rémanence du badge de lancer sur la FICHE (le `.gic` du POC) : celle des lancers. */
export const GRENADE_THROW_HOLD_MS = 1_400

/** Un effet de fin de vol, précalculé en coordonnées monde. */
export interface GrenadeRestFx {
  /** Frame où la réplication du vol s'arrête (t0 + dernier pas du projectile lié). */
  frame: number
  x: number
  y: number
  /** Rang du type de grenade (index dans grenadeLabels), `-1` quand le film ne le publie pas.
   *  PAS DE REPLI SUR 0 : `restKindOf` rend alors le halo discret — exactement ce que sa
   *  propre doctrine promet — et le SON se tait (EXPLOSION_SOUND_STEMS, grenadeSound.ts).
   *  Replier sur 0 ferait détoner une frag, et sonner SON explosion, pour une grenade dont
   *  le type n'est pas établi : l'effet d'une voisine, ce que ce fichier refuse partout. */
  rank: number
  /** Fin de vol CERTIFIÉE (`projectile-at-rest-state`). MESURE sur le témoin 000d5950 :
   *  le composant n'apparaît QUE sur des vols Spike (15/21 liés) — jamais frag, plasma ni
   *  Dynamo (0/44), dont l'objet DÉTONE au lieu de s'immobiliser. Conditionner l'effet à
   *  cette certification éteindrait donc trois types sur quatre — l'effet se pose au
   *  dernier point répliqué, et l'écran dit ce que c'est. */
  rest: boolean
  /** Germe stable pour la géométrie brisée : revenir en arrière redonne la même forme. */
  seed: number
}

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

/**
 * buildGrenadeRestFx relie chaque lancer à la fin de vol de SON projectile.
 *
 * LE POINT EST LA DERNIÈRE POSITION RÉPLIQUÉE — c'est exactement ce que l'écran en dit
 * (« dernière position connue », jamais « impact » : aucun événement de détonation
 * n'existe dans le film, et pour une frag la réplication cesse ~1,4 s avant la mèche).
 * L'effet n'est PAS conditionné à la certification `at-rest` : mesurée par type, elle
 * n'existe que sur les Spike (cf. GrenadeRestFx.rest) — l'exiger éteindrait les autres.
 * Un lien hors bornes (artefact d'une autre version) est ignoré — jamais un effet posé
 * au hasard.
 *
 * DEPUIS LE 2026-08-18, ELLE DATE AUSSI LE SON (lot R2-G) : l'explosion de chaque grenade
 * part à `frame`, la même que l'effet à l'écran (grenadeSound.ts). C'est délibérément la
 * MÊME fonction et non une règle parallèle — deux datations qui divergeraient feraient
 * sonner l'explosion à côté de son image, sans que rien ne le dise.
 */
export function buildGrenadeRestFx(doc: ReplayDocumentReady): GrenadeRestFx[] {
  const out: GrenadeRestFx[] = []
  for (const g of doc.grenades) {
    if (g.proj === undefined || g.proj === null) continue
    const pr = doc.projectiles[g.proj]
    if (!pr || pr.p.length === 0) continue
    const last = pr.p[pr.p.length - 1]
    const frame = (pr.t0 ?? 0) + last[0]
    out.push({
      frame,
      x: last[1],
      y: last[2],
      rank: g.rank ?? -1,
      rest: !!pr.rest,
      seed: (frame * 2654435761) % 100003,
    })
  }
  return out
}
