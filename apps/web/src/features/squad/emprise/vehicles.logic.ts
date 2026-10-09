/**
 * vehicles.logic.ts — LES MODÈLES PURS propres à la ressource « véhicules » de l'Emprise (lot L7.4
 * du plan PLAN_EMPRISE_VEHICULES_2026-09-28). Tout le calcul (prises, temps à bord, frags,
 * rendement sur les frags appariés) vient du bloc `squad_emprise` (Go, lot L7.3) ; ce fichier ne
 * fait que nommer les familles. Pur : aucun React, aucune couleur.
 */

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

