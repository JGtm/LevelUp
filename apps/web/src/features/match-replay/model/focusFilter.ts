/**
 * focusFilter.ts — CE QUE « SEUL » ET « MASQUER » FONT À CHAQUE CALQUE QUI N'EST PAS UN PION.
 *
 * LA RÈGLE (utilisateur, 2026-11-15) : tout calque affiché suit Seul et Masquer pour ce qui appartient
 * à un joueur. Seul n'affiche que ce qui est à lui ; Masquer retire ce qui est à lui. Ce qui n'appartient
 * à personne (véhicule vide, drapeau à sa base, zone, socle d'arme) reste affiché : c'est le décor du
 * match. Un élément dont l'attribution ne se lit pas reste affiché lui aussi — rien n'est deviné.
 *
 * UNE SEULE RÈGLE, ÉCRITE UNE FOIS : `ownershipShown` (playerPanelLogic.ts). Ce module ne dit que QUELLES
 * sont les parties de chaque élément, champ du document par champ du document ; chaque calque reçoit un
 * prédicat par genre d'élément et n'embarque aucune règle. Les calques de pions (pistes, tirs, traces des
 * vies, projectiles) gardent `shownAtSlot` : ils portent déjà un slot et une image dans la vie de leur auteur.
 */
import type {
  ReplayEquipmentPlacement,
  ReplayGroundWeapon,
  ReplayVehicleRide,
} from '@/lib/api/types'

import type { KillFxEntry } from './killFx'
import { lifeSlotAt } from './livesMarks'
import { buildLivesBySlot } from './livesPosition'
import {
  killFxShown,
  ownershipShown,
  partyOf,
  type ElementParty,
  type PlayerFocus,
} from './playerPanelLogic'
import { vehicleActiveRides } from './vehiclesLayer'
import type { ReplayDocumentReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'

/** Un prédicat par genre d'élément : vrai = l'élément se dessine. */
export interface FocusFilter {
  /** Un effet de mort : tueur et victime. */
  killFx: (e: KillFxEntry) => boolean
  /** Une arme au sol : celui qui l'a lâchée et celui qui l'a reprise. */
  groundWeapon: (w: Pick<ReplayGroundWeapon, 'dropper' | 'picker' | 't0' | 't1'>) => boolean
  /** Une pose d'équipement : son poseur (celui qui l'a posée ou lâchée), lu à l'image de la pose. */
  placement: (p: Pick<ReplayEquipmentPlacement, 'owner' | 't0'>) => boolean
  /** Un élément d'UN joueur daté à un instant (faille et aller-retour du translocateur, câble de grappin) : son slot à cette image. */
  slotAt: (slot: number, frame: number) => boolean
  /**
   * Une poussée du répulseur : elle est à la personne POUSSÉE et à son AUTEUR (`by`, absent quand le film ne le
   * publie pas), lus à l'image de la poussée — comme un effet de mort est au tueur et à la victime.
   */
  knockback: (k: { slot: number; by: number | null; frame: number }) => boolean
  /** Un élément attribué par le xuid du document : porteur d'objectif, VIP, auteur d'une capture ou d'une explosion. */
  author: (xuid: string) => boolean
  /**
   * La marque d'un capteur sur une vie révélée : elle est au poseur ET au joueur révélé. `slot` est la vie
   * révélée, lue à `frame` ; `owner` le poseur, lu à l'image de sa pose (`ownerFrame`).
   */
  reveal: (r: { owner: number; ownerFrame: number; slot: number }, frame: number) => boolean
  /**
   * Un véhicule à une image : il est à ses OCCUPANTS, partagé entre eux. Seul le garde dès que le joueur y est
   * monté ; Masquer ne le retire que si le joueur en est l'unique occupant (sinon il emporterait un co-occupant
   * que l'utilisateur n'a pas masqué). Un véhicule vide n'est à personne : il reste.
   */
  vehicleAt: (track: ReplayVehicleTrackReady, frame: number) => boolean
  /** Un occupant à une image (son nom, son cône de visée) : à lui seul. */
  occupant: (ride: Pick<ReplayVehicleRide, 'slot' | 'xuid'>, frame: number) => boolean
  /**
   * L'explosion d'un véhicule détruit : un effet, comme une mort. Elle est à ceux qui étaient à bord à cet
   * instant ET au joueur qui l'a détruit.
   */
  vehicleDestruction: (track: ReplayVehicleTrackReady, frame: number) => boolean
}

/**
 * Le slot d'un élément daté À UN INSTANT, ramené à une partie DANS la vie de son auteur. Les éléments
 * lâchés à la mort sont datés une image APRÈS la fin de la vie (`t0 = finVie + 1`) : le slot est alors
 * lu dans la vie qui vient de finir — la même frontière que `ownerAtFrameOrLast`. Un slot négatif (aucun
 * auteur mesuré) ou sans vie avant l'image n'a pas de partie : rien n'est deviné.
 */
function slotParty(
  bySlot: ReturnType<typeof buildLivesBySlot>,
  slot: number,
  frame: number,
): ElementParty | null {
  if (slot < 0) return null
  const life = lifeSlotAt(bySlot.get(slot), frame, Number.POSITIVE_INFINITY)
  return life ? { life } : null
}

/** La partie d'un occupant : sa vie (slot lu à l'image) et le xuid que le document lui donne. */
function occupantParty(
  bySlot: ReturnType<typeof buildLivesBySlot>,
  ride: Pick<ReplayVehicleRide, 'slot' | 'xuid'>,
  frame: number,
): ElementParty | null {
  return partyOf(slotParty(bySlot, ride.slot, frame)?.life, ride.xuid)
}

function present(...parties: Array<ElementParty | null>): ElementParty[] {
  return parties.filter((p): p is ElementParty => p !== null)
}

/**
 * buildFocusFilter rend les prédicats du panneau de joueur, ou `null` quand rien n'est filtré (ni Seul
 * ni Masquer) : les calques gardent alors exactement leur dessin d'avant.
 */
export function buildFocusFilter(
  doc: ReplayDocumentReady,
  focus: PlayerFocus | null,
): FocusFilter | null {
  const killFx = killFxShown(focus)
  const shown = ownershipShown(focus)
  if (!killFx || !shown) return null
  const bySlot = buildLivesBySlot(doc.tracks)
  return {
    killFx,
    groundWeapon: (w) => shown(present(slotParty(bySlot, w.dropper, w.t0), slotParty(bySlot, w.picker, w.t1))),
    placement: (p) => shown(present(slotParty(bySlot, p.owner, p.t0))),
    slotAt: (slot, frame) => shown(present(slotParty(bySlot, slot, frame))),
    author: (xuid) => shown([{ xuid }]),
    knockback: (k) =>
      shown(present(slotParty(bySlot, k.slot, k.frame), k.by === null ? null : slotParty(bySlot, k.by, k.frame))),
    vehicleAt: (track, frame) =>
      shown(present(...vehicleActiveRides(track, frame).map((r) => occupantParty(bySlot, r, frame))), true),
    occupant: (ride, frame) => shown(present(occupantParty(bySlot, ride, frame))),
    vehicleDestruction: (track, frame) =>
      shown(
        present(
          ...vehicleActiveRides(track, frame).map((r) => occupantParty(bySlot, r, frame)),
          partyOf(undefined, track.destruction?.by?.xuid),
          partyOf(undefined, track.destruction?.occupant),
        ),
      ),
    reveal: (r, frame) =>
      shown(present(slotParty(bySlot, r.owner, r.ownerFrame), slotParty(bySlot, r.slot, frame))),
  }
}
