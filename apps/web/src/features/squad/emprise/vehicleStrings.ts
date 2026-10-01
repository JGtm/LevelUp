/**
 * vehicleStrings.ts — les textes de la ressource « véhicules » de l'onglet Emprise (lot L7.4 du
 * plan PLAN_EMPRISE_VEHICULES_2026-09-28). Fichier à part : `empriseStrings.ts` est au seuil de
 * taille. Il fournit l'entrée `resources.vehicle`, l'exposition `aboard_ms` et les mots propres à
 * la ressource (famille inconnue, case « non mesuré », note de couverture du rendement). Parité
 * FR / EN garantie par le typage `Record<Locale, …>`. Vocabulaire : une « prise » de véhicule est
 * un véhicule qui passe à un camp (D2) ; le temps « à bord » est la somme des occupations (D4).
 */
import type { Locale } from '@/lib/i18n/locale'

import type { ExposureText, ResourceText } from './empriseStrings'

export interface VehicleText {
  resource: ResourceText
  /** La barre fine de « Frags obtenus avec les ressources » : le temps à bord. */
  aboard: ExposureText
  /** Famille de châssis hors table. */
  unknown: string
  /** Case de la grille d'un match dont les véhicules ne sont pas mesurés (D8), et son infobulle. */
  unmeasuredCell: string
  unmeasuredTip: string
  /** Note sous « Rendement face à l'adversaire » : part des frags appariés, épisodes sans nom. */
  pairedNote: (paired: number, total: number, pct: string) => string
  unnamedNote: (n: number) => string
}

/** `duration` : le format « 2 min 39 » de `empriseStrings.ts` (un seul exemplaire). */
export function buildVehicleText(duration: (ms: number) => string): Record<Locale, VehicleText> {
  return {
    fr: {
      resource: {
        label: 'Véhicules',
        pisteSub: 'prises',
        gridSub: 'prises',
        footer: 'véhicules',
        absent: 'Aucun véhicule pris.',
        itemAbsent: ' : aucune prise sur cette carte.',
        productionSub: 'frags depuis un véhicule',
        yieldSub: 'frags par minute à bord',
      },
      aboard: { name: 'temps à bord', fmt: duration },
      unknown: 'Véhicule inconnu',
      unmeasuredCell: 'non mesuré',
      unmeasuredTip:
        'Véhicules non mesurés sur ce match : l’occupation n’a pas été lue (film décodé avant ce relevé, ou camp inconnu).',
      pairedNote: (paired, total, pct) =>
        `Véhicules : le rendement ne compte que les ${paired} frags sur ${total} (${pct}) tombés pendant un ` +
        'passage daté de leur tueur ; les autres sont dans la barre de la carte voisine.',
      unnamedNote: (n) => `${n} passage${n > 1 ? 's' : ''} sans joueur nommé (robots) ne ${n > 1 ? 'comptent' : 'compte'} pas.`,
    },
    en: {
      resource: {
        label: 'Vehicles',
        pisteSub: 'takes',
        gridSub: 'takes',
        footer: 'vehicles',
        absent: 'No vehicle taken.',
        itemAbsent: ': not taken on this map.',
        productionSub: 'kills from a vehicle',
        yieldSub: 'kills per minute aboard',
      },
      aboard: { name: 'time aboard', fmt: duration },
      unknown: 'Unknown vehicle',
      unmeasuredCell: 'not measured',
      unmeasuredTip:
        'Vehicles not measured for this match: occupancy was not read (film decoded before this reading, or side unknown).',
      pairedNote: (paired, total, pct) =>
        `Vehicles: the rate only counts the ${paired} of ${total} kills (${pct}) made during a dated ` +
        'ride of their killer; the others are in the neighbouring card’s bar.',
      unnamedNote: (n) => `${n} ride${n > 1 ? 's' : ''} with no named player (bots) ${n > 1 ? 'are' : 'is'} not counted.`,
    },
  }
}
