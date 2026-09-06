/**
 * carrierPosition.ts — OÙ SE DESSINE LE GLYPHE D'UN PORTEUR D'OBJECTIF, et rien d'autre.
 *
 * LE FAIT, MESURÉ CÔTÉ SERVEUR. Un joueur attaché à un véhicule CESSE DE RÉPLIQUER SA POSITION
 * MONDE (précondition écrite de `replay/vehicle_rides.go:12-15`) : sa trajectoire de bipède n'a
 * plus d'échantillon entre l'embarquement et la descente. `positionAt` (replayLogic.ts) fait
 * alors ce qu'on lui demande — elle INTERPOLE LINÉAIREMENT entre le dernier point avant
 * l'embarquement et le premier point après la descente. Le pion, lui, est déjà supprimé pendant
 * l'épisode (`replayMarkers.ts`, `MarkerStyle.embarkedAtSlot`) ; le GLYPHE porté, non : bombe,
 * crâne, drapeau porté et couronne VIP traversaient donc le décor en ligne droite, seuls.
 *
 * LA DÉCISION PRODUIT (utilisateur, 2026-09-05, « option 1 ») : pendant l'épisode, le glyphe se
 * dessine SUR LE VÉHICULE — il suit la trajectoire de la monture, à la MÊME interpolation que le
 * sprite (`vehiclePositionAt`, l'unique écriture de la position d'un véhicule à une image), pour
 * que le glyphe et le véhicule coïncident à l'écran au pixel près.
 *
 * LA CLÉ EST LE XUID, parce que c'est celle que les calques d'objectif tiennent déjà : une
 * période de portage nomme son porteur par son xuid, jamais par un slot de bipède. Et l'épisode
 * d'occupation le nomme de la même façon (`VehicleRide.xuid`) — c'est même la SOURCE PRIORITAIRE
 * de son identité pour la teinte et le nom du véhicule, pour la raison exacte qui nous occupe
 * ici : pendant l'épisode, le pont slot->joueur est muet. UN ÉPISODE SANS XUID NE DÉPLACE AUCUN
 * GLYPHE : le repli est la position de bipède, c'est-à-dire le comportement d'avant ce lot —
 * jamais une position inventée, jamais un second pont d'identité à entretenir.
 *
 * INDÉPENDANT DE LA BASCULE DU CALQUE DES VÉHICULES (contrat, et c'est une décision) : le glyphe
 * suit le véhicule même quand l'utilisateur a éteint ce calque. La position d'un porteur est un
 * FAIT DU DOCUMENT, pas une décoration du calque ; la bascule ne commande que ce qui se DESSINE.
 * Elle diverge donc volontairement de `Vehicles.isEmbarkedAt` — lui suit la bascule, parce que
 * cacher un pion sans montrer son véhicule ferait disparaître un joueur sans réglage pour le
 * récupérer (revue adversariale du 2026-09-02, point 7). Ici l'arbitrage est inverse : calque
 * éteint, le pion revient à SA position interpolée (fausse, mais c'est le régime assumé de la
 * bascule) tandis que le glyphe garde la seule position vraie dont on dispose.
 *
 * UN SEUL CHEMIN POUR CINQ CALQUES. Les cinq consommateurs (bombe portée, couronne VIP, crâne
 * d'Oddball, drapeau porté, déflagration d'Assaut) appellent `useCarrierPosAt` et rien d'autre :
 * la règle « embarqué -> véhicule, sinon bipède » est écrite ICI, une fois
 * (`positionOfCarrierAt`), et `carrierPosition.guard.test.ts` interdit qu'elle se recopie.
 */
import { useMemo } from 'react'

import { buildPlayerPosAt, type PlayerPosAt } from './livesPosition'
import type { XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { vehicleCanEmbark, vehiclePositionAt } from './vehiclesLayer'

/** La position d'un joueur EMBARQUÉ à une image, ou `null` (il ne l'est pas, ou on l'ignore). */
export type EmbarkedPosAt = (xuid: string, frame: number) => XY | null

/** Un épisode d'occupation réduit à ce dont la position a besoin : sa fenêtre et sa monture. */
interface EmbarkedEpisode {
  t0: number
  t1: number
  vehicle: ReplayVehicleTrackReady
}

/**
 * buildEmbarkedPosAt — l'index des épisodes d'occupation PAR XUID, puis la position de la monture.
 *
 * Même construction et même filtre que `buildEmbarkedPredicate` (vehiclesLayer.ts) : les épisodes
 * de TOUS les véhicules sont regroupés UNE fois, et seules les vies qui passent `vehicleCanEmbark`
 * y entrent — ni décor, ni châssis non résolu. Le filtre doit rester le MÊME que celui du prédicat
 * du pion, sans quoi un faux épisode déplacerait un glyphe là où il ne cache aucun pion (les trois
 * épisodes prêtés au prop Falcon de l'artefact `0d76e8f1`).
 *
 * Le prédicat, lui, indexe par SLOT : c'est ce que le calque des pions tient. Ici c'est le xuid —
 * deux clés pour deux appelants, pas deux règles d'occupation.
 */
export function buildEmbarkedPosAt(
  vehicles: readonly ReplayVehicleTrackReady[],
): EmbarkedPosAt {
  const byXuid = new Map<string, EmbarkedEpisode[]>()
  for (const vehicle of vehicles) {
    if (!vehicleCanEmbark(vehicle)) continue
    for (const ride of vehicle.rides) {
      if (!ride.xuid) continue
      const episode: EmbarkedEpisode = { t0: ride.t0, t1: ride.t1, vehicle }
      const list = byXuid.get(ride.xuid)
      if (list) list.push(episode)
      else byXuid.set(ride.xuid, [episode])
    }
  }
  if (byXuid.size === 0) return () => null
  return (xuid, frame) => {
    const list = byXuid.get(xuid)
    if (!list) return null
    for (const episode of list) {
      if (frame < episode.t0 || frame > episode.t1) continue
      const at = vehiclePositionAt(episode.vehicle, frame)
      if (at) return at
    }
    return null
  }
}

/**
 * positionOfCarrierAt — LA RÈGLE, en une seule écriture : embarqué -> la position du véhicule ;
 * sinon -> la position du bipède, telle que les calques la lisaient jusqu'ici.
 *
 * Le repli couvre trois cas, tous « comportement d'avant ce lot » : le joueur n'est pas à bord,
 * l'épisode ne le nomme pas, ou la monture n'a AUCUNE position à cette image (ni échantillon ni
 * naissance lue — `vehiclePositionAt` refuse d'en inventer une).
 */
export function positionOfCarrierAt(
  embarkedAt: EmbarkedPosAt,
  bipedAt: PlayerPosAt,
  xuid: string,
  frame: number,
): XY | null {
  return embarkedAt(xuid, frame) ?? bipedAt(xuid, frame)
}

/** buildCarrierPosAt — la relecture complète d'un document, pour les constructeurs hors React. */
export function buildCarrierPosAt(doc: ReplayDocumentReady): PlayerPosAt {
  const embarkedAt = buildEmbarkedPosAt(doc.vehicles)
  const bipedAt = buildPlayerPosAt(doc)
  return (xuid, frame) => positionOfCarrierAt(embarkedAt, bipedAt, xuid, frame)
}

/** useCarrierPosAt — la même relecture, mémoïsée pour les hooks de calque. */
export function useCarrierPosAt(doc: ReplayDocumentReady): PlayerPosAt {
  return useMemo(() => buildCarrierPosAt(doc), [doc])
}
