/**
 * vehicleStrings.ts — les textes de la ressource « véhicules » de l'onglet Emprise (lot L7.4 du
 * plan PLAN_EMPRISE_VEHICULES_2026-09-28). Fichier à part : `empriseStrings.ts` est au seuil de
 * taille. Il fournit l'entrée `resources.vehicle`, l'exposition `aboard_ms` et les mots propres à
 * la ressource (famille hors table). Parité
 * FR / EN garantie par le typage `Record<Locale, …>`. Vocabulaire : une « prise » de véhicule est
 * un véhicule qui passe à une équipe (D2) ; le temps « à bord » est la somme des occupations (D4).
 */
import type { Locale } from '@/lib/i18n/locale'

import type { ExposureText, ResourceText } from './empriseStrings'

export interface VehicleText {
  resource: ResourceText
  /** La barre fine de « Frags obtenus avec les ressources » : le temps à bord. */
  aboard: ExposureText
  /** Famille de châssis hors table. */
  unknown: string
}

/** `duration` : le format « 2 min 39 » de `empriseStrings.ts` (un seul exemplaire). */
export function buildVehicleText(duration: (ms: number) => string): Record<Locale, VehicleText> {
  return {
    fr: {
      resource: {
        label: 'Véhicules',
        pisteTitle: 'Prises de véhicules',
        gridTitle: 'Prises de véhicules',
        footer: 'véhicules',
        absent: 'Aucun véhicule pris.',
        itemAbsent: ' : aucune prise sur cette carte.',
        productionTitle: 'Frags depuis un véhicule',
        yieldSub: 'frags par minute à bord',
      },
      aboard: { name: 'temps à bord', fmt: duration },
      unknown: 'Véhicule inconnu',
    },
    en: {
      resource: {
        label: 'Vehicles',
        pisteTitle: 'Vehicle takes',
        gridTitle: 'Vehicle takes',
        footer: 'vehicles',
        absent: 'No vehicle taken.',
        itemAbsent: ': not taken on this map.',
        productionTitle: 'Kills from a vehicle',
        yieldSub: 'kills per minute aboard',
      },
      aboard: { name: 'time aboard', fmt: duration },
      unknown: 'Unknown vehicle',
    },
  }
}
