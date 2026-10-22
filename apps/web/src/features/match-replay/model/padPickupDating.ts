/**
 * padPickupDating.ts — LE RAMASSAGE D'UN SOCLE, DATÉ PAR L'ÉVÉNEMENT NATIF QUAND LE FILM LE DIT.
 *
 * CORRIGÉ LE 2026-10-22 (lot detonations, item 2, suite du contrôle n° 21 des contrôles v8 : un
 * socle de power-up vu « incertain » alors qu'un joueur venait tout juste de passer). Le serveur
 * publie `padPickups[].t`/`.xuid` — l'instant exact et le ramasseur — pour toute occupation que le
 * ramassage natif (`biped_pickup`) date sans ambiguïté, ARME COMME POWER-UP depuis le même lot
 * (`pad_pickup_dating.go`, jointure par le slug canonique côté power-up). Ce module applique CETTE
 * date à la fenêtre `[tLow, tHigh]` de l'occupation qu'elle désigne — avant même le raffinement par
 * proximité (`padPresenceRefine.ts`), qui reste le repli des occupations qu'aucun ramassage natif
 * ne couvre : LA GRAMMAIRE PRIME SUR L'HEURISTIQUE.
 *
 * LA FENÊTRE COLLAPSE, ELLE NE DISPARAÎT PAS : `tLow` ET `tHigh` prennent la valeur datée, séparés
 * de [PAD_BETWEEN_FRAMES] pour ne jamais atteindre le sentinel « jamais vidé » de `padStateAt`
 * (`tHigh <= tLow`) — même garantie, même raison que `padPresenceRefine.ts`. L'état passe donc de
 * PLEIN à VIDE en une fraction de frame (au pire une image peinte « incertain », invisible en
 * pratique), là où il restait incertain jusqu'à vingt secondes — et `padStateAt`/`padRespawnAt`
 * n'ont besoin d'aucun autre changement : ils lisent déjà `tLow`/`tHigh` sans savoir d'où ils
 * viennent.
 *
 * CE QUE CE MODULE NE FAIT PAS : dater une occupation que le serveur n'a pas datée. Une occupation
 * sans `padPickups[].t` correspondant traverse INCHANGÉE — elle reste au raffinement par
 * proximité, ou à l'intervalle brut si rien ne l'approche.
 *
 * LE RAMASSEUR (`pickupXuid`) EST CAPTURÉ, SANS LECTEUR POUR L'INSTANT. L'infobulle du socle
 * (`ReplayWeaponPadTip.tsx`) garde sa décision « deux lignes au plus » (retour utilisateur du
 * 2026-08-28) et l'ancienne surface qui devait le montrer (le tableau « Contrôle des armes
 * spéciales ») a été retirée le 2026-10-07 : aucune page ne nomme donc encore ce ramasseur. Le
 * champ reste sur l'occupation, testé, pour que ce ne soit pas à refaire le jour où une surface
 * l'affichera — pas un calcul mort, une donnée mesurée sans lecteur.
 *
 * Pur : aucune dépendance React, aucun canvas.
 */
import type { ReplayDocumentReady, ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'
import { PAD_BETWEEN_FRAMES } from './padPresenceRefine'

/** Une occupation dont l'instant vient d'un ramassage natif porte en plus son ramasseur. */
export type DatedPadPresence = ReplayWeaponPadReady['presence'][number] & {
  /** XUID du ramasseur, décimal — même forme que Track.xuid. Absent si le ramassage natif
   *  n'a pas nommé de joueur (vie non pontée), présent même sans joueur résolu sinon jamais. */
  pickupXuid?: string
}

/** clePickup — la clé de jointure PadPickup <-> occupation : (index de socle, tLow, tHigh). */
function clePickup(pad: number, tLow: number, tHigh: number): string {
  return `${pad}|${tLow}|${tHigh}`
}

/**
 * applyPadPickupDating — les mêmes socles, dont les occupations que `padPickups` date portent
 * désormais l'instant EXACT (fenêtre collapsée) et le ramasseur.
 *
 * RÉFÉRENCE PRÉSERVÉE QUAND RIEN NE CHANGE, même convention que `refinePadPresence` : un socle
 * sans occupation datée est rendu TEL QUEL.
 */
export function applyPadPickupDating(
  pads: readonly ReplayWeaponPadReady[],
  padPickups: ReplayDocumentReady['padPickups'],
): readonly ReplayWeaponPadReady[] {
  if (!padPickups || padPickups.length === 0) return pads
  const parCle = new Map<string, { t: number; xuid?: string }>()
  for (const p of padPickups) {
    if (p.t === undefined || p.t === null) continue
    parCle.set(clePickup(p.pad, p.tLow, p.tHigh), { t: p.t, xuid: p.xuid ?? undefined })
  }
  if (parCle.size === 0) return pads
  let touche = false
  const out = pads.map((pad, i) => {
    let touchePad = false
    const presence = pad.presence.map((occ) => {
      const date = parCle.get(clePickup(i, occ.tLow, occ.tHigh))
      if (!date) return occ
      touchePad = true
      const datee: DatedPadPresence = { ...occ, tLow: date.t, tHigh: date.t + PAD_BETWEEN_FRAMES }
      if (date.xuid !== undefined) datee.pickupXuid = date.xuid
      return datee
    })
    if (!touchePad) return pad
    touche = true
    return { ...pad, presence }
  })
  return touche ? out : pads
}
