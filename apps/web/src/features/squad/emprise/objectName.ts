/**
 * objectName.ts — LE NOM D'UN OBJET DE L'EMPRISE, une seule source pour l'Escouade, les Séries
 * temporelles et la page Sessions (garde-rail `objectName.guard.test.ts`, CLAUDE.md n° 6).
 *
 * Un bonus est nommé par le web (famille du résumé d'usage), une arme par le titre (libellé publié
 * par le Go, sa clé à défaut), un véhicule par son libellé de titre ou le nom propre tiré de sa clé.
 * Pur : les textes sont fournis par l'appelant.
 */
import { equipmentFamilyLabel, type UsageText } from '@/features/_shared/usage/usageI18n'
import type { SquadEmpriseObject } from '@/lib/api/types'

import { RESOURCE_POWERUP, RESOURCE_VEHICLE } from './emprise.logic'
import { vehicleFamilyName } from './vehicles.logic'

/** Les deux familles de bonus : nommées par l'i18n d'usage quelle que soit la ressource qui les porte. */
const POWERUP_FAMILIES: ReadonlySet<string> = new Set(['powerup_camo', 'powerup_overshield'])

export function empriseObjectName(o: SquadEmpriseObject, usageText: UsageText, unknownVehicle: string): string {
  // Un bonus se nomme par sa famille même si le serveur l'a rangé ailleurs (socle mal classé) :
  // jamais sa clé technique à l'écran.
  if (o.resource === RESOURCE_POWERUP || POWERUP_FAMILIES.has(o.key)) return equipmentFamilyLabel(o.key, usageText)
  if (o.resource === RESOURCE_VEHICLE) return vehicleFamilyName(o.key, o.label, unknownVehicle)
  return o.label || o.key
}
