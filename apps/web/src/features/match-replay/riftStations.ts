/**
 * riftStations.ts — OÙ SE TROUVE LA FAILLE, ET JUSQU'À QUAND.
 *
 * LE TRANSLOCATEUR EST UN VA-ET-VIENT, pas une balise posée une fois. La balise ÉCHANGE sa
 * position avec le joueur à chaque usage : l'arrivée d'un saut est le départ du précédent, à
 * 0,09 m près sur la mesure (rapport R1 §4.4, sur `1b2d9e08` slot 560). Après un échange, la
 * faille est donc AU POINT DE DÉPART de ce saut — `fx/fy` de l'événement 117, lu dans
 * l'exécutable et validé à 0,00-0,26 m sur 18 événements (rapport R6 §1). Une interface qui
 * dessinerait « une faille posée une fois, fixe » serait FAUSSE.
 *
 * ON NE DESSINE RIEN AVANT LE PREMIER ÉCHANGE, et ce n'est pas une prudence : entre la pose et
 * lui, la position de la balise n'est connue d'AUCUN canal — négatif mesuré sur trois (la
 * faille activée n'est pas une entité répliquée lisible, R1 §1-3 ; la charge du 117 ne porte
 * rien d'autre que {effet, départ, arrivée}, R6 §1.4 ; les 16 poses `translocator_beacon` du
 * parc sont 15 `dropped` + 1 `unknown`, zéro `deployed`).
 *
 * ET RIEN NE RESTE À DEMEURE (point utilisateur du 2026-09-03). La fin de l'équipement a deux
 * modes mesurés — l'épuisement par l'usage final (`spent` quasi simultané) et l'expiration
 * 9 à 16,5 s après le dernier échange — et la mort du porteur clôt aussi, sans `spent`. La
 * dernière station s'arrête donc au premier de ces trois faits, jamais à la fin du rejeu.
 *
 * Pas de React, pas de couleur, pas de canevas : de la géométrie datée, testable seule.
 */
import { trackWindow } from './replayLogic'
import { identityIsUnknown, translocatorRanks } from './placementTeleport'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

/**
 * RiftStation — LA FAILLE À UNE POSITION, pendant un intervalle d'images.
 *
 * Une même vie en produit autant que d'échanges situés : chaque échange déplace la faille au
 * point qu'il quitte, et clôt la station précédente. La dernière porte la fin mesurée.
 */
export interface RiftStation {
  /** Slot de la vie qui porte le translocateur — la faille est à elle. */
  slot: number
  /** Première image où la faille est ici : celle de l'échange qui l'y met. */
  t0: number
  /** Dernière image où elle y est (incluse) : l'échange suivant, ou la fin mesurée. */
  t1: number
  x: number
  y: number
}

/** Un échange SITUÉ d'une vie : l'image, et le point que le joueur quitte. */
interface Echange {
  frame: number
  x: number
  y: number
}

/**
 * riftStations — les stations de toutes les failles du film, dans l'ordre des images.
 *
 * Un balayage par document, jamais par image : les appelants le mémoïsent (même règle que
 * les comptes de poses).
 */
export function riftStations(doc: ReplayDocumentReady): RiftStation[] {
  const ranks = translocatorRanks(doc.abilityLabels)
  const parSlot = new Map<number, Echange[]>()
  for (const t of doc.translocations) {
    // LES POSITIONS SONT SOLIDAIRES au contrat, et sans elles il n'y a rien à situer : le
    // geste reste daté sur la fiche (`translocationMoments`), la carte se tait.
    if (t.fx === undefined || t.fy === undefined) continue
    const liste = parSlot.get(t.slot) ?? []
    liste.push({ frame: t.t, x: t.fx, y: t.fy })
    parSlot.set(t.slot, liste)
  }
  const out: RiftStation[] = []
  for (const [slot, echanges] of parSlot) {
    echanges.sort((a, b) => a.frame - b.frame)
    for (let i = 0; i < echanges.length; i++) {
      const e = echanges[i]
      // CHAQUE STATION EST BORNÉE POUR SON PROPRE COMPTE, et pas seulement la dernière
      // (correction du 2026-09-03, revue ronde 1, constat K1). La borner par le seul échange
      // suivant laissait une station intermédiaire survivre à la fin MESURÉE de son
      // équipement (un `spent` entre deux échanges) et — pire — à la MORT de son porteur,
      // jusque dans la vie du joueur suivant sur le même slot. Le premier des trois faits
      // ferme : l'échange suivant, la consommation, la mort.
      const suivant = i + 1 < echanges.length ? echanges[i + 1].frame - 1 : Number.MAX_SAFE_INTEGER
      const t1 = Math.min(suivant, stationEnd(doc, slot, e.frame, ranks))
      if (t1 < e.frame) continue
      out.push({ slot, t0: e.frame, t1, x: e.x, y: e.y })
    }
  }
  out.sort((a, b) => a.t0 - b.t0)
  return out
}

/**
 * stationEnd — QUAND LA FAILLE POSÉE PAR CET ÉCHANGE S'ÉTEINT, l'échange suivant mis à part.
 *
 * DEUX FAITS MESURÉS, ET LE PREMIER DES DEUX GAGNE :
 *  - la CONSOMMATION de l'équipement (`spent` du calque d'équipement, au rang du
 *    translocateur) — qu'elle soit lue par le balayage strict ou RÉCUPÉRÉE (`recovered`), la
 *    certification vient du témoin de compteur et non du chemin (schéma 38, décision D1) :
 *    une récupérée vaut donc exactement une stricte, et rien ici ne la dévalue ;
 *  - la MORT DU PORTEUR (fin de la vie QUI COUVRE cet échange), qui clôt sans `spent`.
 *
 * ON NE REGARDE QUE LES `spent` À PARTIR DE CET ÉCHANGE : un `spent` antérieur a consommé un
 * équipement que la faille courante ne connaît pas. Et la borne de mort étant celle de la vie
 * qui couvre l'échange, un `spent` d'une vie ULTÉRIEURE du même slot tombe hors fenêtre de
 * lui-même — il ne peut pas fermer une faille qui a déjà disparu avec son porteur.
 *
 * UN `from` SOUS SAUT DE COMPTEUR NE FERME RIEN (`identityIsUnknown`) : il ne dit ni que
 * c'est le translocateur qui s'épuise ni que ce ne l'est pas. On retombe alors sur la mort du
 * porteur — une borne plus tardive, mais MESURÉE, plutôt qu'une fermeture affirmée à tort.
 * C'est le prix, écrit, de la règle P2.4.
 */
function stationEnd(
  doc: ReplayDocumentReady,
  slot: number,
  echange: number,
  ranks: ReadonlySet<number>,
): number {
  let fin = lifeEnd(doc.tracks, slot, echange)
  for (const c of doc.equipmentChanges) {
    if (c.slot !== slot || c.kind !== 'spent') continue
    // À l'image même de l'échange, c'est l'épuisement par l'usage final (mode mesuré n°1) —
    // il ferme. `c.t > fin` écarte à la fois l'après-mort et les `spent` déjà surclassés par
    // un plus précoce : la boucle converge donc vers le PREMIER, quel que soit l'ordre du
    // tableau.
    if (c.t < echange || c.t > fin) continue
    if (identityIsUnknown(c) || !ranks.has(c.from)) continue
    fin = c.t
  }
  return fin
}

/**
 * lifeEnd — la dernière image de la vie qui occupe ce slot à l'instant de cet échange.
 *
 * LA RECHERCHE PORTE SUR LA VIE QUI COUVRE L'ÉCHANGE, pas sur le slot seul : un slot de bipède
 * est réattribué entre les manches, et prendre « la dernière piste de ce slot » ferait survivre
 * une faille à travers la mort de son porteur, jusque dans la vie d'un autre joueur. Sans vie
 * couvrante — cas qui ne devrait pas exister, le serveur ne publiant que les slots à piste — on
 * s'arrête à l'échange lui-même : rien ne se dessine plutôt qu'une faille sans fin.
 */
function lifeEnd(tracks: readonly ReplayTrackReady[], slot: number, frame: number): number {
  for (const t of tracks) {
    if (t.slot !== slot) continue
    const w = trackWindow(t)
    if (frame >= w.start && frame <= w.end) return w.end
  }
  return frame
}

/** riftStationAt — la station active de cette faille à cette image, s'il y en a une. */
export function riftStationsAt(
  stations: readonly RiftStation[],
  frame: number,
): RiftStation[] {
  return stations.filter((s) => frame >= s.t0 && frame <= s.t1)
}
