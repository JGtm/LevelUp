/**
 * vehicles.logic.ts — LES MODÈLES PURS propres à la ressource « véhicules » de l'Emprise (lot L7.4
 * du plan PLAN_EMPRISE_VEHICULES_2026-09-28). Tout le calcul (prises, temps à bord, frags,
 * rendement sur les frags appariés) vient du bloc `squad_emprise` (Go, lot L7.3) ; ce fichier ne
 * fait que nommer les familles et lire la couverture. Pur : aucun React, aucune couleur.
 */
import type { SquadEmpriseBlock } from '@/lib/api/types'

import { RESOURCE_VEHICLE } from './emprise.logic'

/** Clé publiée pour un châssis hors table (contrat Go `replay.VehicleFamilyUnknown`). */
export const VEHICLE_FAMILY_UNKNOWN = 'unknown'

/**
 * Le nom d'une famille de véhicule : le libellé du titre quand le manifeste qualifie la famille
 * (une tourelle fixe), « Véhicule inconnu » pour un châssis hors table, sinon le nom propre du jeu
 * tiré de la clé (`warthog` → « Warthog », `rocket_hog` → « Rocket Hog »).
 */
export function vehicleFamilyName(key: string, label: string | undefined, unknownLabel: string): string {
  if (label) return label
  if (key === VEHICLE_FAMILY_UNKNOWN || key === '') return unknownLabel
  return key
    .split('_')
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

/** La couverture du rendement des véhicules, pour la note de « Rendement face à l'adversaire ». */
export interface VehicleCoverage {
  /** Frags de classe véhicule des matchs du rendement, et ceux tombés pendant un épisode daté. */
  fragsTotal: number
  fragsPaired: number
  /** Part appariée, 0..1 (calcul Go). */
  pairedShare: number
  /** Épisodes sans joueur nommé (robots) : ni prise ni temps (D10). */
  episodesUnnamed: number
}

/**
 * buildVehicleCoverage — la couverture à écrire sous le rendement des véhicules, ou null : sans
 * lecture, en échec, ou sans frag d'engin à apparier, rien à dire.
 */
export function buildVehicleCoverage(block: SquadEmpriseBlock): VehicleCoverage | null {
  const v = block.vehicles
  if (!v || v.unavailable) return null
  const hasYield = (block.production ?? []).some((p) => p.resource === RESOURCE_VEHICLE && p.relative_gap != null)
  if (!hasYield || v.frags_total <= 0 || v.paired_share == null) return null
  return {
    fragsTotal: v.frags_total,
    fragsPaired: v.frags_paired,
    pairedShare: v.paired_share,
    episodesUnnamed: v.episodes_unnamed,
  }
}
